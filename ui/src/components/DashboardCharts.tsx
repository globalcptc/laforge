import { useState } from 'react'
import { ChevronDown, ChevronRight } from 'lucide-react'
import type { DashboardData, ResourceBucket } from '../api/types'
import { useTimeFormat } from '../lib/time'
import { StatusBadge, statusFillClass, statusLabel } from './StatusBadge'
import { Card, CardHeader, cn } from '../ui'

// The environment dashboard's real charts -- "Build → Overview: health
// band... all fed by server-side aggregates", backed
// by internal/api/dashboard.go's real aggregation, not client-side
// counting. Hand-rolled (no charting library added) -- these are the
// specific shapes the plan names, not a general-purpose chart kit.

// Colours and labels come from StatusBadge (statusFillClass/statusLabel), so
// the bar and its legend always match the badges on the host rows.
const STATUS_ORDER = [
  'pending',
  'deploying',
  'running',
  'building',
  'finished',
  'deploy_failed',
  'build_failed',
  'invalid',
  'destroying',
  'destroyed',
] as const

function ChartCard({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <Card>
      <CardHeader title={title} />
      <div className="p-4 pt-3">{children}</div>
    </Card>
  )
}

// "Provisioning progress: a stacked bar across the whole build: pending
// -> running -> provisioned, with failed called out. Fills in live
// during a deploy."
export function ProvisioningProgress({ data }: { data: DashboardData }) {
  const total = Object.values(data.provisioning_by_status).reduce((a, b) => a + b, 0)
  if (total === 0) return null
  return (
    <ChartCard title="Provisioning Progress">
      <div className="flex h-4 overflow-hidden rounded-full bg-surface-sunken">
        {STATUS_ORDER.filter((s) => data.provisioning_by_status[s] > 0).map((status) => (
          <div
            key={status}
            className={statusFillClass(status)}
            style={{ width: `${(data.provisioning_by_status[status] / total) * 100}%` }}
            title={`${statusLabel(status)}: ${data.provisioning_by_status[status]}`}
          />
        ))}
      </div>
      <div className="mt-2 flex flex-wrap gap-x-4 gap-y-1 text-xs text-fg-muted">
        {STATUS_ORDER.filter((s) => data.provisioning_by_status[s] > 0).map((status) => (
          <span key={status} className="flex items-center gap-1">
            <span className={cn('h-2 w-2 rounded-full', statusFillClass(status))} />
            {statusLabel(status)} ({data.provisioning_by_status[status]})
          </span>
        ))}
      </div>
    </ChartCard>
  )
}

// "State by team: ... surfaces the 'one team is wrong' shape here on the
// dashboard before anyone opens the host matrix." One stacked bar per team,
// split by the full host/container lifecycle -- the same statuses, colours
// and icons the rows and the provisioning bar use (statusFillClass /
// StatusBadge), so a red Creation-Failed slice here is the exact red of the
// badge you'll find when you drill in. Fed by the server-side per-team
// `by_status` aggregate (dashboard.go), not re-derived in the browser.
export function StateByTeam({ data }: { data: DashboardData }) {
  // by_team can be absent on a freshly-planned build that has deployed
  // nothing yet -- guard rather than crashing the whole Overview.
  if (!data.by_team || data.by_team.length === 0) return null
  // Only the statuses that actually occur, in lifecycle order, so a clean
  // build doesn't show a row of empty failure swatches.
  const present = STATUS_ORDER.filter((s) => data.by_team.some((t) => (t.by_status?.[s] ?? 0) > 0))
  return (
    <ChartCard title="State by Team">
      <div className="grid grid-cols-2 gap-x-4 gap-y-3 sm:grid-cols-3">
        {data.by_team.map((t) => {
          const counts = t.by_status ?? {}
          const total = STATUS_ORDER.reduce((a, s) => a + (counts[s] ?? 0), 0)
          return (
            <div key={t.team_number}>
              <div className="mb-1 flex items-baseline justify-between">
                <span className="text-xs font-medium text-fg">Team {t.team_number}</span>
                <span className="text-[10px] text-fg-muted">{total}</span>
              </div>
              <div className="flex h-3 w-full overflow-hidden rounded-full bg-surface-sunken">
                {total > 0 &&
                  STATUS_ORDER.filter((s) => (counts[s] ?? 0) > 0).map((s) => (
                    <div
                      key={s}
                      className={statusFillClass(s)}
                      style={{ width: `${((counts[s] ?? 0) / total) * 100}%` }}
                      title={`${statusLabel(s)}: ${counts[s]}`}
                    />
                  ))}
              </div>
            </div>
          )
        })}
      </div>

      {present.length > 0 && (
        <div className="mt-4 flex flex-wrap gap-1.5 border-t border-border pt-3">
          {present.map((s) => (
            <StatusBadge key={s} status={s} />
          ))}
        </div>
      )}
    </ChartCard>
  )
}

