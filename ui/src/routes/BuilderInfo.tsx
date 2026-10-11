import { Link, useParams } from '@tanstack/react-router'
import { Pencil, TriangleAlert } from 'lucide-react'
import { useBuilderStats, useBuilderUsage, useBuilderConfigs } from '../api/hooks'
import type { MetricSeries, MetricGauge, UsageResource, UsagePublic, UsageBuild } from '../api/types'
import { BuilderKindBadge } from '../components/BuilderKindBadge'
import { StatusBadge } from '../components/StatusBadge'
import { Card, CardHeader, PageHeader, EmptyState, buttonClass, cn } from '../ui'

// The builder info page: backend-health charts + quota gauges, rendered
// generically from whatever stats the builder publishes (so adding a metric in a
// builder needs no UI change). The edit flow lives behind the Edit button.
// See docs/plans/builder-monitoring.md.
export function BuilderInfo() {
  const { name } = useParams({ from: '/admin/infrastructure/$name' })
  const configs = useBuilderConfigs()
  const bc = configs.data?.find((c) => c.name === name)
  const stats = useBuilderStats(name)
  const usage = useBuilderUsage(name)

  const series = stats.data?.series ?? []
  const gauges = stats.data?.gauges ?? []
  const warnings = stats.data?.warnings ?? []

  // Each edit action gets its own color so the three read apart at a glance (and
  // never the neutral/white `secondary` default). See feedback-buttons-use-color.
  const editBtn = (label: string, section: string, variant: 'brand' | 'primary' | 'coral') => (
    <Link
      to="/admin/infrastructure/$name/edit"
      params={{ name }}
      search={{ section }}
      className={buttonClass({ variant, size: 'sm' })}
    >
      <Pencil size={13} /> {label}
    </Link>
  )

  return (
    <div className="mx-auto max-w-5xl px-4 py-6">
      <PageHeader
        title={name}
        description="Backend health and capacity for this builder."
        actions={
          <>
            {editBtn('Config', 'config', 'brand')}
            {editBtn('Images', 'images', 'primary')}
            {editBtn('Sizes', 'sizes', 'coral')}
          </>
        }
      />
      {bc && (
        <div className="mb-4">
          <BuilderKindBadge kind={bc.kind} />
        </div>
      )}

      {/* Status spans the full width -- it's the at-a-glance summary. */}
      <div className="mb-4">
        <StatusPanel builds={usage.data?.active_builds ?? []} resources={usage.data?.resources ?? []} />
      </div>

      {warnings.length > 0 && (
        <div className="mb-4 flex items-start gap-2.5 rounded-token border border-warning/30 bg-warning-soft px-3.5 py-2.5 text-sm text-warning">
          <TriangleAlert size={16} className="mt-0.5 shrink-0" />
          <div className="min-w-0">
            <div className="font-medium">Some metrics couldn’t be read</div>
            <ul className="mt-1 list-disc space-y-0.5 pl-4 text-warning/90">
              {warnings.map((warn, i) => (
                <li key={i} className="break-words">{warn}</li>
              ))}
            </ul>
          </div>
        </div>
      )}

      {gauges.length > 0 && (
        <Card className="mb-4">
          <CardHeader title="Quotas" />
          <div className="grid grid-cols-1 gap-3 p-4 sm:grid-cols-2">
            {gauges.map((g) => (
              <GaugeBar key={g.key + g.node} gauge={g} />
            ))}
          </div>
        </Card>
      )}

      {series.length === 0 ? (
        <EmptyState
          title={stats.isLoading ? 'Loading metrics…' : 'No metrics for this builder'}
          description={
            stats.isLoading
              ? ''
              : 'This builder reports no backend health, or its stack is unreachable right now. OpenStack does not report host metrics.'
          }
        />
      ) : (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          {groupByKey(series).map((group) => (
            <MetricChart key={group.key} label={group.label} unit={group.unit} lines={group.lines} />
          ))}
        </div>
      )}

      {/* Public endpoints last: a full-width list grouped by team. */}
      <div className="mt-4">
        <PublicPanel endpoints={usage.data?.public ?? []} />
      </div>
    </div>
  )
}

