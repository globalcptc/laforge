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

// How many capacity signals to show per builder before collapsing the rest into
// a "+N more" link -- enough to see the real constraints without the home page
// turning into a full metrics dashboard (that's what the builder page is for).
const CAPACITY_SHOWN = 5

const fmtNum = (n: number) => (Number.isInteger(n) ? n.toLocaleString() : n.toFixed(1))

// CapacityBar is one build-gating signal, named concretely: a storage pool or a
// node's memory as percent-full, or an AWS quota as used/limit. The number on
// the right is the whole point -- "Disk (local) · node3  92%" or "Elastic IPs
// 5 / 5  100%" tells you exactly what's full and by how much.
function CapacityBar({ sig }: { sig: HomeCapacity }) {
  const pct = Math.min(100, Math.max(0, sig.pct))
  const tone = sig.pct >= 90 ? 'bg-danger' : sig.pct >= 75 ? 'bg-warning' : 'bg-accent'
  const label = sig.node ? `${sig.label} · ${sig.node}` : sig.label
  const quota = sig.max != null
  return (
    <div className="grid grid-cols-[minmax(0,1fr)_5rem_auto] items-center gap-3 text-xs">
      <span className="truncate text-fg-muted" title={label}>
        {label}
      </span>
      <div className="h-1.5 overflow-hidden rounded-full bg-surface-sunken" title={`${sig.pct.toFixed(0)}%`}>
        <div className={cn('h-full rounded-full', tone)} style={{ width: `${pct}%` }} />
      </div>
      <span className="text-right font-mono tabular-nums text-fg">
        {quota ? `${fmtNum(sig.value)} / ${fmtNum(sig.max!)}` : `${sig.pct.toFixed(0)}%`}
        {quota && <span className="ml-1.5 text-fg-subtle">{sig.pct.toFixed(0)}%</span>}
      </span>
    </div>
  )
}

function BuilderUsage({ builders, buildersInUse }: { builders: HomeData['builders']; buildersInUse: number }) {
  const { data: me } = useMe()
  return (
    <Card>
      <div className="flex items-center justify-between border-b border-border px-4 py-2.5">
        <div className="flex items-center gap-2 text-sm font-semibold text-fg">
          <ServerCog size={14} className="text-fg-muted" /> Infrastructure usage
        </div>
        <span className="text-xs text-fg-muted">
          {buildersInUse} of {builders.length} in use
        </span>
      </div>
      <div className="divide-y divide-border">
        {builders.map((b) => {
          const capacity = b.capacity ?? []
          const shown = capacity.slice(0, CAPACITY_SHOWN)
          const hidden = capacity.length - shown.length
          const name = me?.is_instance_admin ? (
            <Link to="/admin/infrastructure/$name" params={{ name: b.name }} className="hover:opacity-80">
              <BuilderPill name={b.name} color={b.color} kind={b.kind} />
            </Link>
          ) : (
            <BuilderPill name={b.name} color={b.color} kind={b.kind} />
          )
          return (
            <div key={b.name} className="px-4 py-3">
              <div className="flex flex-wrap items-center justify-between gap-x-4 gap-y-1">
                <div className="flex min-w-0 items-center gap-2">
                  {name}
                  <span className="text-xs text-fg-muted">{KIND_LABEL[b.kind as Kind] ?? b.kind}</span>
                </div>
                <div className="flex gap-3 text-xs text-fg-muted">
                  <span>
                    <span className="font-medium text-fg">{b.active_builds}</span> builds
                  </span>
                  <span>
                    <span className="font-medium text-fg">{b.counts.hosts}</span> hosts
                  </span>
                  <span>
                    <span className="font-medium text-fg">{b.counts.containers}</span> containers
                  </span>
                  <span>
                    <span className="font-medium text-fg">{b.counts.networks}</span> networks
                  </span>
                </div>
              </div>
              {capacity.length > 0 ? (
                <div className="mt-2.5 space-y-1.5">
                  {shown.map((sig) => (
                    <CapacityBar key={`${sig.key}|${sig.node}`} sig={sig} />
                  ))}
                  {hidden > 0 &&
                    (me?.is_instance_admin ? (
                      <Link
                        to="/admin/infrastructure/$name"
                        params={{ name: b.name }}
                        className="inline-block text-xs text-accent hover:underline"
                      >
                        +{hidden} more →
                      </Link>
                    ) : (
                      <span className="text-xs text-fg-subtle">+{hidden} more</span>
                    ))}
                </div>
              ) : (
                <div className="mt-2 text-xs text-fg-subtle">No capacity metrics reported yet.</div>
              )}
            </div>
          )
        })}
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

      <div className="grid grid-cols-4 gap-2 text-center">
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
