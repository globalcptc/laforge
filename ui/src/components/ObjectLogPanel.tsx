import { useEffect, useMemo, useRef, useState } from 'react'
import { Link } from '@tanstack/react-router'
import { Radio, ScrollText, FileCode, Maximize2, Minimize2, ListChecks, CircleCheck, CircleX, CircleDashed, Loader, ShieldCheck, ShieldOff, ChevronDown, Info, ServerCog, Eye, EyeOff, Copy, Check, Terminal, Hourglass, Flag, KeyRound, TriangleAlert, RotateCcw } from 'lucide-react'
import { useObjectEvents, useObjectHeartbeats, useObjectRender, useObjectCloudInit, useObjectSteps, useObjectInfra, useObjectConfig } from '../api/hooks'
import { useTimeFormat } from '../lib/time'
import { ApiError } from '../api/client'
import type { RenderedStep, CloudInitView, StepStatus, ObjectKind, DeployedObject, HostCredential, AgentHeartbeat } from '../api/types'
import { EmptyState } from './EmptyState'
import { Badge, Button, buttonClass, cn, InspectorPanel, Spinner } from '../ui'
import { StatusBadge, PowerBadge, GONE_STATES } from './StatusBadge'

type Tab = 'info' | 'events' | 'steps' | 'heartbeats' | 'render'

// A percentage metric (0-100), rounded, or a dash when the agent didn't send it.
function pct(v: number | null): string {
  return v == null ? '—' : `${Math.round(v)}%`
}

// A bytes-per-second rate, compact (—, 0, 1.2K, 3M, ...).
function bps(v: number | null): string {
  if (v == null) return '—'
  if (v < 1) return '0'
  const units = ['', 'K', 'M', 'G']
  let n = v
  let u = 0
  while (n >= 1024 && u < units.length - 1) {
    n /= 1024
    u++
  }
  return `${n >= 10 || u === 0 ? Math.round(n) : n.toFixed(1)}${units[u]}`
}