// The resource kinds a builder can deploy. Always shown (even at 0/0) so an idle
// builder reads as "nothing in use" rather than hiding the counts entirely --
// seeing a capacity sit empty is itself useful.
const RESOURCE_KINDS = ['host', 'container', 'network'] as const

// mergeResourceKinds overlays the live counts onto the canonical kind list, so
// every kind always appears -- unused ones as 0/0. Any extra kind the backend
// reports (e.g. dns) is appended after the canonical three.
function mergeResourceKinds(resources: UsageResource[]): UsageResource[] {
  const byKind = new Map(resources.map((r) => [r.kind, r]))
  const base = RESOURCE_KINDS.map((kind) => byKind.get(kind) ?? { kind, running: 0, total: 0 })
  const extra = resources.filter((r) => !RESOURCE_KINDS.includes(r.kind as (typeof RESOURCE_KINDS)[number]))
  return [...base, ...extra]
}

// StatusPanel shows the builder's current load: how many builds are live on it
// and how many resources of each kind are up.
function StatusPanel({ builds, resources }: { builds: UsageBuild[]; resources: UsageResource[] }) {
  return (
    <Card>
      <CardHeader title="Status" />
      <div className="p-4 pt-3 text-sm">
        <div className="mb-3">
          <span className="font-mono text-lg text-fg">{builds.length}</span>{' '}
          <span className="text-fg-muted">active build{builds.length === 1 ? '' : 's'}</span>
        </div>
        <div className="flex flex-wrap gap-x-5 gap-y-1">
          {mergeResourceKinds(resources).map((r) => (
            <span key={r.kind}>
              <span className="text-fg-muted">{r.kind}s </span>
              <span className="font-mono text-fg">{r.running}</span>
              <span className="text-fg-subtle">/{r.total}</span>
            </span>
          ))}
        </div>
        {builds.length > 0 && (
          <ul className="mt-3 space-y-1 border-t border-border pt-2 text-xs">
            {builds.slice(0, 6).map((b) => (
              <li key={b.id} className="flex items-center justify-between gap-2">
                <span className="truncate text-fg-muted">{b.environment_name}</span>
                <StatusBadge status={b.status} />
              </li>
            ))}
          </ul>
        )}
      </div>
    </Card>
  )
}

// PublicPanel lists the externally-reachable endpoints across the builder's
// builds -- the address:port a team connects to for a public: host -- grouped by
// team so each team's exposed surface reads together.
function PublicPanel({ endpoints }: { endpoints: UsagePublic[] }) {
  const teams = groupByTeam(endpoints)
  return (
    <Card>
      <CardHeader title="Public endpoints" />
      <div className="p-4 pt-3 text-sm">
        {endpoints.length === 0 ? (
          <div className="text-xs text-fg-subtle">No public endpoints exposed.</div>
        ) : (
          <div className="grid grid-cols-1 gap-x-8 gap-y-4 sm:grid-cols-2">
            {teams.map((t) => (
              <div key={t.team}>
                <div className="mb-1 text-xs font-medium uppercase tracking-wide text-fg-muted">Team {t.team}</div>
                <ul className="space-y-1">
                  {t.endpoints.map((e, i) => (
                    <li key={i} className="flex items-center justify-between gap-2">
                      <span className="truncate text-fg-muted">{e.as_name || e.object_name}</span>
                      <span className="font-mono text-xs text-fg">{e.public_address}</span>
                    </li>
                  ))}
                </ul>
              </div>
            ))}
          </div>
        )}
      </div>
    </Card>
  )
}

// groupByTeam buckets endpoints under their team number, teams in ascending order.
function groupByTeam(endpoints: UsagePublic[]): { team: number; endpoints: UsagePublic[] }[] {
  const byTeam = new Map<number, UsagePublic[]>()
  for (const e of endpoints) {
    const list = byTeam.get(e.team_number) ?? []
    list.push(e)
    byTeam.set(e.team_number, list)
  }
  return [...byTeam.entries()].sort((a, b) => a[0] - b[0]).map(([team, eps]) => ({ team, endpoints: eps }))
}