// "Failures grouped by cause: a short ranked bar: '40 hosts:
// install-mysql exit 1' as one entry, not forty. This is the thing that
// turns a wall of red into a single fixable problem."
export function FailuresByCause({ data }: { data: DashboardData }) {
  const [expanded, setExpanded] = useState<Set<number>>(new Set())
  if (data.failures_by_cause.length === 0) return null
  const maxCount = data.failures_by_cause[0].count
  return (
    <ChartCard title={`Failures grouped by cause (${data.failures_by_cause.length})`}>
      <div className="flex flex-col gap-2">
        {data.failures_by_cause.map((f, i) => {
          const isOpen = expanded.has(i)
          return (
            <div key={i}>
              <button
                onClick={() =>
                  setExpanded((prev) => {
                    const next = new Set(prev)
                    next.has(i) ? next.delete(i) : next.add(i)
                    return next
                  })
                }
                className="flex w-full items-center gap-2 text-left"
              >
                {isOpen ? <ChevronDown size={12} /> : <ChevronRight size={12} />}
                <span className="w-10 shrink-0 text-xs font-medium text-danger">{f.count}x</span>
                <div className="h-2 flex-1 overflow-hidden rounded-full bg-surface-sunken">
                  <div className="h-full bg-danger" style={{ width: `${(f.count / maxCount) * 100}%` }} />
                </div>
                <span className="max-w-[50%] truncate text-xs text-fg-muted">{f.message}</span>
              </button>
              {isOpen && (
                <div className="ml-16 mt-1 flex flex-wrap gap-1">
                  {f.objects.map((name) => (
                    <span key={name} className="rounded bg-surface-sunken px-1.5 py-0.5 text-[10px] text-fg-muted">
                      {name}
                    </span>
                  ))}
                </div>
              )}
            </div>
          )
        })}
      </div>
    </ChartCard>
  )
}

// "Agent check-ins over time: a line chart, so a cliff (a network
// dropping, a team going dark) is visible the instant it happens rather
// than inferred from a count." Real, genuinely ongoing data as of the
// append-only agent_heartbeat log (migrations/00007) -- each point is
// how many DISTINCT objects had at least one real heartbeat in that
// window (internal/api/dashboard.go's bucketHeartbeats), computed
// server-side. Capable of actually showing a cliff, unlike the earlier
// first-seen-only cumulative curve this replaced: a batch of agents
// going quiet shows up as Active dropping, verified in
// TestDashboardAggregatesRealData against a real staggered heartbeat
// history where one object genuinely stops reporting partway through.
export function AgentActivity({ data }: { data: DashboardData }) {
  const fmt = useTimeFormat()
  const buckets = data.agent_activity
  const height = 80
  if (buckets.length === 0) {
    return (
      <ChartCard title="Agent Activity (Distinct Check-Ins)">
        <div className="flex items-center justify-center text-xs text-fg-subtle" style={{ height }}>
          No heartbeats yet
        </div>
      </ChartCard>
    )
  }

  const width = 400
  const maxActive = Math.max(1, ...buckets.map((b) => b.active))
  const stepX = buckets.length > 1 ? width / (buckets.length - 1) : 0

  const path = buckets
    .map((b, i) => `${i === 0 ? 'M' : 'L'} ${i * stepX} ${height - (b.active / maxActive) * height}`)
    .join(' ')

  const last = buckets[buckets.length - 1]
  const peak = buckets.find((b) => b.active === maxActive)!
  const droppedFromPeak = peak.active - last.active

  const fmtTime = (iso: string) => fmt.timeShort(iso)
  const midY = height / 2
  const midValue = Math.round(maxActive / 2)

  return (
    <ChartCard title="Agent Activity (Distinct Check-Ins)">
      <div className="flex gap-2">
        {/* Y-axis: active-object count, max at top down to 0. */}
        <div className="flex shrink-0 flex-col justify-between text-right text-[10px] text-fg-subtle" style={{ height }}>
          <span>{maxActive}</span>
          {midValue !== maxActive && midValue !== 0 && <span>{midValue}</span>}
          <span>0</span>
        </div>
        <div className="min-w-0 flex-1">
          <svg viewBox={`0 0 ${width} ${height}`} preserveAspectRatio="none" className="w-full" style={{ height }}>
            {/* Baseline + midline gridlines so the curve reads against a scale. */}
            <line x1={0} y1={height} x2={width} y2={height} stroke="var(--color-border)" strokeWidth={1} vectorEffect="non-scaling-stroke" />
            <line x1={0} y1={midY} x2={width} y2={midY} stroke="var(--color-border)" strokeWidth={1} strokeDasharray="3 3" vectorEffect="non-scaling-stroke" />
            <path d={path} fill="none" stroke="var(--color-accent)" strokeWidth={2} vectorEffect="non-scaling-stroke" />
          </svg>
          {/* X-axis: the time span the buckets cover. */}
          <div className="mt-1 flex justify-between text-[10px] text-fg-subtle">
            <span>{fmtTime(buckets[0].at)}</span>
            <span>{fmtTime(last.at)}</span>
          </div>
        </div>
      </div>
      <div className="mt-1 text-xs text-fg-muted">
        {last.active} active now (peak {maxActive})
        {droppedFromPeak > 0 && <span className="text-warning"> — {droppedFromPeak} fewer than peak</span>}
      </div>
    </ChartCard>
  )
}