// "Per-object logs" -- one host/container's own
// slice of the journal (real data, GET
// /builds/{id}/objects/{objectId}/events) plus, as of the real
// append-only agent_heartbeat log (migrations/00007), its own real
// heartbeat history -- "live troubleshooting" for exactly this object,
// including the real remote address each check-in came from (real
// "rules checking" signal: an address change mid-competition is
// actionable, not just an oddity).
export function ObjectLogPanel({
  repoId,
  buildId,
  objectId,
  label,
  kind = 'host',
  object,
  teamNumber,
  initialTab,
  focusGroup,
  onRestartFrom,
  onClose,
}: {
  repoId: string
  buildId: string
  objectId: string
  label: string
  kind?: ObjectKind
  // The clicked object's own row, when the caller has it -- powers the Info
  // tab (identity + agent + power) and scopes the Findings tab to this
  // object. Optional so the panel still works from a caller that only knows
  // the id (it just shows less on Info).
  object?: DeployedObject
  teamNumber?: number
  // Open straight to a tab (e.g. the dependency graph deep-links to 'steps')
  // and, for the Steps tab, scroll to and briefly highlight one authored step
  // group by its group_index.
  initialTab?: Tab
  focusGroup?: number
  // Restart from step (docs/plans/restart-from-step.md): when set, each
  // authored step group on the Steps tab offers "Restart from here" and this
  // is called with the group's index. The caller decides when it applies
  // (the host's state, the build not read-only).
  onRestartFrom?: (group: number) => void
  onClose: () => void
}) {
  const [tab, setTab] = useState<Tab>(initialTab ?? 'info')
  const [wide, setWide] = useState(false)
  // When a step in the Steps tab is clicked, jump to the Render tab and scroll
  // its rendered script into view (RenderedStep.index). Carries a nonce so
  // clicking the same step twice still re-scrolls.
  const [renderFocus, setRenderFocus] = useState<{ index: number; n: number } | null>(null)

  const fmt = useTimeFormat()
  // A network is provisioned by the builder, not by an in-guest agent, so
  // it has a log (events) and findings but no steps/heartbeats/render --
  // show only what applies to what was clicked.
  const isNetwork = kind === 'network'
  const { data: events, isLoading: eventsLoading } = useObjectEvents(buildId, objectId)
  const { data: heartbeats, isLoading: heartbeatsLoading } = useObjectHeartbeats(buildId, objectId)
  const { data: buildSteps, isLoading: stepsLoading } = useObjectSteps(buildId, objectId, !isNetwork && tab === 'steps')
  const {
    data: renderedSteps,
    isLoading: renderLoading,
    error: renderError,
  } = useObjectRender(buildId, objectId, tab === 'render')
  const { data: cloudInit, error: cloudInitError } = useObjectCloudInit(buildId, objectId, !isNetwork && tab === 'render')

  return (
    <InspectorPanel
      open
      overlay
      onClose={onClose}
      title={label}
      subtitle={kind}
      width={wide ? 'min(80vw, 60rem)' : 'min(92vw, 28rem)'}
      headerAction={
        <div className="flex items-center gap-1.5">
          {/* A root/admin shell on this host, the natural place to launch it:
              the panel you open to work on a host. Hosts and containers only,
              and not once the instance is gone. */}
          {!isNetwork && (!object || !GONE_STATES.has(object.status)) && (
            <Link
              to="/repos/$repoId/builds/$buildId/hosts/$objectId/terminal"
              params={{ repoId, buildId, objectId }}
              // Its own browser tab, so several shells can sit side by side and
              // the build page stays where it was.
              target="_blank"
              rel="noopener"
              className={buttonClass({ variant: 'secondary', size: 'sm' })}
              title="Open an interactive root/admin shell on this host in a new tab"
            >
              <Terminal size={14} /> Open Terminal
            </Link>
          )}
          <Button
            variant="ghost"
            size="icon"
            onClick={() => setWide((v) => !v)}
            aria-label={wide ? 'Collapse panel' : 'Expand panel'}
            title={wide ? 'Collapse' : 'Expand'}
          >
            {wide ? <Minimize2 size={14} /> : <Maximize2 size={14} />}
          </Button>
        </div>
      }
    >
      <div className="-mx-4 -mt-3 mb-3 flex flex-nowrap gap-1 overflow-x-auto border-b border-border px-4 pb-2">
        <TabButton active={tab === 'info'} onClick={() => setTab('info')} icon={Info} label="Info" />
        <TabButton active={tab === 'events'} onClick={() => setTab('events')} icon={ScrollText} label="Events" />
        {!isNetwork && <TabButton active={tab === 'steps'} onClick={() => setTab('steps')} icon={ListChecks} label="Steps" />}
        {!isNetwork && <TabButton active={tab === 'heartbeats'} onClick={() => setTab('heartbeats')} icon={Radio} label="Heartbeats" />}
        {!isNetwork && <TabButton active={tab === 'render'} onClick={() => setTab('render')} icon={FileCode} label="Render" />}
      </div>

      {tab === 'info' && <InfoTab buildId={buildId} objectId={objectId} object={object} kind={kind} label={label} teamNumber={teamNumber} fmtTime={fmt.time} />}

      {tab === 'events' && (
        <>
          {eventsLoading && <LoadingRow />}
          {!eventsLoading && (!events || events.length === 0) && (
            <EmptyState
              icon={ScrollText}
              title="No events yet for this object"
              hint="Deploy, destroy, and access lifecycle events for this specific host/container will appear here."
            />
          )}
          {events && events.length > 0 && (
            <div className="flex flex-col gap-2 font-mono text-xs">
              {events.map((ev) => (
                <div key={ev.id} className="border-b border-border pb-2">
                  <div className="flex items-center gap-2">
                    <span className="text-fg-muted">{fmt.time(ev.created_at)}</span>
                    <Badge tone="accent">{ev.kind}</Badge>
                  </div>
                  <div className="mt-1 text-fg">{ev.message}</div>
                  {isDetailObject(ev.payload) && (
                    <div className="mt-1 flex flex-col gap-0.5 border-l-2 border-border pl-2 text-[11px] text-fg-muted">
                      {Object.entries(ev.payload)
                        .filter(([, v]) => v !== '' && v != null)
                        .map(([k, v]) => (
                          <div key={k} className="break-all">
                            <span className="text-fg-subtle">{k}:</span> <span className="text-fg">{fmtDetailValue(v)}</span>
                          </div>
                        ))}
                    </div>
                  )}
                </div>
              ))}
            </div>
          )}
        </>
      )}
      {tab === 'heartbeats' && (
        <>
          {heartbeatsLoading && <LoadingRow />}
          {!heartbeatsLoading && (!heartbeats || heartbeats.length === 0) && (
            <EmptyState
              icon={Radio}
              title="No heartbeats yet for this object"
              hint="Real check-ins from this object's agent will appear here as it heartbeats."
            />
          )}
          {heartbeats && heartbeats.length > 0 && (
            <>
              <HeartbeatCharts heartbeats={heartbeats} wide={wide} />
              <div className="overflow-x-auto">
              <table className="w-full border-collapse font-mono text-xs">
                <thead>
                  <tr className="border-b border-border text-left text-fg-subtle">
                    <th className="py-1 pr-2 font-medium">Time</th>
                    <th className="py-1 px-2 text-right font-medium">CPU</th>
                    <th className="py-1 px-2 text-right font-medium">Mem</th>
                    <th className="py-1 px-2 text-right font-medium">Disk</th>
                    <th className="py-1 px-2 text-right font-medium" title="Network received / sent per second">Net ↓/↑</th>
                    <th className="py-1 pl-2 font-medium">Remote</th>
                  </tr>
                </thead>
                <tbody>
                  {heartbeats.map((h, i) => {
                    const addrChanged = i < heartbeats.length - 1 && h.remote_addr !== heartbeats[i + 1].remote_addr
                    return (
                      <tr key={h.id} className="border-b border-border/60">
                        <td className="whitespace-nowrap py-1 pr-2 text-fg-muted">{fmt.time(h.created_at)}</td>
                        <td className="py-1 px-2 text-right">{pct(h.cpu_pct)}</td>
                        <td className="py-1 px-2 text-right">{pct(h.mem_pct)}</td>
                        <td className="py-1 px-2 text-right">{pct(h.disk_pct)}</td>
                        <td className="whitespace-nowrap py-1 px-2 text-right text-fg-muted">
                          {bps(h.net_rx_bps)} / {bps(h.net_tx_bps)}
                        </td>
                        <td className={cn('whitespace-nowrap py-1 pl-2', addrChanged ? 'text-warning' : 'text-fg-muted')} title={addrChanged ? 'remote address changed from the previous check-in' : undefined}>
                          {h.remote_addr ?? '—'}
                        </td>
                      </tr>
                    )
                  })}
                </tbody>
              </table>
              </div>
            </>
          )}
        </>
      )}
      {tab === 'steps' && (
        <>
          {stepsLoading && <LoadingRow />}
          {!stepsLoading && (!buildSteps || buildSteps.length === 0) && (
            <EmptyState
              icon={ListChecks}
              title="No build steps yet"
              hint="Steps appear here once the agent has checked in and they're queued to run."
            />
          )}
          {buildSteps && buildSteps.length > 0 && (
            <StepStatusList
              steps={buildSteps}
              focusGroup={focusGroup}
              onRestartFrom={onRestartFrom}
              onOpenRendered={(index) => {
                setRenderFocus({ index, n: Date.now() })
                setTab('render')
              }}
            />
          )}
        </>
      )}
      {tab === 'render' && (
        <>
          {cloudInit && !cloudInitError && <CloudInitSection view={cloudInit} />}
          {renderLoading && <LoadingRow />}
          {renderError && <div className="text-sm text-danger">{renderError instanceof ApiError ? renderError.message : 'Failed to render'}</div>}
          {renderedSteps && renderedSteps.length === 0 && <EmptyState icon={FileCode} title="No steps on this object" hint="Nothing to render." />}
          {renderedSteps && renderedSteps.length > 0 && <RenderedSteps steps={renderedSteps} focus={renderFocus} />}
        </>
      )}
    </InspectorPanel>
  )
}