// GaugeBar is a quota usage-vs-limit bar; it reddens as it nears the limit.
function GaugeBar({ gauge }: { gauge: MetricGauge }) {
  const max = gauge.max ?? 0
  const pct = max > 0 ? Math.min(100, (gauge.value / max) * 100) : 0
  const tone = pct >= 90 ? 'bg-danger' : pct >= 75 ? 'bg-warning' : 'bg-accent'
  return (
    <div>
      <div className="mb-1 flex items-baseline justify-between text-xs">
        <span className="text-fg">{gauge.label}</span>
        <span className="font-mono text-fg-muted">
          {fmt(gauge.value)} / {max > 0 ? fmt(max) : '—'}
        </span>
      </div>
      <div className="h-2 overflow-hidden rounded-token bg-surface-sunken">
        <div className={cn('h-full rounded-token', tone)} style={{ width: `${pct}%` }} />
      </div>
    </div>
  )
}

interface chartLine {
  node: string
  points: number[]
}
interface chartGroup {
  key: string
  label: string
  unit: string
  lines: chartLine[]
}

// groupByKey collapses per-node series into one chart per metric key, each with
// a line per node.
function groupByKey(series: MetricSeries[]): chartGroup[] {
  const byKey = new Map<string, chartGroup>()
  for (const s of series) {
    let g = byKey.get(s.key)
    if (!g) {
      g = { key: s.key, label: s.label, unit: s.unit, lines: [] }
      byKey.set(s.key, g)
    }
    g.lines.push({ node: s.node, points: s.points.map((p) => p.v) })
  }
  return [...byKey.values()]
}

// MetricChart draws a small multi-line sparkline, one polyline per node, scaled
// to the group's own min/max. Hand-rolled SVG, matching DashboardCharts (no chart
// library). Latest value per line is listed below.
function MetricChart({ label, unit, lines }: { label: string; unit: string; lines: chartLine[] }) {
  const all = lines.flatMap((l) => l.points)
  const min = all.length ? Math.min(...all) : 0
  const max = all.length ? Math.max(...all) : 1
  const span = max - min || 1
  const width = 300
  const height = 80
  const palette = ['var(--color-accent)', 'var(--color-brand)', 'var(--color-success)', 'var(--color-warning)', 'var(--color-info)']

  const latest = (pts: number[]) => (pts.length ? pts[pts.length - 1] : 0)

  return (
    <Card>
      <CardHeader title={`${label}${unit ? ` (${unit})` : ''}`} />
      <div className="p-4 pt-3">
        <svg viewBox={`0 0 ${width} ${height}`} preserveAspectRatio="none" className="w-full" style={{ height }}>
          {lines.map((l, i) => {
            if (l.points.length === 0) return null
            const y = (v: number) => (height - ((v - min) / span) * height).toFixed(1)
            // A single sample (e.g. the first live reading before any history is
            // collected) still gets a flat line across the card, so the block is
            // visibly "here, currently N" rather than blank.
            const d =
              l.points.length === 1
                ? `M 0 ${y(l.points[0])} L ${width} ${y(l.points[0])}`
                : l.points
                    .map((v, j) => `${j === 0 ? 'M' : 'L'} ${((j * width) / (l.points.length - 1)).toFixed(1)} ${y(v)}`)
                    .join(' ')
            return <path key={l.node || i} d={d} fill="none" stroke={palette[i % palette.length]} strokeWidth={2} vectorEffect="non-scaling-stroke" />
          })}
        </svg>
        <div className="mt-2 flex flex-wrap gap-x-4 gap-y-1 text-xs">
          {lines.map((l, i) => (
            <span key={l.node || i} className="inline-flex items-center gap-1.5">
              <span className="inline-block h-2 w-2 rounded-full" style={{ background: palette[i % palette.length] }} />
              <span className="text-fg-muted">{l.node || 'builder'}</span>
              <span className="font-mono text-fg">{fmt(latest(l.points))}</span>
            </span>
          ))}
        </div>
      </div>
    </Card>
  )
}

function fmt(n: number): string {
  if (n >= 1000) return n.toLocaleString(undefined, { maximumFractionDigits: 0 })
  if (n === Math.round(n)) return String(n)
  return n.toFixed(1)
}