// Build-wide average host metrics over time: one small line graph per resource,
// fed by GET /builds/{id}/dashboard's resource_usage buckets. Percentages
// (CPU/mem/disk) scale 0-100; the network graph auto-scales to its own peak and
// draws download and upload as two lines.
function ResourceLineChart({
  title,
  buckets,
  series,
  yMax,
  fmtValue,
}: {
  title: string
  buckets: ResourceBucket[]
  series: { values: number[]; color: string; label: string }[]
  yMax: number
  fmtValue: (n: number) => string
}) {
  const fmt = useTimeFormat()
  const height = 72
  // Keep the card present before any heartbeats arrive, so the row doesn't look
  // broken mid-build -- just say there's no data yet.
  if (buckets.length === 0) {
    return (
      <ChartCard title={title}>
        <div className="flex items-center justify-center text-xs text-fg-subtle" style={{ height }}>
          No heartbeats yet
        </div>
      </ChartCard>
    )
  }

  const width = 400
  const max = Math.max(yMax, 1)
  const stepX = buckets.length > 1 ? width / (buckets.length - 1) : 0
  const midY = height / 2
  const pathFor = (values: number[]) =>
    values.map((v, i) => `${i === 0 ? 'M' : 'L'} ${i * stepX} ${height - (Math.min(v, max) / max) * height}`).join(' ')
  const last = buckets[buckets.length - 1]

  return (
    <ChartCard title={title}>
      <div className="flex gap-2">
        <div className="flex shrink-0 flex-col justify-between text-right text-[10px] text-fg-subtle" style={{ height }}>
          <span>{fmtValue(max)}</span>
          <span>{fmtValue(max / 2)}</span>
          <span>{fmtValue(0)}</span>
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
            <span>{fmt.timeShort(buckets[0].at)}</span>
            <span>{fmt.timeShort(last.at)}</span>
          </div>
        </div>
      </div>
      <div className="mt-1 flex flex-wrap gap-x-3 text-xs text-fg-muted">
        {series.map((s) => (
          <span key={s.label} className="inline-flex items-center gap-1">
            <span className="inline-block h-2 w-2 rounded-full" style={{ background: s.color }} />
            {s.label} {fmtValue(s.values[s.values.length - 1] ?? 0)}
          </span>
        ))}
      </div>
    </ChartCard>
  )
}

const pct = (n: number) => `${Math.round(n)}%`

function fmtBps(n: number): string {
  if (n < 1024) return `${Math.round(n)} B/s`
  if (n < 1024 * 1024) return `${Math.round(n / 1024)} KB/s`
  return `${(n / (1024 * 1024)).toFixed(1)} MB/s`
}

export function ResourceCpu({ data }: { data: DashboardData }) {
  const b = data.resource_usage
  return <ResourceLineChart title="Average CPU" buckets={b} yMax={100} fmtValue={pct} series={[{ values: b.map((x) => x.cpu), color: 'var(--color-accent)', label: 'CPU' }]} />
}

export function ResourceMem({ data }: { data: DashboardData }) {
  const b = data.resource_usage
  return <ResourceLineChart title="Average Memory" buckets={b} yMax={100} fmtValue={pct} series={[{ values: b.map((x) => x.mem), color: 'var(--color-purple)', label: 'Memory' }]} />
}

export function ResourceDisk({ data }: { data: DashboardData }) {
  const b = data.resource_usage
  return <ResourceLineChart title="Average Disk" buckets={b} yMax={100} fmtValue={pct} series={[{ values: b.map((x) => x.disk), color: 'var(--color-warning)', label: 'Disk' }]} />
}

export function ResourceNet({ data }: { data: DashboardData }) {
  const b = data.resource_usage
  const yMax = Math.max(1, ...b.map((x) => Math.max(x.net_rx, x.net_tx)))
  return (
    <ResourceLineChart
      title="Average Network"
      buckets={b}
      yMax={yMax}
      fmtValue={fmtBps}
      series={[
        { values: b.map((x) => x.net_rx), color: 'var(--color-success)', label: 'Down' },
        { values: b.map((x) => x.net_tx), color: 'var(--color-sky)', label: 'Up' },
      ]}
    />
  )
}
