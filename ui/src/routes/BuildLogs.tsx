import { useParams } from '@tanstack/react-router'
import { useEffect, useMemo, useRef, useState, useLayoutEffect } from 'react'
import { useVirtualizer } from '@tanstack/react-virtual'
import { Radio, Search, ChevronRight, Download, Copy, Check } from 'lucide-react'
import { ApiError } from '../api/client'
import { fetchEventsBefore, useEvents, useEventsSummary, useLiveEvents } from '../api/hooks'
import { useTimeFormat } from '../lib/time'
import type { LFEvent } from '../api/types'
import { EmptyState } from '../components/EmptyState'
import { Badge, Button, Select, cn } from '../ui'

// logsToText renders the full journal as a plain-text log for export: one line
// per event (ISO timestamp, host-or-"build", kind, message), with any longer
// payload detail (a step's command output / error) indented beneath it.
function logsToText(events: LFEvent[]): string {
  return events
    .map((e) => {
      const ts = new Date(e.created_at).toISOString()
      const who = e.host ? (e.team_number != null ? `t${e.team_number}/${e.host}` : e.host) : 'build'
      let line = `${ts}  ${who}  ${e.kind}  ${e.message}`
      const detail = eventDetail(e.payload)
      if (detail && detail !== e.message) {
        line += '\n' + detail.split('\n').map((l) => '    ' + l).join('\n')
      }
      return line
    })
    .join('\n')
}

// downloadText triggers a browser download of text as a file -- no server round
// trip, since the journal is already fully loaded in the page.
function downloadText(filename: string, text: string) {
  const url = URL.createObjectURL(new Blob([text], { type: 'text/plain;charset=utf-8' }))
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  a.remove()
  URL.revokeObjectURL(url)
}

// toneForKind colours an event's kind badge by what it IS, so the journal is
// scannable by type/outcome: red for failures, green for completions, amber for
// tolerated/skipped, and a distinct hue per subsystem (access, network, steps,
// deploy, agent). Matched loosely on the kind string so new kinds still land
// somewhere sensible.
function toneForKind(kind: string): 'danger' | 'success' | 'warning' | 'info' | 'purple' | 'sky' | 'indigo' | 'ink' | 'neutral' {
  const k = kind.toLowerCase()
  if (k.includes('fail') || k.includes('error') || k.includes('invalid')) return 'danger'
  if (k.includes('ignored') || k.includes('retry') || k.includes('skip') || k.includes('note') || k.includes('warn')) return 'warning'
  if (k.includes('done') || k.includes('finish') || k.includes('complete') || k.includes('succeed') || k.includes('deployed') || k.includes('opened') || k.includes('configured') || k.includes('materialized')) return 'success'
  if (k.startsWith('access')) return 'purple'
  if (k.startsWith('network')) return 'sky'
  if (k.includes('step')) return 'info'
  if (k.includes('deploy') || k.includes('destroy') || k.includes('teardown') || k.includes('build')) return 'indigo'
  if (k.includes('heartbeat') || k.includes('session')) return 'ink'
  return 'neutral'
}

// eventDetail extracts the meaningful extra text an event carries BEYOND its
// one-line message, so a row is only expandable when there is genuinely more to
// read. Three payload shapes occur in practice:
//   - a non-empty `detail` string: step_failed/step_retry store the full command error
//     there (step_started/step_done carry an empty one -> nothing to show);
//   - a non-empty object: builder.* events carry the whole request (os, size,
//     ports, cidr, external_name...) -> render it as aligned "key: value" lines,
//     which is the real detail, NOT a repeat of the message;
//   - an empty object ({}): task.*, *.configured, *.reconciled -> nothing.
// Returning '' means "no detail": the caller renders a plain, non-expandable row.
function eventDetail(payload: unknown): string {
  if (!payload || typeof payload !== 'object') return ''
  const obj = payload as Record<string, unknown>
  if (typeof obj.detail === 'string' && obj.detail.trim() !== '') return obj.detail.trim()
  const lines = Object.keys(obj)
    .filter((k) => k !== 'detail')
    .map((k) => {
      const v = obj[k]
      if (v == null || v === '') return ''
      const val = typeof v === 'object' ? JSON.stringify(v, null, 2) : String(v)
      return `${k}: ${val}`
    })
    .filter(Boolean)
  return lines.join('\n')
}