// Per-host resource charts at the top of the Heartbeats tab: CPU, memory and
// disk over this object's own heartbeat history. One combined chart in the
// narrow drawer; three larger independent charts when the panel is expanded.
// Colours match the Overview dashboard's resource charts (DashboardCharts):
// CPU = accent, memory = purple, disk = warning.
const HB_METRICS = [
  { key: 'cpu_pct', label: 'CPU', color: 'var(--color-accent)' },
  { key: 'mem_pct', label: 'Memory', color: 'var(--color-purple)' },
  { key: 'disk_pct', label: 'Disk', color: 'var(--color-warning)' },
] as const

function HeartbeatCharts({ heartbeats, wide }: { heartbeats: AgentHeartbeat[]; wide: boolean }) {
  // Heartbeats come newest-first; a time series reads oldest -> newest.
  const chrono = useMemo(() => [...heartbeats].reverse(), [heartbeats])
  const times = chrono.map((h) => h.created_at)
  const series = HB_METRICS.map((m) => ({ values: chrono.map((h) => h[m.key]), color: m.color, label: m.label }))
  // Nothing to plot until at least one check-in carried a metric (older agents
  // report null); skip the charts rather than drawing flat lines at zero.
  if (!series.some((s) => s.values.some((v) => v != null))) return null
  if (wide) {
    return (
      <div className="mb-3 grid grid-cols-3 gap-3">
        {series.map((s) => (
          <MetricChart key={s.label} title={s.label} times={times} series={[s]} height={120} />
        ))}
      </div>
    )
  }
  return (
    <div className="mb-3">
      <MetricChart title="CPU · Memory · Disk" times={times} series={series} height={96} />
    </div>
  )
}

// MetricChart is a small 0-100% line chart over the heartbeat timeline, drawn
// the same way as the dashboard's ResourceLineChart (shared axis/gridline
// layout and tokens). A null sample breaks the line rather than reading as a
// drop to zero.
function MetricChart({ title, times, series, height = 96 }: {
  title: string
  times: string[]
  series: { values: (number | null)[]; color: string; label: string }[]
  height?: number
}) {
  const fmt = useTimeFormat()
  const width = 400
  const max = 100
  const n = times.length
  const stepX = n > 1 ? width / (n - 1) : 0
  const midY = height / 2
  const pathFor = (values: (number | null)[]) => {
    let d = ''
    let pen = false
    values.forEach((v, i) => {
      if (v == null) {
        pen = false
        return
      }
      const x = n > 1 ? i * stepX : width / 2
      const y = height - (Math.min(Math.max(v, 0), max) / max) * height
      d += `${pen ? 'L' : 'M'} ${x.toFixed(1)} ${y.toFixed(1)} `
      pen = true
    })
    return d.trim()
  }
  const latest = (values: (number | null)[]) => {
    for (let i = values.length - 1; i >= 0; i--) if (values[i] != null) return values[i] as number
    return null
  }
  return (
    <div className="rounded-token border border-border p-3">
      <div className="mb-2 text-[11px] font-semibold uppercase tracking-wide text-fg-subtle">{title}</div>
      <div className="flex gap-2">
        <div className="flex shrink-0 flex-col justify-between text-right text-[10px] text-fg-subtle" style={{ height }}>
          <span>100%</span>
          <span>50%</span>
          <span>0%</span>
        </div>
        <div className="min-w-0 flex-1">
          <svg viewBox={`0 0 ${width} ${height}`} preserveAspectRatio="none" className="w-full" style={{ height }}>
            <line x1={0} y1={height} x2={width} y2={height} stroke="var(--color-border)" strokeWidth={1} vectorEffect="non-scaling-stroke" />
            <line x1={0} y1={midY} x2={width} y2={midY} stroke="var(--color-border)" strokeWidth={1} strokeDasharray="3 3" vectorEffect="non-scaling-stroke" />
            {series.map((s) => (
              <path key={s.label} d={pathFor(s.values)} fill="none" stroke={s.color} strokeWidth={2} vectorEffect="non-scaling-stroke" />
            ))}
          </svg>
          <div className="mt-1 flex justify-between text-[10px] text-fg-subtle">
            <span>{n > 0 ? fmt.timeShort(times[0]) : ''}</span>
            <span>{n > 0 ? fmt.timeShort(times[n - 1]) : ''}</span>
          </div>
        </div>
      </div>
      <div className="mt-1.5 flex flex-wrap gap-x-3 gap-y-1 text-xs text-fg-muted">
        {series.map((s) => {
          const lv = latest(s.values)
          return (
            <span key={s.label} className="inline-flex items-center gap-1">
              <span className="inline-block h-2 w-2 rounded-full" style={{ background: s.color }} />
              {s.label}
              {lv != null ? ` ${Math.round(lv)}%` : ''}
            </span>
          )
        })}
      </div>
    </div>
  )
}

