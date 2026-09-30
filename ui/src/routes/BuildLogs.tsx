import { useParams } from '@tanstack/react-router'
import { useMemo, useState } from 'react'
import { Radio, Search, ChevronRight } from 'lucide-react'
import { useEvents, useLiveEvents } from '../api/hooks'
import { useTimeFormat } from '../lib/time'
import type { LFEvent } from '../api/types'
import { EmptyState } from '../components/EmptyState'
import { Badge, Select, cn } from '../ui'

// toneForKind colours an event's kind badge by what it IS, so the journal is
// scannable by type/outcome: red for failures, green for completions, amber for
// tolerated/skipped, and a distinct hue per subsystem (access, network, steps,
// deploy, agent). Matched loosely on the kind string so new kinds still land
// somewhere sensible.
function toneForKind(kind: string): 'danger' | 'success' | 'warning' | 'info' | 'purple' | 'sky' | 'indigo' | 'ink' | 'neutral' {
  const k = kind.toLowerCase()
  if (k.includes('fail') || k.includes('error') || k.includes('invalid')) return 'danger'
  if (k.includes('ignored') || k.includes('skip') || k.includes('note') || k.includes('warn')) return 'warning'
  if (k.includes('done') || k.includes('finish') || k.includes('complete') || k.includes('succeed') || k.includes('deployed') || k.includes('opened') || k.includes('configured') || k.includes('materialized')) return 'success'
  if (k.startsWith('access')) return 'purple'
  if (k.startsWith('network')) return 'sky'
  if (k.includes('step')) return 'info'
  if (k.includes('deploy') || k.includes('destroy') || k.includes('teardown') || k.includes('build')) return 'indigo'
  if (k.includes('heartbeat') || k.includes('session')) return 'ink'
  return 'neutral'
}

// eventDetail pulls the full text an event carries in its payload (step events
// store the whole command output / error under `detail`, while the message is
// only a short one-liner) so an expanded row can show the real error.
function eventDetail(payload: unknown): string {
  if (payload && typeof payload === 'object' && 'detail' in payload) {
    const d = (payload as { detail?: unknown }).detail
    if (typeof d === 'string' && d.trim() !== '') return d
  }
  return ''
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
  const { data: history } = useEvents(buildId)
  const [live, setLive] = useState<LFEvent[]>([])
  const { connected } = useLiveEvents(buildId, (ev) => setLive((prev) => [...prev, ev]))
  const [query, setQuery] = useState('')
  const [kind, setKind] = useState('')
  const [expanded, setExpanded] = useState<Set<string>>(new Set())
  const fmt = useTimeFormat()

  function toggle(id: string) {
    setExpanded((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  const all = useMemo(() => {
    const seen = new Set((history ?? []).map((e) => e.id))
    return [...(history ?? []), ...live.filter((e) => !seen.has(e.id))]
  }, [history, live])

  // Every distinct kind actually present, for the kind filter -- so it only
  // ever offers values that exist in this build's journal.
  const kinds = useMemo(() => Array.from(new Set(all.map((e) => e.kind))).sort(), [all])

  const shown = useMemo(() => {
    const q = query.trim().toLowerCase()
    return all.filter((e) => {
      if (kind && e.kind !== kind) return false
      if (!q) return true
      return [e.message, e.kind, e.host ?? '', e.team_number != null ? `team ${e.team_number}` : '']
        .some((v) => v.toLowerCase().includes(q))
    })
  }, [all, query, kind])

  return (
    <div>
      <div className="mb-3 flex flex-wrap items-center gap-2">
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
        </div>
      </div>

      {all.length === 0 ? (
        <EmptyState icon={Radio} title="No events yet" hint="Events appear here as the build progresses." />
      ) : shown.length === 0 ? (
        <EmptyState icon={Search} title="No events match" hint="Clear the search or kind filter to see the full journal." />
      ) : (
        <div className="flex flex-col font-mono text-xs">
          {shown.map((ev) => {
            const isOpen = expanded.has(ev.id)
            const detail = eventDetail(ev.payload)
            return (
              <div key={ev.id} className="border-b border-border">
                {/* Whole row is a toggle: click to expand and read the full
                    message + payload detail (errors are longer than one line). */}
                <button
                  type="button"
                  onClick={() => toggle(ev.id)}
                  className="flex w-full items-center gap-3 py-1.5 text-left hover:bg-surface-hover"
                >
                  <ChevronRight size={12} className={cn('shrink-0 text-fg-subtle transition-transform', isOpen && 'rotate-90')} />
                  <span className="w-24 shrink-0 tabular-nums text-fg-muted">{fmt.time(ev.created_at)}</span>
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
                  <span className={cn('min-w-0 flex-1 text-fg', !isOpen && 'truncate')}>{ev.message}</span>
                </button>
                {isOpen && (
                  <div className="pb-2 pl-[9.75rem] pr-3 text-fg-muted">
                    <div className="whitespace-pre-wrap break-words">{ev.message}</div>
                    {detail && detail !== ev.message && (
                      <pre className="mt-1 max-h-96 overflow-auto whitespace-pre-wrap break-words rounded-token border border-border bg-surface-sunken p-2 text-fg">
                        {detail}
                      </pre>
                    )}
                  </div>
                )}
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}
