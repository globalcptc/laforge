import { useState } from 'react'
import { Link, Outlet, useParams, useRouterState } from '@tanstack/react-router'
import { GitCommitHorizontal, Rocket, Trash2 } from 'lucide-react'
import { useBuild, useDeployBuild, useTeardownBuild } from '../api/hooks'
import { StatusBadge } from '../components/StatusBadge'
import { BuilderKindBadge } from '../components/BuilderKindBadge'
import { ApiError } from '../api/client'
import { Button, cn, PageHeader, useToast } from '../ui'

const TABS = [
  { to: '', label: 'Overview' },
  { to: 'hosts', label: 'Hosts' },
  { to: 'access', label: 'Access' },
  { to: 'schedule', label: 'Schedule' },
  { to: 'logs', label: 'Logs' },
  { to: 'findings', label: 'Findings' },
  { to: 'artifacts', label: 'Artifacts' },
] as const

// The build's own sub-navigation -- the full "Build → Overview / Hosts /
// Topology / Access / Schedule / Logs / Findings / Artifacts" information
// architecture.
export function BuildLayout() {
  const { repoId, buildId } = useParams({ from: '/repos/$repoId/builds/$buildId' })
  const { data: build } = useBuild(buildId)
  const deploy = useDeployBuild(buildId)
  const teardown = useTeardownBuild(buildId)
  const [deployError, setDeployError] = useState<string | null>(null)
  const [teardownError, setTeardownError] = useState<string | null>(null)
  const toast = useToast()
  const pathname = useRouterState({ select: (s) => s.location.pathname })
  const base = `/repos/${repoId}/builds/${buildId}`

  async function onDeploy() {
    setDeployError(null)
    try {
      // No success toast for a routine start -- the status badge flips to
      // "deploying" immediately, and anything that goes wrong surfaces in
      // the app-wide alerts. Toasts are reserved for failures now.
      await deploy.mutateAsync()
    } catch (e) {
      const message = e instanceof ApiError ? e.message : 'Deploy failed'
      setDeployError(message)
      toast({ title: 'Deploy failed', description: message, tone: 'danger', duration: 0 })
    }
  }

  async function onTeardown() {
    if (!confirm('Tear down this build? Every real host, container, and network it deployed will be destroyed.')) return
    setTeardownError(null)
    try {
      await teardown.mutateAsync()
    } catch (e) {
      const message = e instanceof ApiError ? e.message : 'Teardown failed'
      setTeardownError(message)
      toast({ title: 'Teardown failed', description: message, tone: 'danger', duration: 0 })
    }
  }

  return (
    <>
      <PageHeader
        title={
          <span className="flex min-w-0 items-center gap-3">
            <span className="shrink-0">{build?.environment_name ?? '…'}</span>
            {build?.builder_kind && <BuilderKindBadge kind={build.builder_kind} name={build.builder_config_name} label={build.builder_config_name} />}
            {build && <StatusBadge status={build.status} />}
            {build?.commit_sha && (
              <span className="flex min-w-0 items-center gap-1.5 text-sm font-normal text-fg-muted" title={build.commit_message ?? undefined}>
                <GitCommitHorizontal size={13} className="shrink-0 text-fg-subtle" />
                {build.commit_message ? (
                  <span className="truncate">
                    {build.commit_message} <span className="font-mono">({build.commit_sha.slice(0, 7)})</span>
                  </span>
                ) : (
                  <span className="shrink-0 font-mono">{build.commit_sha.slice(0, 7)}</span>
                )}
              </span>
            )}
          </span>
        }
        breadcrumbs={[{ label: 'Repositories', href: '/repos' }, { label: 'Repository', href: `/repos/${repoId}` }, { label: build?.environment_name ?? '…' }]}
        actions={
          <>
            {build?.status === 'planned' && (
              <Button variant="primary" onClick={onDeploy} disabled={deploy.isPending}>
                <Rocket size={12} /> {deploy.isPending ? 'Deploying…' : 'Deploy'}
              </Button>
            )}
            {build && ['deploying', 'building', 'finished', 'failed'].includes(build.status) && (
              <Button variant="danger" onClick={onTeardown} disabled={teardown.isPending}>
                <Trash2 size={12} /> {teardown.isPending ? 'Tearing Down…' : 'Tear Down'}
              </Button>
            )}
            {deployError && <div className="text-xs text-danger">{deployError}</div>}
            {teardownError && <div className="text-xs text-danger">{teardownError}</div>}
          </>
        }
      />
      {/* Header and tabs are fixed; only the tab body below scrolls. The
         page-level scroll (app-content) let a long tab -- the Logs journal
         especially -- carry the tab bar off-screen entirely, so navigating
         between tabs meant scrolling back up first. Giving the body its own
         scroll region keeps the header and tabs put no matter how long the
         content is. Six tabs don't fit a phone width, so the bar itself
         scrolls horizontally rather than wrapping. */}
      <nav className="flex shrink-0 gap-1 overflow-x-auto border-b border-border bg-surface px-6">
        {TABS.map((tab) => {
          const href = tab.to ? `${base}/${tab.to}` : base
          const active = pathname === href || (tab.to === '' && pathname === base + '/')
          return (
            <Link
              key={tab.label}
              to={href}
              className={cn(
                'shrink-0 whitespace-nowrap border-b-2 px-3 py-2 text-sm font-medium',
                active ? 'border-accent text-fg' : 'border-transparent text-fg-muted hover:text-fg',
              )}
            >
              {tab.label}
            </Link>
          )
        })}
      </nav>
      <div className="min-h-0 flex-1 overflow-y-auto px-6 pb-8 pt-2">
        <Outlet />
      </div>
    </>
  )
}