// RenderedSteps lists a host's fully-resolved steps -- shared by the narrow
// drawer preview and the wide expanded modal, so both show identical
// content, just with different room.
function RenderedSteps({ steps, focus }: { steps: RenderedStep[]; focus?: { index: number; n: number } | null }) {
  const hasTemplated = steps.some((s) => s.segments?.some((seg) => seg.templated))
  const stepRefs = useRef<Record<number, HTMLDivElement | null>>({})
  const [flash, setFlash] = useState<number | null>(null)
  // Scroll the step the Steps tab linked to into view and briefly flash it.
  // Keyed on focus.n so re-clicking the same step re-triggers.
  useEffect(() => {
    if (!focus) return
    const el = stepRefs.current[focus.index]
    if (!el) return
    el.scrollIntoView({ block: 'center', behavior: 'smooth' })
    setFlash(focus.index)
    const t = setTimeout(() => setFlash(null), 1800)
    return () => clearTimeout(t)
  }, [focus])
  return (
    <div className="flex flex-col gap-3">
      {hasTemplated && (
        <div className="flex items-center gap-1.5 text-xs text-fg-subtle">
          <span className="rounded-sm bg-accent-soft px-1 font-mono text-accent-fg">highlighted</span>
          replaced by a template value
        </div>
      )}
      {steps.map((s) => (
        <div
          key={s.index}
          ref={(el) => { stepRefs.current[s.index] = el }}
          className={cn('overflow-hidden rounded-token border', flash === s.index ? 'border-accent ring-2 ring-accent/50 transition-colors' : 'border-border')}
        >
          <div className="flex items-center gap-2 border-b border-border bg-surface-sunken px-2 py-1 text-xs">
            <span className="text-fg-muted">#{s.index}</span>
            <span className="font-medium text-fg">{s.action}</span>
            {s.script_name && <span className="text-accent">{s.script_name}</span>}
          </div>
          {s.segments && s.segments.length > 0 ? (
            <pre className="overflow-x-auto whitespace-pre-wrap p-2 font-mono text-xs text-fg">
              {s.segments.map((seg, i) =>
                seg.templated ? (
                  <span key={i} className="rounded-sm bg-accent-soft text-accent-fg" title="Replaced by a template value">
                    {seg.text}
                  </span>
                ) : (
                  <span key={i}>{seg.text}</span>
                ),
              )}
            </pre>
          ) : s.rendered ? (
            <pre className="overflow-x-auto whitespace-pre-wrap p-2 font-mono text-xs text-fg">{s.rendered}</pre>
          ) : (
            <pre className="overflow-x-auto whitespace-pre-wrap p-2 font-mono text-xs text-fg-muted">{JSON.stringify(s.raw, null, 2)}</pre>
          )}
        </div>
      ))}
    </div>
  )
}