// "Build → Logs: the event journal as a hierarchy... Live over SSE while
// anything is running." Live streams from the real SSE
// endpoint (internal/api/live.go) on top of the initial history from GET
// /builds/{id}/events. Each event now carries the host it's about
// (resolved server-side -- see builds_ui.go's eventView) so the journal
// shows "which host" per line, and a search box plus a kind filter narrow
// it -- everything an agent does (step.started/completed/failed, heartbeat
// lifecycle) lands here alongside the deploy/access lifecycle.
export function BuildLogs() {
  const { buildId } = useParams({ from: '/repos/$repoId/builds/$buildId/logs' })
  const { data: history, error: historyError, isLoading: historyLoading } = useEvents(buildId)
  const { data: summary } = useEventsSummary(buildId)
  // Earlier pages, fetched on demand ("Load earlier"): the api serves the
  // newest 5,000 at first; a long build's journal can be far larger.
  const [older, setOlder] = useState<LFEvent[]>([])
  const [loadingOlder, setLoadingOlder] = useState(false)
  const [olderError, setOlderError] = useState<string | null>(null)
  const [exhausted, setExhausted] = useState(false)
  const [live, setLive] = useState<LFEvent[]>([])
  // Batch incoming live events instead of a setState per event. A busy build
  // streams events faster than React can repaint; appending one at a time meant
  // a full re-render (and an O(n) recompute of all/kinds/shown) on every event --
  // the main cause of the "page is slowing down" warning. Buffer in a ref and
  // flush on a short interval, so N events become one re-render.
  const liveBuf = useRef<LFEvent[]>([])
  const { connected } = useLiveEvents(buildId, (ev) => {
    liveBuf.current.push(ev)
  })
  useEffect(() => {
    const t = setInterval(() => {
      if (liveBuf.current.length === 0) return
      const batch = liveBuf.current
      liveBuf.current = []
      setLive((prev) => [...prev, ...batch])
    }, 400)
    return () => clearInterval(t)
  }, [])
  const [query, setQuery] = useState('')
  const [kind, setKind] = useState('')
  const [expanded, setExpanded] = useState<Set<string>>(new Set())
  const [copied, setCopied] = useState(false)
  const fmt = useTimeFormat()

  function toggle(id: string) {
    setExpanded((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  // The complete journal for Export/Copy: the full history + live, PLUS any live
  // events still buffered (not yet flushed into `all`), so a download never
  // misses the last ~400ms of an actively-streaming build. Not the capped render,
  // and not the search/kind filter -- always everything.
  function fullLog(): LFEvent[] {
    const seen = new Set(all.map((e) => e.id))
    return [...all, ...liveBuf.current.filter((e) => !seen.has(e.id))]
  }

  // Export the whole journal (not the filtered view) as a timestamped .txt.
  function exportLogs() {
    const stamp = new Date().toISOString().slice(0, 19).replace(/[:T]/g, '-')
    downloadText(`laforge-build-${buildId.slice(0, 8)}-logs-${stamp}.txt`, logsToText(fullLog()))
  }

  // Copy the whole journal to the clipboard, with a brief "Copied" confirmation.
  async function copyLogs() {
    try {
      await navigator.clipboard.writeText(logsToText(fullLog()))
      setCopied(true)
      setTimeout(() => setCopied(false), 1500)
    } catch {
      // Clipboard blocked (insecure context / permission) -- fall back to export.
      exportLogs()
    }
  }

  const all = useMemo(() => {
    const seen = new Set([...older, ...(history ?? [])].map((e) => e.id))
    return [...older, ...(history ?? []), ...live.filter((e) => !seen.has(e.id))]
  }, [older, history, live])

  async function loadEarlier() {
    const oldest = all[0]
    if (!oldest || loadingOlder) return
    setLoadingOlder(true)
    setOlderError(null)
    try {
      const page = await fetchEventsBefore(buildId, oldest.created_at)
      if (page.length === 0) setExhausted(true)
      setOlder((prev) => [...page, ...prev])
    } catch (e) {
      setOlderError(e instanceof ApiError ? e.message : 'Could not load earlier events')
    } finally {
      setLoadingOlder(false)
    }
  }
  const total = summary?.total
  const loaded = (older.length + (history?.length ?? 0))
  const moreAvailable = !exhausted && total != null && loaded < total

  // Every distinct kind actually present, for the kind filter -- so it only
  // ever offers values that exist in this build's journal.
  const kinds = useMemo(() => Array.from(new Set(all.map((e) => e.kind))).sort(), [all])

  const shown = useMemo(() => {
    const q = query.trim().toLowerCase()
    return all.filter((e) => {
      if (kind && e.kind !== kind) return false
      if (!q) return true
      // Match the host both bare ("bff-dc-01") and as the team-qualified label
      // the row actually shows ("t1/bff-dc-01") -- searching the displayed form
      // returned nothing before, since no single field held the combined string.
      const qualified = e.host && e.team_number != null ? `t${e.team_number}/${e.host}` : ''
      return [e.message, e.kind, e.host ?? '', qualified, e.team_number != null ? `team ${e.team_number}` : '']
        .some((v) => v.toLowerCase().includes(q))
    }).reverse() // newest first -- the journal reads latest -> oldest (filter() returns a fresh array, so reversing it never touches `all`)
  }, [all, query, kind])

  // Virtualize the list: only the rows in (and near) the viewport are in the DOM,
  // so the whole journal -- tens of thousands of events on a big DC build -- stays
  // fully scrollable without locking the main thread. Rows are variable height
  // (they expand on click), so the virtualizer measures each rendered row and
  // re-measures on expand/collapse (measureElement's ResizeObserver).
  const parentRef = useRef<HTMLDivElement>(null)
  const rowVirtualizer = useVirtualizer({
    count: shown.length,
    getScrollElement: () => parentRef.current,
    estimateSize: () => 29, // a collapsed row (py-1.5 line + border); refined by measurement
    overscan: 16,
    getItemKey: (index) => shown[index].id,
  })

  // Tail-follow: the newest event is at the TOP now, so keep the top pinned as the
  // build streams -- but only while the user is already at the top. If they've
  // scrolled down to read older history, leave them there. atTopRef is updated on
  // scroll (the last known position, before new rows pushed the content down).
  const atTopRef = useRef(true)
  function onScroll() {
    const el = parentRef.current
    if (!el) return
    atTopRef.current = el.scrollTop < 60
  }
  useEffect(() => {
    if (atTopRef.current && shown.length > 0) {
      rowVirtualizer.scrollToIndex(0, { align: 'start' })
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [shown.length])

  return (
    <div className="flex h-full flex-col">
      <div className="sticky top-0 z-20 mb-3 flex shrink-0 flex-wrap items-center gap-2 bg-surface pb-3 pt-1">
        <div className="flex items-center gap-2 text-xs text-fg-muted">
          <Radio size={12} className={connected ? 'text-success' : 'text-fg-subtle'} />
          {connected ? 'Live' : 'Reconnecting…'}
        </div>
        <div className="ml-auto flex items-center gap-2">
          <div className="relative">
            <Search size={13} className="pointer-events-none absolute left-2 top-1/2 -translate-y-1/2 text-fg-subtle" />
            <input
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="Search logs…"
              className="h-8 w-56 rounded-token border border-border bg-surface-raised pl-7 pr-2 text-sm text-fg placeholder:text-fg-subtle outline-none focus-visible:border-accent focus-visible:outline-2 focus-visible:outline-offset-0 focus-visible:outline-accent-soft"
            />
          </div>
          <Select value={kind} onChange={(e) => setKind(e.target.value)} className="h-8 w-40">
            <option value="">Filter By Type</option>
            {kinds.map((k) => (
              <option key={k} value={k}>
                {k}
              </option>
            ))}
          </Select>
          <Button variant="primary" size="sm" onClick={copyLogs} disabled={all.length === 0} title="Copy the full log to the clipboard">
            {copied ? <Check size={13} /> : <Copy size={13} />} {copied ? 'Copied' : 'Copy'}
          </Button>
          <Button variant="primary" size="sm" onClick={exportLogs} disabled={all.length === 0} title="Export the full log as a text file">
            <Download size={13} /> Export
          </Button>
        </div>
      </div>

      {historyError ? (
        <EmptyState
          icon={Radio}
          title="The log couldn’t be loaded"
          hint={`${historyError instanceof ApiError ? historyError.message : 'The request failed.'} Live events still stream in below as they happen; reload the page to try again.`}
        />
      ) : historyLoading && all.length === 0 ? (
        <EmptyState icon={Radio} title="Loading the log…" hint="Fetching the newest events." />
      ) : all.length === 0 ? (
        <EmptyState icon={Radio} title="No events yet" hint="Events appear here as the build progresses." />
      ) : shown.length === 0 ? (
        <EmptyState icon={Search} title="No events match" hint="Clear the search or kind filter to see the full journal." />
      ) : (
        // The scroll container the virtualizer measures against. Only the rows in
        // view are in the DOM; the sized inner div gives the scrollbar the full
        // height so scrolling reaches every event.
        <div ref={parentRef} onScroll={onScroll} className="min-h-0 flex-1 overflow-auto font-mono text-xs">
          <div style={{ height: `${rowVirtualizer.getTotalSize()}px`, position: 'relative', width: '100%' }}>
            {rowVirtualizer.getVirtualItems().map((vi) => {
              const ev = shown[vi.index]
              return (
                <div
                  key={vi.key}
                  data-index={vi.index}
                  ref={rowVirtualizer.measureElement}
                  className="absolute left-0 top-0 w-full border-b border-border"
                  style={{ transform: `translateY(${vi.start}px)` }}
                >
                  <LogRow ev={ev} open={expanded.has(ev.id)} onToggle={() => toggle(ev.id)} timeLabel={fmt.time(ev.created_at)} />
                </div>
              )
            })}
          </div>
        </div>
      )}
      {!historyError && all.length > 0 && (total != null || moreAvailable) && (
        <div className="flex shrink-0 items-center gap-3 border-t border-border pt-2 text-xs text-fg-muted">
          <span>
            {total != null && loaded < total ? `Showing the newest ${loaded.toLocaleString()} of ${total.toLocaleString()} events` : `${loaded.toLocaleString()} events`}
            {query || kind ? ` (${shown.length.toLocaleString()} match the filter)` : ''}
          </span>
          {moreAvailable && (
            <Button variant="secondary" size="sm" onClick={loadEarlier} disabled={loadingOlder}>
              {loadingOlder ? 'Loading…' : 'Load earlier'}
            </Button>
          )}
          {olderError && <span className="text-danger">{olderError}</span>}
        </div>
      )}
    </div>
  )
}

// LogRow is one event: one line by default. A row can be opened when there's
// more to read than fits -- its message is cut off at the window's edge, or
// it carries detail (a step's error, a builder request) -- and opening it
// shows the whole message wrapped, then the detail. A short row with nothing
// more to show has no chevron and doesn't open.
function LogRow({ ev, open, onToggle, timeLabel }: { ev: LFEvent; open: boolean; onToggle: () => void; timeLabel: string }) {
  const detail = eventDetail(ev.payload)
  const msgRef = useRef<HTMLSpanElement>(null)
  const [overflows, setOverflows] = useState(false)
  // Measured, not guessed: whether the one-line message is cut off depends on
  // the window. Re-measured when the window resizes.
  useLayoutEffect(() => {
    const el = msgRef.current
    if (!el) return
    const measure = () => setOverflows(el.scrollWidth > el.clientWidth + 1)
    measure()
    const ro = typeof ResizeObserver !== 'undefined' ? new ResizeObserver(measure) : null
    ro?.observe(el)
    return () => ro?.disconnect()
  }, [ev.message, open])
  const expandable = detail !== '' || overflows || open
  const lead = (
    <>
      {expandable ? (
        <ChevronRight size={12} className={cn('shrink-0 text-fg-subtle transition-transform', open && 'rotate-90')} />
      ) : (
        <span className="w-3 shrink-0" />
      )}
      <span className="w-24 shrink-0 tabular-nums text-fg-muted">{timeLabel}</span>
      {ev.host ? (
        <span className="w-28 shrink-0 truncate text-fg-muted" title={ev.team_number != null ? `t${ev.team_number}/${ev.host}` : ev.host}>
          {ev.team_number != null && <span className="text-fg-subtle">t{ev.team_number}/</span>}
          {ev.host}
        </span>
      ) : (
        <span className="w-28 shrink-0 text-fg-subtle">build</span>
      )}
      <Badge tone={toneForKind(ev.kind)} className="shrink-0">
        {ev.kind}
      </Badge>
      <span ref={msgRef} className={cn('min-w-0 flex-1 text-fg', open ? 'whitespace-pre-wrap break-words' : 'truncate')} title={open ? undefined : ev.message}>
        {ev.message}
      </span>
    </>
  )
  return (
    <>
      {expandable ? (
        <button type="button" onClick={onToggle} className="flex w-full items-start gap-3 py-1.5 text-left hover:bg-surface-hover" aria-expanded={open}>
          {lead}
        </button>
      ) : (
        <div className="flex w-full items-center gap-3 py-1.5">{lead}</div>
      )}
      {open && detail !== '' && (
        <pre className="mb-2 ml-[9.75rem] mr-3 max-h-96 overflow-auto whitespace-pre-wrap break-words rounded-token border border-border bg-surface-sunken p-2 text-fg">{detail}</pre>
      )}
    </>
  )
}
