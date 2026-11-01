import { useState, type ReactNode } from 'react'
import { Activity, Check, ChevronLeft, ChevronRight, Copy } from 'lucide-react'
import type { McpActivity, McpActivityPage, McpCallDetail, McpTraceStep } from '../api/types'
import { useApiKeyActivityDetail } from '../api/hooks'
import { ApiError } from '../api/client'
import { useTimeFormat } from '../lib/time'
import { EmptyState } from './EmptyState'
import { Badge, Button, Spinner, cn } from '../ui'

const STATUS_TONE = { ok: 'success', denied: 'warning', error: 'danger', rate_limited: 'orange' } as const
const STATUS_LABEL = { ok: 'OK', denied: 'Denied', error: 'Error', rate_limited: 'Rate Limited' } as const
const KIND_TONE = { sql: 'sky', api: 'indigo', power: 'brand', access: 'neutral' } as const
const KIND_LABEL = { sql: 'SQL', api: 'API', power: 'Power', access: 'Access' } as const

function bytes(n: number): string {
  if (n < 1024) return `${n} B`
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`
  return `${(n / 1024 / 1024).toFixed(1)} MB`
}

function msLabel(n: number): string {
  if (n < 1) return `${n.toFixed(2)} ms`
  if (n < 1000) return `${Math.round(n)} ms`
  return `${(n / 1000).toFixed(2)} s`
}

// The call in one line: build, then the other arguments,
// e.g. "9531b5b4 · action=reboot force=true objects=["vdi-kali-01"]".
function summarize(a: McpActivity): string {
  const args = Object.entries(a.args ?? {})
    .filter(([k, v]) => k !== 'build_id' && v !== '' && v !== null && v !== undefined && v !== false && !(Array.isArray(v) && v.length === 0))
    .map(([k, v]) => `${k}=${typeof v === 'string' ? v : JSON.stringify(v)}`)
    .join(' ')
  const parts = [a.build_id ? a.build_id.slice(0, 8) : '', args, a.error].filter(Boolean)
  return parts.join(' · ')
}

// The MCP calls made with API keys, one line each like the build log; opening
// a line shows everything the call did. Paged, newest first.
export function ApiKeyActivityLog({
  page,
  isLoading,
  error,
  admin = false,
  pageIndex,
  onNewer,
  onOlder,
}: {
  page: McpActivityPage | undefined
  isLoading: boolean
  error: unknown
  admin?: boolean
  pageIndex: number
  onNewer: () => void
  onOlder: () => void
}) {
  const fmt = useTimeFormat()
  const [open, setOpen] = useState<Set<string>>(new Set())
  const rows = page?.items ?? []

  function toggle(id: string) {
    setOpen((s) => {
      const n = new Set(s)
      if (n.has(id)) n.delete(id)
      else n.add(id)
      return n
    })
  }

  if (error) return <div className="text-sm text-danger">{error instanceof ApiError ? error.message : 'Failed to load activity.'}</div>
  if (isLoading && !page)
    return (
      <div className="flex items-center gap-2 text-sm text-fg-muted">
        <Spinner /> Loading…
      </div>
    )
  if (rows.length === 0 && pageIndex === 0) return <EmptyState icon={Activity} title="No Activity Yet" hint="Calls made with API keys show up here." />

  const first = pageIndex * 50 + 1
  return (
    <div>
      <div className="overflow-hidden rounded-token border border-border font-mono text-xs">
        {rows.map((a) => (
          <div key={a.id} className="border-b border-border last:border-b-0">
            <button type="button" onClick={() => toggle(a.id)} aria-expanded={open.has(a.id)} className="flex w-full items-center gap-2.5 px-2 py-1.5 text-left hover:bg-surface-hover">
              <ChevronRight size={12} className={cn('shrink-0 text-fg-subtle transition-transform', open.has(a.id) && 'rotate-90')} />
              <span className="w-36 shrink-0 whitespace-nowrap tabular-nums text-fg-muted">{fmt.dateTime(a.created_at)}</span>
              {admin && (
                <span className="w-24 shrink-0 truncate text-fg-muted" title={`@${a.owner_login}`}>
                  @{a.owner_login}
                </span>
              )}
              <Badge tone={STATUS_TONE[a.status]} className="w-[5.75rem] shrink-0 justify-center whitespace-nowrap">
                {STATUS_LABEL[a.status]}
              </Badge>
              <span className={cn('w-32 shrink-0 truncate', a.tool === 'power' ? 'font-semibold text-brand-fg' : 'text-fg')}>{a.tool}</span>
              <span className={cn('min-w-0 flex-1 truncate', a.error ? 'text-danger' : 'text-fg-muted')} title={summarize(a)}>
                {summarize(a)}
              </span>
              <span className="hidden w-16 shrink-0 text-right tabular-nums text-fg-muted xl:block">{a.result_bytes ? bytes(a.result_bytes) : ''}</span>
              <span className="w-14 shrink-0 text-right tabular-nums text-fg-muted">{a.duration_ms} ms</span>
            </button>
            {open.has(a.id) && <CallDetail id={a.id} admin={admin} />}
          </div>
        ))}
        {rows.length === 0 && <div className="p-3 text-fg-muted">No older calls.</div>}
      </div>
      <div className="mt-3 flex items-center gap-2 text-xs text-fg-muted">
        <span className="flex-1">
          {rows.length > 0 ? `Calls ${first.toLocaleString()}–${(first + rows.length - 1).toLocaleString()} of ${(page?.total ?? 0).toLocaleString()}` : ''}
          {pageIndex === 0 && ' · refreshes every 30 seconds'}
        </span>
        <Button variant="primary" size="sm" onClick={onNewer} disabled={pageIndex === 0}>
          <ChevronLeft size={12} /> Newer
        </Button>
        <Button variant="primary" size="sm" onClick={onOlder} disabled={!page?.next}>
          Older <ChevronRight size={12} />
        </Button>
      </div>
    </div>
  )
}

function CallDetail({ id, admin }: { id: string; admin: boolean }) {
  const { data, isLoading, error } = useApiKeyActivityDetail(id, admin)
  const fmt = useTimeFormat()
  if (isLoading)
    return (
      <div className="flex items-center gap-2 px-8 pb-3 text-fg-muted">
        <Spinner /> Loading the call…
      </div>
    )
  if (error || !data) return <div className="px-8 pb-3 text-danger">{error instanceof ApiError ? error.message : 'Failed to load the call.'}</div>
  const d = data.detail
  if (!d) {
    return (
      <div className="flex flex-col gap-3 bg-surface-sunken/40 px-8 py-3">
        <div className="text-fg-muted">This call was recorded before LaForge kept full detail. Here's what it has:</div>
        <Fields
          rows={[
            ['Key', `${data.key_name} (${data.key_prefix}…)`],
            ['From', data.remote_addr],
            ['Client', data.user_agent],
            ['Took', `${data.duration_ms} ms`],
            ['Result', bytes(data.result_bytes)],
          ]}
        />
        <Section title="Arguments">
          <Pre text={JSON.stringify(data.args, null, 2)} />
        </Section>
        {data.error && <ErrorBox text={data.error} />}
      </div>
    )
  }
  return (
    <div className="flex flex-col gap-4 bg-surface-sunken/40 px-8 py-3">
      {d.outcome.error && <ErrorBox text={d.outcome.error} phase={d.outcome.phase} />}
      <div className="grid gap-4 lg:grid-cols-2">
        <Section title="Call">
          <Fields
            rows={[
              ['Tool', d.request.tool],
              ['Status', `${STATUS_LABEL[d.outcome.status]}${d.outcome.phase ? ` (stopped at: ${d.outcome.phase})` : ''}`],
              ['When', fmt.dateTime(data.created_at)],
              ['Build', d.request.build_id ?? '—'],
              ['Call ID', data.id],
              ['Server', d.server.host ?? '—'],
            ]}
          />
        </Section>
        <Section title="Key And Client">
          <Fields
            rows={[
              ['Owner', `@${d.key.owner}`],
              ['Key', `${d.key.name} (${d.key.prefix}…), expires ${fmt.dateTime(d.key.expires_at)}`],
              ['Access', `${d.key.readable_repositories} repositories readable${d.key.limited_to?.length ? `; key limited to ${d.key.limited_to.length}` : ''}`],
              ['IP', d.client.ip ?? '—'],
              ['Client', d.client.user_agent ?? '—'],
              ['MCP Protocol', d.client.protocol_version ?? '—'],
              ...(d.client.session_id ? ([['MCP Session', d.client.session_id]] as [string, string][]) : []),
              ...Object.entries(d.client.headers ?? {}),
            ]}
          />
        </Section>
      </div>
      <Section title={`Timing (${msLabel(d.timing.total_ms)} in all)`}>
        <TimingBar t={d.timing} />
      </Section>
      <Section title={`What It Did (${d.trace.length} step${d.trace.length === 1 ? '' : 's'}${d.trace_dropped ? `, ${d.trace_dropped} more not kept` : ''})`}>
        <Trace steps={d.trace} />
      </Section>
      <Section title={`Arguments (${bytes(d.request.arguments_bytes)}${d.request.arguments_truncated ? ', cut' : ''})`}>
        <Pre text={typeof d.request.arguments === 'string' ? d.request.arguments : JSON.stringify(d.request.arguments ?? {}, null, 2)} />
      </Section>
      {d.result.preview !== undefined && (
        <Section
          title={`Result (${bytes(d.result.bytes)}${d.result.lists ? `; ${Object.entries(d.result.lists).map(([k, n]) => `${k}: ${n}`).join(', ')}` : ''}${d.result.clipped_fields ? `; ${d.result.clipped_fields} long fields cut` : ''})`}
        >
          <Pre text={d.result.preview + (d.result.preview_truncated ? '\n… (preview ends here; the agent got the whole result)' : '')} copy />
        </Section>
      )}
    </div>
  )
}

function Section({ title, children }: { title: string; children: ReactNode }) {
  return (
    <div className="min-w-0">
      <div className="mb-1 font-sans text-[11px] font-semibold uppercase tracking-wide text-fg-subtle">{title}</div>
      {children}
    </div>
  )
}

function Fields({ rows }: { rows: [string, string][] }) {
  return (
    <dl className="grid grid-cols-[8rem_minmax(0,1fr)] gap-x-3 gap-y-0.5">
      {rows.map(([k, v]) => (
        <div key={k} className="contents">
          <dt className="text-fg-subtle">{k}</dt>
          <dd className="break-all text-fg">{v || '—'}</dd>
        </div>
      ))}
    </dl>
  )
}

function ErrorBox({ text, phase }: { text: string; phase?: string }) {
  return (
    <div className="whitespace-pre-wrap break-words rounded-token border border-danger/30 bg-danger-soft p-2 text-danger">
      {phase && <span className="font-semibold">Failed at {phase}: </span>}
      {text}
    </div>
  )
}

function Pre({ text, copy = false }: { text: string; copy?: boolean }) {
  const [copied, setCopied] = useState(false)
  async function doCopy() {
    try {
      await navigator.clipboard.writeText(text)
      setCopied(true)
      setTimeout(() => setCopied(false), 1500)
    } catch {
      // Clipboard blocked; the text is selectable.
    }
  }
  return (
    <div className="relative">
      {copy && (
        <Button variant="brand" size="sm" className="absolute right-2 top-2" onClick={doCopy}>
          {copied ? <Check size={12} /> : <Copy size={12} />} {copied ? 'Copied' : 'Copy'}
        </Button>
      )}
      <pre className="max-h-96 overflow-auto whitespace-pre-wrap break-words rounded-token border border-border bg-surface p-2 text-fg">{text}</pre>
    </div>
  )
}

const PHASES = [
  ['queue_ms', 'Queue', 'bg-fg-subtle'],
  ['access_ms', 'Access check', 'bg-sky'],
  ['run_ms', 'Tool', 'bg-indigo'],
  ['encode_ms', 'Encode', 'bg-purple'],
] as const

function TimingBar({ t }: { t: McpCallDetail['timing'] }) {
  const total = Math.max(t.total_ms, PHASES.reduce((n, [k]) => n + t[k], 0), 0.001)
  return (
    <div>
      <div className="flex h-2.5 overflow-hidden rounded-full bg-surface">
        {PHASES.map(([k, label, color]) => (
          <div key={k} className={color} style={{ width: `${(t[k] / total) * 100}%` }} title={`${label}: ${msLabel(t[k])}`} />
        ))}
      </div>
      <div className="mt-1.5 flex flex-wrap gap-x-4 gap-y-1 text-fg-muted">
        {PHASES.map(([k, label, color]) => (
          <span key={k} className="flex items-center gap-1.5">
            <span className={cn('inline-block size-2 rounded-full', color)} />
            {label} {msLabel(t[k])}
          </span>
        ))}
        <span>Other {msLabel(Math.max(0, t.total_ms - PHASES.reduce((n, [k]) => n + t[k], 0)))}</span>
      </div>
    </div>
  )
}

function Trace({ steps }: { steps: McpTraceStep[] }) {
  const [open, setOpen] = useState<Set<number>>(new Set())
  if (steps.length === 0) return <div className="text-fg-muted">Nothing: the call stopped before doing anything.</div>
  return (
    <div className="overflow-hidden rounded-token border border-border bg-surface">
      {steps.map((s, i) => {
        const extra = s.kind === 'power' || (s.detail?.length ?? 0) > 120 || (s.error?.length ?? 0) > 120
        const isOpen = open.has(i)
        const facts = [
          s.rows !== undefined ? `${s.rows} row${s.rows === 1 ? '' : 's'}` : '',
          s.status ? `HTTP ${s.status}` : '',
          s.bytes ? bytes(s.bytes) : '',
        ].filter(Boolean)
        return (
          <div key={i} className="border-b border-border last:border-b-0">
            <button
              type="button"
              disabled={!extra}
              onClick={() => setOpen((o) => (o.has(i) ? new Set([...o].filter((x) => x !== i)) : new Set(o).add(i)))}
              className={cn('flex w-full items-start gap-3 px-2 py-1 text-left', extra && 'hover:bg-surface-hover')}
            >
              <span className="w-16 shrink-0 text-right tabular-nums text-fg-subtle">+{msLabel(s.at_ms)}</span>
              <Badge tone={KIND_TONE[s.kind]} className="w-14 shrink-0 justify-center">
                {KIND_LABEL[s.kind]}
              </Badge>
              <span className="min-w-0 flex-1">
                <span className={cn('break-all', s.error ? 'text-danger' : 'text-fg')}>{s.name}</span>
                {s.detail && <span className={cn('ml-2 text-fg-muted', isOpen ? 'whitespace-pre-wrap break-all' : 'line-clamp-1 break-all')}>{s.detail}</span>}
                {s.error && <div className={cn('text-danger', !isOpen && 'line-clamp-2')}>{s.error}</div>}
              </span>
              <span className="shrink-0 text-right tabular-nums text-fg-muted">{facts.join(' · ')}</span>
              <span className="w-16 shrink-0 text-right tabular-nums text-fg">{msLabel(s.ms)}</span>
            </button>
            {isOpen && s.response && (
              <div className="px-2 pb-2 pl-[9.5rem]">
                <div className="mb-1 text-fg-subtle">Response</div>
                <pre className="max-h-72 overflow-auto whitespace-pre-wrap break-words rounded-token bg-surface-sunken p-2 text-fg">{prettyJSON(s.response)}</pre>
              </div>
            )}
          </div>
        )
      })}
    </div>
  )
}

function prettyJSON(s: string): string {
  try {
    return JSON.stringify(JSON.parse(s), null, 2)
  } catch {
    return s
  }
}