// CloudInitSection shows the first-boot configuration a host received before
// any authored step ran -- the cloud-init / cloudbase-init / EC2Launch user-data
// (and, on Incus/MicroCloud Windows, the config-drive meta-data) that installs
// the agent. Shown above the rendered steps because it runs first, and because
// when the agent HASN'T run this is the only config that did -- the view you
// need to debug a box you can't log into yet.
function CloudInitSection({ view }: { view: CloudInitView }) {
  return (
    <div className="mb-3 flex flex-col gap-2">
      <div className="flex flex-wrap items-center gap-2 text-xs">
        <span className="font-semibold text-fg">First-Boot Configuration</span>
        <Badge tone="info" className="rounded-full py-0.5">
          {view.format}
        </Badge>
        {view.os && <span className="text-fg-muted">{view.os}</span>}
      </div>
      <div className="text-xs text-fg-subtle">{view.delivery}</div>
      {view.temp_password && (
        <div className="flex flex-wrap items-center gap-x-3 gap-y-1 rounded-token border border-border bg-surface-sunken px-2 py-1.5 text-xs">
          <span className="flex items-center gap-1.5 font-medium text-fg-muted">
            <KeyRound size={12} /> Temporary login
          </span>
          <span className="text-fg">
            {view.temp_username}
            {' / '}
            <span className="font-mono">{view.temp_password}</span>
          </span>
          <span className="text-fg-subtle">until the agent sets the real password</span>
        </div>
      )}
      {view.note && (
        <div className="flex items-start gap-1.5 rounded-token border border-warning bg-warning-soft p-2 text-xs text-fg">
          <TriangleAlert size={13} className="mt-0.5 shrink-0 text-warning" />
          <span>{view.note}</span>
        </div>
      )}
      {view.meta_data && (
        <div className="overflow-hidden rounded-token border border-border">
          <div className="border-b border-border bg-surface-sunken px-2 py-1 text-xs font-medium text-fg">meta-data</div>
          <pre className="overflow-x-auto whitespace-pre-wrap p-2 font-mono text-xs text-fg">{view.meta_data}</pre>
        </div>
      )}
      <div className="overflow-hidden rounded-token border border-border">
        <div className="border-b border-border bg-surface-sunken px-2 py-1 text-xs font-medium text-fg">user-data</div>
        <pre className="overflow-x-auto whitespace-pre-wrap p-2 font-mono text-xs text-fg">{view.user_data}</pre>
      </div>
    </div>
  )
}

// StepStatusList shows where a host is in its build: each materialized step
// with its live status (pending → running → done / failed) and, when a step
// has validators, whether each passed. This is the execution truth (the
// real agent_task rows), distinct from the Render tab's rendered script.
const STEP_STYLE: Record<string, { icon: typeof CircleCheck; tone: 'neutral' | 'info' | 'success' | 'warning' | 'danger'; label: string; spin?: boolean }> = {
  pending: { icon: CircleDashed, tone: 'neutral', label: 'Pending' },
  leased: { icon: Loader, tone: 'info', label: 'Running', spin: true },
  done: { icon: CircleCheck, tone: 'success', label: 'Done' },
  failed: { icon: CircleX, tone: 'danger', label: 'Failed' },
  // Queued but held until a dependency (depends_on, or this or an earlier
  // step's wait_for) is ready -- visible, not yet run.
  blocked: { icon: Hourglass, tone: 'warning', label: 'Dependency Blocked' },
  // An ignore_errors step that failed: tolerated, the host moved on past it.
  ignored: { icon: CircleX, tone: 'neutral', label: 'Ignored' },
}

function StepBadge({ status }: { status: string }) {
  const s = STEP_STYLE[status] ?? { icon: CircleDashed, tone: 'neutral' as const, label: status }
  const Icon = s.icon
  return (
    <Badge tone={s.tone} className="rounded-full py-0.5">
      <Icon size={11} className={s.spin ? 'animate-spin' : undefined} /> {s.label}
    </Badge>
  )
}

// How long an authored step took, wall-clock: from the first sub-command that
// started to the last that finished (or now, if any is still running). null when
// the step hasn't started yet.
function groupDurationMs(steps: StepStatus[]): number | null {
  let min = Infinity
  let max = 0
  let running = false
  for (const s of steps) {
    if (s.started_at) {
      const t = Date.parse(s.started_at)
      if (!Number.isNaN(t)) min = Math.min(min, t)
      if (!s.finished_at) running = true
    }
    if (s.finished_at) {
      const t = Date.parse(s.finished_at)
      if (!Number.isNaN(t)) max = Math.max(max, t)
    }
  }
  if (min === Infinity) return null
  return Math.max(0, (running ? Date.now() : max) - min)
}

// An event's payload (what we sent the builder, a step's output, etc.) is a
// non-empty object worth showing as detail under the message.
function isDetailObject(p: unknown): p is Record<string, unknown> {
  return !!p && typeof p === 'object' && !Array.isArray(p) && Object.keys(p as object).length > 0
}

// Render one payload value compactly: arrays comma-joined, objects as JSON,
// long strings truncated so a step's full output doesn't flood the row.
function fmtDetailValue(v: unknown): string {
  if (Array.isArray(v)) return v.length ? v.join(', ') : '—'
  if (v && typeof v === 'object') return JSON.stringify(v)
  const s = String(v)
  return s.length > 160 ? s.slice(0, 160) + '…' : s
}

// Compact duration: 15s, 1m 12s, 2h 5m.
function formatStepDuration(ms: number): string {
  const secs = Math.round(ms / 1000)
  if (secs < 60) return `${secs}s`
  const m = Math.floor(secs / 60)
  const s = secs % 60
  if (m < 60) return s ? `${m}m ${s}s` : `${m}m`
  const h = Math.floor(m / 60)
  const mm = m % 60
  return mm ? `${h}h ${mm}m` : `${h}h`
}

