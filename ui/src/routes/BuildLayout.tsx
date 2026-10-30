import { useState } from 'react'
import { Link, Outlet, useParams, useRouterState } from '@tanstack/react-router'
import { GitCommitHorizontal, RotateCcw, Rocket, Trash2, TriangleAlert } from 'lucide-react'
import {
  useBuild,
  useBuildReadiness,
  useBuilders,
  useDeployBuild,
  useMe,
  useRetryTeardown,
  useTeardownBuild,
} from '../api/hooks'
import { StatusBadge } from '../components/StatusBadge'
import { BuilderPill } from '../components/BuilderPill'
import { ApiError } from '../api/client'
import { Button, cn, PageHeader, useToast } from '../ui'

const TABS = [
  { to: '', label: 'Overview' },
  { to: 'hosts', label: 'Hosts' },
  { to: 'access', label: 'Access' },
  { to: 'external-access', label: 'External Access' },
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
  const { data: builders } = useBuilders()
  const builderColor = builders?.find((b) => b.name === build?.builder_config_name)?.color
  const deploy = useDeployBuild(buildId)
  const teardown = useTeardownBuild(buildId)
  const retryTeardown = useRetryTeardown(buildId)
  const me = useMe()
  // Pre-deploy quota warnings matter only while the build can still be deployed.
  const readiness = useBuildReadiness(buildId, build?.status === 'planned')
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

  async function onRetryTeardown() {
    try {
      await retryTeardown.mutateAsync()
    } catch (e) {
      const message = e instanceof ApiError ? e.message : 'Retry failed'
      toast({ title: 'Teardown retry failed', description: message, tone: 'danger', duration: 0 })
    }
  }

  return (
    <>
      <PageHeader
        title={
          <span className="flex min-w-0 flex-wrap items-center gap-x-3 gap-y-1.5">
            <span className="min-w-0 break-words">{build?.environment_name ?? '…'}</span>
            {build?.builder_config_name && <BuilderPill name={build.builder_config_name} color={builderColor} kind={build.builder_kind} />}
            {build && <StatusBadge status={build.status} />}
            {build?.commit_sha && (
              <span className="flex w-full min-w-0 items-center gap-1.5 text-sm font-normal text-fg-muted md:w-auto md:flex-1" title={build.commit_message ?? undefined}>
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
        breadcrumbs={[
          { label: 'Repositories', href: '/repos' },
          { label: 'Repository', href: `/repos/${repoId}` },
          ...(build?.branch ? [{ label: build.branch }] : []),
          { label: build?.environment_name ?? '…' },
        ]}
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
      <nav className="flex shrink-0 gap-1 overflow-x-auto border-b border-border bg-surface px-6 [scrollbar-width:none] [&::-webkit-scrollbar]:hidden">
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
        {/* Admin escape hatches live in their own tab at the very end, visible
            only to instance admins (the route itself also gates the content). */}
        {me.data?.is_instance_admin && (
          (() => {
            const href = `${base}/admin`
            const active = pathname === href
            return (
              <Link
                to={href}
                className={cn(
                  'ml-auto shrink-0 whitespace-nowrap border-b-2 px-3 py-2 text-sm font-medium',
                  active ? 'border-accent text-fg' : 'border-transparent text-fg-subtle hover:text-fg',
                )}
              >
                Admin
              </Link>
            )
          })()
        )}
      </nav>
      <div className="min-h-0 flex-1 overflow-y-auto overflow-x-hidden px-6 pb-8 pt-2">
        {build?.status === 'planned' && readiness.data?.checked && (readiness.data.warnings ?? []).length > 0 && (
          <div className="mb-3 flex items-start gap-2.5 rounded-token border border-warning/30 bg-warning-soft px-3.5 py-2.5 text-sm text-warning">
            <TriangleAlert size={16} className="mt-0.5 shrink-0" />
            <div className="min-w-0">
              <div className="font-medium">This build may exceed the builder’s account quotas</div>
              <ul className="mt-1 list-disc space-y-0.5 pl-4 text-warning/90">
                {(readiness.data.warnings ?? []).map((warn, i) => (
                  <li key={i} className="break-words">{warn}</li>
                ))}
              </ul>
            </div>
          </div>
        )}
        {/* reconcile_error carries two things: a deploy that can't start
            (builder/content incompatibility) and, while tearing_down, a
            teardown that stopped because the hoster refused a destroy for a
            reason retrying can't fix. The second gets its own title and the
            explicit retry the orchestrator waits for. */}
        {build?.reconcile_error && (
          <div className="mb-3 flex items-start gap-2.5 rounded-token border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-sm text-danger">
            <TriangleAlert size={16} className="mt-0.5 shrink-0" />
            <div className="min-w-0 flex-1">
              <div className="font-medium">
                {build.status === 'tearing_down' ? 'Teardown Is Stuck' : 'This build is blocked and can’t deploy'}
              </div>
              <div className="mt-0.5 break-words text-danger/90">{build.reconcile_error}</div>
              {build.status === 'tearing_down' && (
                <div className="mt-2">
                  <Button variant="primary" onClick={onRetryTeardown} disabled={retryTeardown.isPending}>
                    <RotateCcw size={12} /> {retryTeardown.isPending ? 'Retrying…' : 'Retry Teardown'}
                  </Button>
                </div>
              )}
            </div>
          </div>
        )}
        <Outlet />
      </div>
    </>
  )
}
