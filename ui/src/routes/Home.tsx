import type { ReactNode } from 'react'
import { Link } from '@tanstack/react-router'
import { AlertTriangle, Boxes, ChevronRight, FolderGit2, GitCommit, Network, Server, ServerCog, Users, X } from 'lucide-react'
import { useDismissAttention, useHome, useMe } from '../api/hooks'
import type { AttentionItem, HomeBuild, HomeCounts, HomeData } from '../api/types'
import { AccessCountdown } from '../components/AccessCountdown'
import { EmptyState } from '../components/EmptyState'
import { StatusBadge } from '../components/StatusBadge'
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
  const t = data.totals
  const failures = t.objects_failed + t.tasks_failed
  const agents = t.agents_healthy + t.agents_late + t.agents_missing + t.agents_booting
  const buildersInUse = data.builders.filter((b) => b.active_builds > 0).length

  return (
    <>
      <div className="grid grid-cols-2 gap-3 md:grid-cols-3 xl:grid-cols-6">
        <StatTile label="Active builds" value={data.active_builds} detail={`${data.repositories.length} repositor${data.repositories.length === 1 ? 'y' : 'ies'}`} />
        <StatTile label="Hosts" value={t.hosts} detail={`${t.containers} containers`} />
        <StatTile label="Networks" value={t.networks} />
        <StatTile
          label="Agents healthy"
          value={agents ? `${t.agents_healthy}/${agents}` : '—'}
          detail={agents ? `${t.agents_late} late · ${t.agents_missing} missing` : 'No agents yet'}
          tone={t.agents_missing > 0 ? 'bad' : agents > 0 && t.agents_healthy === agents ? 'good' : undefined}
        />
        <StatTile label="Steps outstanding" value={t.tasks_outstanding} />
        <StatTile label="Failures" value={failures} detail={`${t.objects_failed} objects · ${t.tasks_failed} steps`} tone={failures > 0 ? 'bad' : 'good'} />
      </div>

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

function StatTile({ label, value, detail, tone }: { label: string; value: ReactNode; detail?: string; tone?: 'good' | 'bad' }) {
  return (
    <Card className="px-4 py-3">
      <div className="text-xs font-medium uppercase tracking-wide text-fg-muted">{label}</div>
      <div className={cn('mt-1 text-2xl font-semibold text-fg', tone === 'bad' && 'text-danger', tone === 'good' && 'text-success')}>{value}</div>
      {detail && <div className="mt-0.5 text-xs text-fg-muted">{detail}</div>}
    </Card>
  )
}

function BuilderUsage({ builders, buildersInUse }: { builders: HomeData['builders']; buildersInUse: number }) {
  const { data: me } = useMe()
  const maxHosts = Math.max(1, ...builders.map((b) => b.counts.hosts + b.counts.containers))
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
          const machines = b.counts.hosts + b.counts.containers
          const name = me?.is_instance_admin ? (
            <Link to="/admin/infrastructure/$name" params={{ name: b.name }} className="font-medium text-fg hover:text-accent">
              {b.name}
            </Link>
          ) : (
            <span className="font-medium text-fg">{b.name}</span>
          )
          return (
            <div key={b.name} className="grid grid-cols-[minmax(0,12rem)_1fr_auto] items-center gap-4 px-4 py-2.5 text-sm">
              <div className="min-w-0">
                <div className="truncate">{name}</div>
                <div className="text-xs text-fg-muted">{KIND_LABEL[b.kind as Kind] ?? b.kind}</div>
              </div>
              <div className="h-2 overflow-hidden rounded-full bg-surface-sunken" title={`${machines} machines`}>
                <div className="h-full rounded-full bg-accent" style={{ width: `${(machines / maxHosts) * 100}%` }} />
              </div>
              <div className="flex gap-4 text-xs text-fg-muted">
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