function StepStatusList({ steps, focusGroup, onRestartFrom, onOpenRendered }: { steps: StepStatus[]; focusGroup?: number; onRestartFrom?: (group: number) => void; onOpenRendered?: (renderIndex: number) => void }) {
  const [open, setOpen] = useState<Set<number>>(new Set())
  const groupRefs = useRef<Record<number, HTMLDivElement | null>>({})
  const [flash, setFlash] = useState<number | null>(null)
  // Deep-link from the dependency graph: scroll the targeted step group into
  // view and flash it, once its steps have loaded.
  useEffect(() => {
    if (focusGroup == null) return
    const el = groupRefs.current[focusGroup]
    if (!el) return
    el.scrollIntoView({ block: 'center', behavior: 'smooth' })
    setFlash(focusGroup)
    const t = setTimeout(() => setFlash(null), 1800)
    return () => clearTimeout(t)
  }, [focusGroup, steps])
  function toggle(i: number) {
    setOpen((prev) => {
      const next = new Set(prev)
      next.has(i) ? next.delete(i) : next.add(i)
      return next
    })
  }
  // Group consecutive steps that came from the same authored step, so the
  // expanded sub-commands (write file, execute, validate) read as one step
  // under a named heading rather than a flat wall of rows.
  const groups: { key: number; label?: string; steps: StepStatus[]; waitFor?: string[]; milestone?: string; renderIndex?: number }[] = []
  for (const s of steps) {
    const last = groups[groups.length - 1]
    if (last && last.key === s.group_index) last.steps.push(s)
    else groups.push({ key: s.group_index, label: s.group_label, steps: [s], waitFor: s.wait_for, milestone: s.milestone, renderIndex: s.render_index })
  }
  return (
    <div className="flex flex-col gap-3">
      {groups.map((g) => (
        <div
          key={g.key}
          ref={(el) => { groupRefs.current[g.key] = el }}
          className={cn('flex flex-col gap-1.5 rounded-token', flash === g.key && 'bg-accent-soft/60 ring-2 ring-accent/50 -mx-1 px-1 py-1 transition-colors')}
        >
          {g.label && (() => {
            // Clicking the step heading jumps to its rendered script in the Render
            // tab. Visually unchanged -- only the cursor becomes a pointer.
            const canLink = !!onOpenRendered && g.renderIndex != null && g.renderIndex >= 0
            return (
            <div
              className={cn('flex items-center gap-2 px-0.5', canLink && 'cursor-pointer')}
              onClick={canLink ? () => onOpenRendered!(g.renderIndex!) : undefined}
              title={canLink ? 'View the rendered script for this step' : undefined}
            >
              <span className="text-[11px] font-semibold uppercase tracking-wide text-fg-subtle">{g.label}</span>
              {g.milestone && (
                <span className="inline-flex shrink-0 items-center gap-1 text-[11px] text-accent" title="Other hosts can wait_for this milestone">
                  <Flag size={11} /> {g.milestone}
                </span>
              )}
              <span className="h-px flex-1 bg-border" />
              {onRestartFrom && g.label && (
                <button
                  type="button"
                  onClick={(e) => { e.stopPropagation(); onRestartFrom(g.key) }}
                  title="Run this step and every step after it again, in full, on the box as it is"
                  className={cn(
                    'inline-flex shrink-0 items-center gap-1 rounded-token-sm px-1.5 py-0.5 text-[11px] hover:bg-surface-hover',
                    g.steps.some((s) => s.status === 'failed') ? 'text-danger' : 'text-fg-subtle hover:text-fg',
                  )}
                >
                  <RotateCcw size={11} /> Restart from here
                </button>
              )}
              {(() => {
                const ms = groupDurationMs(g.steps)
                return ms != null ? <span className="shrink-0 font-mono text-[11px] tabular-nums text-fg-subtle">{formatStepDuration(ms)}</span> : null
              })()}
            </div>
            )
          })()}
          {g.waitFor && g.waitFor.length > 0 && (
            <div
              className={cn(
                'flex items-center gap-1.5 px-0.5 text-[11px]',
                g.steps.some((s) => s.status === 'blocked') ? 'text-warning' : 'text-fg-subtle',
              )}
            >
              <Hourglass size={11} className="shrink-0" />
              <span>
                {g.steps.some((s) => s.status === 'blocked') ? 'Waits for' : 'Waited for'}{' '}
                <span className="font-mono">{g.waitFor.join(', ')}</span>
                {g.steps.some((s) => s.status === 'blocked') && ' before this and every later step run'}
              </span>
            </div>
          )}
          {g.steps.map((s) => (
            <StepRow key={s.step_index} step={s} expanded={open.has(s.step_index)} onToggle={() => toggle(s.step_index)} />
          ))}
        </div>
      ))}
    </div>
  )
}

