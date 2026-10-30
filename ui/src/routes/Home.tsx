import type { ReactNode } from 'react'
import { Link } from '@tanstack/react-router'
import { AlertTriangle, Boxes, ChevronRight, FolderGit2, GitCommit, Network, Server, ServerCog, Users, X } from 'lucide-react'
import { useDismissAttention, useHome, useMe } from '../api/hooks'
import type { AttentionItem, HomeBuild, HomeCapacity, HomeCounts, HomeData } from '../api/types'
import { AccessCountdown } from '../components/AccessCountdown'
import { EmptyState } from '../components/EmptyState'
import { StatusBadge } from '../components/StatusBadge'
import { BuilderPill } from '../components/BuilderPill'
import { KIND_LABEL, type Kind } from '../components/builder-wizard/model'
import { Button, Card, PageHeader, Spinner, cn } from '../ui'

// Home: what is live right now and whether it's healthy,
// without walking into each repository. Totals across everything, anything
// failing, how much each builder is carrying, then every active build
// grouped by repository. Every number is the server's aggregate
// (GET /home) and every card clicks through to the build.
export function Home() {
  const { data, isLoading, error } = useHome()

  return (
    <>
      <PageHeader title="Home" description="What's live right now, across every repository you can access." />
      <div className="page-body flex flex-col gap-6">
        {isLoading && (
          <div className="flex items-center gap-2 text-sm text-fg-muted">
            <Spinner /> Loading…
          </div>
        )}
        {error && <div className="card p-4 text-sm text-danger">Couldn't load Home.</div>}
        {data && <Dashboard data={data} />}
      </div>
    </>
  )
}

function Dashboard({ data }: { data: HomeData }) {
  // The home page is a cross-build overview, not a build view: per-build health
  // (failures, outstanding steps, agent status) is noise here -- it's loud and
  // meaningless while many builds are mid-test. So this shows only what belongs
  // at the instance level: infrastructure usage and the active builds.
  const buildersInUse = data.builders.filter((b) => b.active_builds > 0).length

  return (
    <>
      {data.attention.length > 0 && (
        <div className="rounded-token border border-danger/30 bg-danger-soft">
          <div className="flex items-center gap-2 px-4 py-2 text-sm font-medium text-danger">
            <AlertTriangle size={14} /> Needs attention
          </div>
          <div className="divide-y divide-danger/20 border-t border-danger/20">
            {data.attention.map((a) => (
              <AttentionRow key={`${a.build_id}:${a.category}`} item={a} />
            ))}
          </div>
        </div>
      )}

      {data.builders.length > 0 && <BuilderUsage builders={data.builders} buildersInUse={buildersInUse} />}

      {data.repositories.length === 0 ? (
        <EmptyState
          icon={Boxes}
          title="Nothing is deployed right now"
          hint="Active builds show up here, grouped by repository, as soon as one starts deploying."
          action={
            <Link to="/repos" className="text-sm font-medium text-accent hover:underline">
              Go to repositories
            </Link>
          }
        />
      ) : (
        data.repositories.map((repo) => (
          <section key={repo.id}>
            <Link to="/repos/$repoId" params={{ repoId: repo.id }} className="mb-2 inline-flex items-center gap-2 text-sm font-semibold text-fg hover:text-accent">
              <FolderGit2 size={14} className="text-fg-muted" />
              {repo.github_owner}/{repo.github_repo}
              <span className="text-xs font-normal text-fg-muted">
                {repo.builds.length} active build{repo.builds.length === 1 ? '' : 's'}
              </span>
            </Link>
            <div className="grid gap-3 lg:grid-cols-2">
              {repo.builds.map((b) => (
                <BuildCard key={b.id} repoId={repo.id} build={b} />
              ))}
            </div>
          </section>
        ))
      )}
    </>
  )
}

// AttentionRow is one needs-attention item: it clicks through to the build,
// with a close button that dismisses the item for this person (server-side, so
// it stays closed). Close is a sibling of the Link, not nested inside it, so a
// click on the X never also navigates.
function AttentionRow({ item }: { item: AttentionItem }) {
  const dismiss = useDismissAttention()
  return (
    <div className="flex items-center gap-3 px-4 py-2 text-sm text-fg hover:bg-danger/10">
      <Link
        to="/repos/$repoId/builds/$buildId"
        params={{ repoId: item.repository_id, buildId: item.build_id }}
        className="flex min-w-0 flex-1 items-center gap-3"
      >
        <span className="font-medium">{item.reason}</span>
        <span className="truncate text-fg-muted">
          {item.repository} · {item.environment_name}
        </span>
        <ChevronRight size={14} className="ml-auto shrink-0 text-fg-subtle" />
      </Link>
      <Button
        variant="ghost"
        size="icon"
        aria-label="Close this item"
        title="Close"
        disabled={dismiss.isPending}
        onClick={() => dismiss.mutate({ buildId: item.build_id, category: item.category, dismissed: true })}
      >
        <X size={14} />
      </Button>
    </div>
  )
}

const fmtNum = (n: number) => (Number.isInteger(n) ? n.toLocaleString() : n.toFixed(1))

// CapacityCell is one build-gating signal, kept small enough that a builder's
// two heaviest signals sit in their own fixed columns: a short label, a thin
// bar and the value. The node (and the full label) live in the tooltip; the
// builder page has the full set.
function CapacityCell({ sig, className }: { sig: HomeCapacity | undefined; className?: string }) {
  if (!sig) return <span className={className} />
  const pct = Math.min(100, Math.max(0, sig.pct))
  const tone = sig.pct >= 90 ? 'bg-danger' : sig.pct >= 75 ? 'bg-warning' : 'bg-accent'
  const title = sig.node ? `${sig.label} · ${sig.node}` : sig.label
  const value = sig.max != null ? `${fmtNum(sig.value)}/${fmtNum(sig.max)}` : `${sig.pct.toFixed(0)}%`
  return (
    <div className={cn('flex items-center gap-1.5 whitespace-nowrap text-[11px]', className)} title={`${title}: ${value}`}>
      <span className="hidden w-20 truncate text-fg-muted @2xl:block">{sig.label}</span>
      <div className="h-1 w-10 shrink-0 overflow-hidden rounded-full bg-surface-sunken">
        <div className={cn('h-full rounded-full', tone)} style={{ width: `${pct}%` }} />
      </div>
      <span className="w-12 text-right font-mono tabular-nums text-fg">{value}</span>
    </div>
  )
}

// The usage table's columns: builder, its two heaviest signals, the counts and
// a chevron. Every row is a subgrid of this one grid, so each column lines up
// across builders. Columns drop off as the CARD narrows -- container queries,
// not viewport breakpoints, since the sidebar eats into the page -- so a row
// never wraps: the second signal and labels, then the counts, then
// containers/networks.
const USAGE_COLS =
  'grid-cols-[minmax(max-content,1fr)_auto_auto] @2xl:grid-cols-[minmax(max-content,1fr)_auto_auto_auto] @4xl:grid-cols-[minmax(max-content,1fr)_auto_auto_repeat(3,auto)_auto] @5xl:grid-cols-[minmax(max-content,1fr)_auto_auto_repeat(5,auto)_auto]'
// Builds carries extra left padding so the counts read as their own group,
// apart from the capacity columns.
const COUNT_COLS: { label: string; className: string; value: (b: HomeData['builders'][number]) => number }[] = [
  { label: 'Builds', className: 'hidden pl-8 @4xl:block', value: (b) => b.active_builds },
  { label: 'Teams', className: 'hidden @4xl:block', value: (b) => b.teams },
  { label: 'Hosts', className: 'hidden @4xl:block', value: (b) => b.counts.hosts },
  { label: 'Containers', className: 'hidden @5xl:block', value: (b) => b.counts.containers },
  { label: 'Networks', className: 'hidden @5xl:block', value: (b) => b.counts.networks },
]

function BuilderUsage({ builders, buildersInUse }: { builders: HomeData['builders']; buildersInUse: number }) {
  const { data: me } = useMe()
  const rowClass = 'col-span-full grid grid-cols-subgrid items-center gap-x-3 px-4 py-2 @sm:gap-x-5'
  return (
    <Card>
      <div className="flex items-center justify-between border-b border-border px-4 py-2.5">
        <div className="flex items-center gap-2 text-sm font-semibold text-fg">
          <ServerCog size={14} className="text-fg-muted" /> Infrastructure Usage
        </div>
        <span className="text-xs text-fg-muted">
          {buildersInUse} of {builders.length} in use
        </span>
      </div>
      <div className="@container">
        <div className={cn('grid', USAGE_COLS)}>
          <div className={cn(rowClass, 'border-b border-border py-1.5 text-[11px] text-fg-subtle')}>
            <span>Builder</span>
            <span className="@2xl:col-span-2">Top Capacity</span>
            {COUNT_COLS.map((c) => (
              <span key={c.label} className={cn('text-right', c.className)}>
                {c.label}
              </span>
            ))}
            <span />
          </div>
          {builders.map((b, i) => {
            // The two heaviest-used signals; the rest are on the builder page.
            const topTwo = [...(b.capacity ?? [])].sort((a, z) => z.pct - a.pct).slice(0, 2)
            const cells = (
              <>
                <div className="flex items-center gap-2 whitespace-nowrap">
                  <BuilderPill name={b.name} color={b.color} kind={b.kind} />
                  {/* The kind label is the first thing to go on a phone-width card. */}
                  <span className="hidden text-xs text-fg-muted @md:inline">{KIND_LABEL[b.kind as Kind] ?? b.kind}</span>
                </div>
                {topTwo.length > 0 ? (
                  <>
                    <CapacityCell sig={topTwo[0]} />
                    <CapacityCell sig={topTwo[1]} className="hidden @2xl:flex" />
                  </>
                ) : (
                  <span className="whitespace-nowrap text-[11px] text-fg-subtle @2xl:col-span-2">No metrics yet</span>
                )}
                {COUNT_COLS.map((c) => (
                  <span
                    key={c.label}
                    className={cn('text-right font-mono text-xs tabular-nums text-fg', c.className)}
                  >
                    {c.value(b)}
                  </span>
                ))}
                {me?.is_instance_admin ? <ChevronRight size={14} className="text-fg-subtle" /> : <span />}
              </>
            )
            const border = i > 0 ? 'border-t border-border' : ''
            // The whole row opens the builder page for admins (the only ones who
            // can see it); for everyone else it's a plain row.
            return me?.is_instance_admin ? (
              <Link
                key={b.name}
                to="/admin/infrastructure/$name"
                params={{ name: b.name }}
                className={cn(rowClass, border, 'hover:bg-surface-hover')}
              >
                {cells}
              </Link>
            ) : (
              <div key={b.name} className={cn(rowClass, border)}>
                {cells}
              </div>
            )
          })}
        </div>
      </div>
    </Card>
  )
}

function BuildCard({ repoId, build }: { repoId: string; build: HomeBuild }) {
  const c: HomeCounts = build.counts
  const agents = c.agents_healthy + c.agents_late + c.agents_missing + c.agents_booting
  return (
    <Link
      to="/repos/$repoId/builds/$buildId"
      params={{ repoId, buildId: build.id }}
      className="card flex flex-col gap-3 p-4 hover:border-border-strong"
    >
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="truncate font-medium text-fg">{build.environment_name}</div>
          <div className="mt-0.5 flex flex-wrap items-center gap-x-3 gap-y-0.5 text-xs text-fg-muted">
            <span className="flex items-center gap-1 font-mono">
              <GitCommit size={11} />
              {build.commit_sha.slice(0, 7)}
            </span>
            {build.ref && <span className="truncate">{build.ref.replace(/^refs\/heads\//, '')}</span>}
            {build.builder_config_name && (
              <span className="flex items-center gap-1">
                <ServerCog size={11} />
                {build.builder_config_name}
              </span>
            )}
          </div>
        </div>
        <StatusBadge status={build.status} />
      </div>

      <div className="grid grid-cols-2 gap-2 text-center sm:grid-cols-4">
        <Metric icon={<Users size={12} />} label="Teams" value={build.teams ? `${build.teams_open}/${build.teams} open` : '—'} />
        <Metric icon={<Server size={12} />} label="Hosts" value={c.hosts} />
        <Metric icon={<Boxes size={12} />} label="Containers" value={c.containers} />
        <Metric icon={<Network size={12} />} label="Networks" value={c.networks} />
      </div>

      <div className="flex flex-wrap items-center gap-x-4 gap-y-1 border-t border-border pt-2 text-xs">
        <span className="text-fg-muted">
          Agents{' '}
          {agents === 0 ? (
            '—'
          ) : (
            <>
              <span className="font-medium text-success">{c.agents_healthy} healthy</span>
              {c.agents_late > 0 && <span className="font-medium text-warning"> · {c.agents_late} late</span>}
              {c.agents_missing > 0 && <span className="font-medium text-danger"> · {c.agents_missing} missing</span>}
              {c.agents_booting > 0 && <span> · {c.agents_booting} booting</span>}
            </>
          )}
        </span>
        <span className="text-fg-muted">
          Steps outstanding <span className="font-medium text-fg">{c.tasks_outstanding}</span>
        </span>
        <span className={cn(c.objects_failed + c.tasks_failed > 0 ? 'font-medium text-danger' : 'text-fg-muted')}>
          Failures {c.objects_failed + c.tasks_failed}
        </span>
      </div>

      {build.access && build.access.length > 0 && <AccessCountdown windows={build.access} />}
    </Link>
  )
}

function Metric({ icon, label, value }: { icon: ReactNode; label: string; value: ReactNode }) {
  return (
    <div className="rounded-token bg-surface-sunken px-2 py-1.5">
      <div className="flex items-center justify-center gap-1 text-[11px] text-fg-muted">
        {icon}
        {label}
      </div>
      <div className="text-sm font-semibold text-fg">{value}</div>
    </div>
  )
}