function StepRow({ step: s, expanded, onToggle }: { step: StepStatus; expanded: boolean; onToggle: () => void }) {
  const hasDetail = !!s.output || !!s.last_error || s.validators.length > 0 || s.attempts > 1
  return (
    <div className="overflow-hidden rounded-token border border-border">
      <button
        onClick={() => hasDetail && onToggle()}
        disabled={!hasDetail}
        className={cn('flex w-full items-center gap-2 bg-surface-sunken px-2 py-1.5 text-left text-xs', hasDetail && 'hover:bg-surface-hover')}
      >
        {hasDetail ? (
          <ChevronDown size={12} className={cn('shrink-0 text-fg-subtle transition-transform', !expanded && '-rotate-90')} />
        ) : (
          <span className="w-3 shrink-0" />
        )}
        <span className="text-fg-muted">#{s.step_index}</span>
        <span className="truncate font-mono font-medium text-fg">{s.command}</span>
        <span className="ml-auto shrink-0">
          <StepBadge status={s.status} />
        </span>
      </button>
      {expanded && hasDetail && (
        <div className="flex flex-col gap-2 border-t border-border px-2 py-2 text-xs">
          {s.attempts > 1 && <div className="text-fg-subtle">{s.attempts} attempts</div>}
          {s.output && (
            <div>
              <div className="mb-1 text-[11px] font-semibold uppercase tracking-wide text-fg-subtle">Output</div>
              <pre className="max-h-72 overflow-auto whitespace-pre-wrap rounded-token bg-surface-sunken p-2 font-mono text-fg">{s.output}</pre>
            </div>
          )}
          {s.last_error && (
            <div>
              <div className="mb-1 text-[11px] font-semibold uppercase tracking-wide text-danger">Error</div>
              <pre className="max-h-48 overflow-auto whitespace-pre-wrap rounded-token border border-danger/30 bg-danger-soft p-2 font-mono text-danger">{s.last_error}</pre>
            </div>
          )}
          {s.validators.length > 0 && (
            <div className="flex flex-col gap-0.5">
              {s.validators.map((v, i) => (
                <div key={i} className={cn('flex items-start gap-1.5', v.passed ? 'text-success' : 'text-danger')}>
                  {v.passed ? <ShieldCheck size={12} className="mt-0.5 shrink-0" /> : <ShieldOff size={12} className="mt-0.5 shrink-0" />}
                  <span className="font-medium">{v.kind}</span>
                  {v.message && <span className="text-fg-muted">— {v.message}</span>}
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  )
}

// InfoTab is the host-detail "Identity" + "Agent" summary.
// It renders from the DeployedObject row the caller already
// has -- lifecycle status, live power, agent check-in, network, and the
// hoster's own external ref. Address/image/size are content-resolved and not
// stored on the row, so they're deliberately omitted here rather than shown
// blank; the Render tab shows exactly what was delivered.
function InfoTab({
  buildId,
  objectId,
  object,
  kind,
  label,
  teamNumber,
  fmtTime,
}: {
  buildId: string
  objectId: string
  object?: DeployedObject
  kind: ObjectKind
  label: string
  teamNumber?: number
  fmtTime: (iso: string) => string
}) {
  // Builder placement + the root password for hand access -- resolved
  // server-side from the builder-config chain (varies by builder).
  const { data: infra } = useObjectInfra(buildId, objectId, true)
  // The host's full authored config (open ports, env, vars, deps, findings) --
  // content-resolved, not on the row. Networks have none.
  const { data: config } = useObjectConfig(buildId, objectId, kind !== 'network')
  const ports = [...(config?.tcp_ports ?? []).map((p) => `${p}/tcp`), ...(config?.udp_ports ?? []).map((p) => `${p}/udp`)].join(', ')
  const kv = (o?: Record<string, string>) => (o ? Object.entries(o).map(([k, v]) => `${k}=${v}`).join(', ') : '')
  const envLabel = kv(config?.env)
  const varsLabel = kv(config?.vars)
  const tagsLabel = kv(config?.tags)
  return (
    <div className="flex flex-col gap-4">
      <dl className="flex flex-col divide-y divide-border text-sm">
        <InfoRow label="Object" value={object?.object_name ?? label} mono />
        {object?.as_name && <InfoRow label="Name" value={object.as_name} mono />}
        {teamNumber != null && <InfoRow label="Team" value={String(teamNumber)} />}
        <InfoRow label="Kind" value={kind} />
        {object?.network_name && <InfoRow label="Network" value={object.network_name} mono />}
        {config?.address && <InfoRow label="IP address" value={config.address} mono />}
        {object?.public_address && <InfoRow label="Public IP" value={object.public_address} mono />}
        {object && (
          <InfoRow label="State" valueNode={<StatusBadge status={object.status} kind={kind} blockedOn={object.blocked_on} />} />
        )}
        {object?.power_state && !GONE_STATES.has(object.status) ? <InfoRow label="Infra" valueNode={<PowerBadge state={object.power_state} />} /> : null}
        {object?.agent && (
          <InfoRow label="Agent" valueNode={<StatusBadge status={object.agent.agent_status} />} />
        )}
        {object?.agent?.last_heartbeat_at && (
          <InfoRow label="Last check-in" value={fmtTime(object.agent.last_heartbeat_at)} />
        )}
        {object?.external_ref && <InfoRow label="Hoster ref" value={object.external_ref} mono />}
        {object?.last_error && <InfoRow label="Last error" value={object.last_error} mono danger />}
      </dl>

      {infra && infra.placement.length > 0 && (
        <div>
          <div className="mb-1.5 flex items-center gap-1.5 text-[11px] font-semibold uppercase tracking-wide text-fg-subtle">
            <ServerCog size={12} /> Builder
          </div>
          <dl className="flex flex-col divide-y divide-border text-sm">
            {infra.placement.map((f) => (
              <InfoRow key={f.label} label={f.label} value={f.value} mono />
            ))}
          </dl>
        </div>
      )}

      {config && kind !== 'network' && (
        <div>
          <div className="mb-1.5 flex items-center gap-1.5 text-[11px] font-semibold uppercase tracking-wide text-fg-subtle">
            <Info size={12} /> Configuration
          </div>
          <dl className="flex flex-col divide-y divide-border text-sm">
            {config.os && <InfoRow label="OS" value={config.os} mono />}
            {config.image && <InfoRow label="Image" value={config.image} mono />}
            {config.size && <InfoRow label="Size" value={config.size} />}
            {!!config.disk && <InfoRow label="Disk" value={`${config.disk} GB`} />}
            {config.command && config.command.length > 0 && <InfoRow label="Command" value={config.command.join(' ')} mono />}
            {ports && <InfoRow label="Open ports" value={ports} mono />}
            {config.depends_on && config.depends_on.length > 0 && <InfoRow label="Depends on" value={config.depends_on.join(', ')} mono />}
            {config.waits_for && config.waits_for.length > 0 && <InfoRow label="Waits for" value={config.waits_for.join(', ')} mono />}
            {envLabel && <InfoRow label="Environment" value={envLabel} mono />}
            {varsLabel && <InfoRow label="Vars" value={varsLabel} mono />}
            {tagsLabel && <InfoRow label="Tags" value={tagsLabel} mono />}
          </dl>
        </div>
      )}

      {/* Every account LaForge set a password for on this host, admin first;
          a host without per-account steps still shows the environment password. */}
      {infra && ((infra.credentials?.length ?? 0) > 0 || infra.password) && (
        <div>
          <div className="mb-1.5 flex items-center gap-1.5 text-[11px] font-semibold uppercase tracking-wide text-fg-subtle">
            <KeyRound size={12} /> Credentials
          </div>
          <dl className="flex flex-col divide-y divide-border text-sm">
            {infra.credentials && infra.credentials.length > 0
              ? infra.credentials.map((c) => <CredentialRow key={c.username} credential={c} />)
              : infra.password && (
                  <CredentialRow credential={{ username: 'Root password', password: infra.password, applied: true, source: 'environment', admin: true }} />
                )}
          </dl>
        </div>
      )}

      {config?.findings && config.findings.length > 0 && (
        <div>
          <div className="mb-1.5 text-[11px] font-semibold uppercase tracking-wide text-fg-subtle">Findings</div>
          <div className="flex flex-col gap-1">
            {config.findings.map((f, i) => (
              <div key={i} className="rounded-token border border-border bg-surface-sunken px-2 py-1 text-xs text-fg-muted">
                <span className="font-medium text-fg">sev {f.severity} · diff {f.difficulty}</span> — {f.description}
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}

// CredentialRow shows one account's current password for hand login, hidden by
// default with a reveal toggle and a one-click copy -- it's a real secret, so
// it isn't rendered in the clear until asked for. "Pending" marks a password
// whose step hasn't run yet (what the host is about to get, not what it has).
function CredentialRow({ credential: c }: { credential: HostCredential }) {
  const [shown, setShown] = useState(false)
  const [copied, setCopied] = useState(false)
  async function copy() {
    try {
      await navigator.clipboard.writeText(c.password)
      setCopied(true)
      setTimeout(() => setCopied(false), 1500)
    } catch {
      /* clipboard blocked -- reveal is still available */
    }
  }
  return (
    <div className="flex items-start justify-between gap-3 py-2">
      <dt className="flex shrink-0 items-center gap-1.5 text-xs font-medium text-fg-muted">
        <span className="font-mono text-fg">{c.username}</span>
        {c.admin && c.source === 'step' && <Badge tone="neutral" className="py-0 text-[10px]">Admin</Badge>}
        {!c.applied && (
          <Badge tone="warning" className="py-0 text-[10px]" title="This password's step hasn't run on the host yet">
            Pending
          </Badge>
        )}
      </dt>
      <dd className="flex min-w-0 items-center gap-1.5">
        <span className="min-w-0 break-all text-right font-mono text-xs">{shown ? c.password : '••••••••••'}</span>
        <Button variant="ghost" size="icon" onClick={() => setShown((v) => !v)} aria-label={shown ? 'Hide password' : 'Show password'} title={shown ? 'Hide' : 'Show'}>
          {shown ? <EyeOff size={13} /> : <Eye size={13} />}
        </Button>
        <Button variant="ghost" size="icon" onClick={copy} aria-label="Copy password" title="Copy">
          {copied ? <Check size={13} className="text-success" /> : <Copy size={13} />}
        </Button>
      </dd>
    </div>
  )
}

function InfoRow({
  label,
  value,
  valueNode,
  mono,
  danger,
}: {
  label: string
  value?: string
  valueNode?: React.ReactNode
  mono?: boolean
  danger?: boolean
}) {
  return (
    <div className="flex items-start justify-between gap-3 py-2">
      <dt className="shrink-0 text-xs font-medium text-fg-muted">{label}</dt>
      <dd className={cn('min-w-0 break-words text-right', mono && 'font-mono text-xs', danger && 'text-danger')}>
        {valueNode ?? value}
      </dd>
    </div>
  )
}

function TabButton({ active, onClick, icon: Icon, label }: { active: boolean; onClick: () => void; icon: typeof ScrollText; label: string }) {
  return (
    <Button variant="ghost" size="sm" onClick={onClick} className={cn('shrink-0', active && 'bg-surface-hover text-fg')}>
      <Icon size={12} /> {label}
    </Button>
  )
}

function LoadingRow() {
  return (
    <div className="flex items-center gap-2 text-sm text-fg-muted">
      <Spinner /> Loading…
    </div>
  )
}
