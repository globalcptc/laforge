import { useState } from 'react'
import { Link, Outlet, useNavigate, useParams, useRouterState } from '@tanstack/react-router'
import { GitCommitHorizontal, PowerOff, RotateCcw, Rocket, Trash2, TriangleAlert } from 'lucide-react'
import {
  useBuild,
  useBuildReadiness,
  useBuilders,
  useDeployBuild,
  useForceDeleteBuild,
  useForceTornDown,
  useMe,
  useRetryTeardown,
  useTeardownBuild,
} from '../api/hooks'
import { StatusBadge } from '../components/StatusBadge'
import { BuilderPill } from '../components/BuilderPill'
import { ApiError } from '../api/client'
import { Button, Input, cn, PageHeader, useToast } from '../ui'

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
  const forceDelete = useForceDeleteBuild(buildId, repoId)
  const retryTeardown = useRetryTeardown(buildId)
  const forceTornDown = useForceTornDown(buildId)
  const me = useMe()
  const navigate = useNavigate()
  // Pre-deploy quota warnings matter only while the build can still be deployed.
  const readiness = useBuildReadiness(buildId, build?.status === 'planned')
  const [deployError, setDeployError] = useState<string | null>(null)
  const [teardownError, setTeardownError] = useState<string | null>(null)
  const [forceConfirm, setForceConfirm] = useState('')
  const [forceError, setForceError] = useState<string | null>(null)
  const [forceDownError, setForceDownError] = useState<string | null>(null)
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

  async function onForceTornDown() {
    setForceDownError(null)
    try {
      await forceTornDown.mutateAsync()
      toast({ title: 'Build marked torn down', description: 'No hosts were destroyed; clean up any that remain by hand.', tone: 'info' })
    } catch (e) {
      const message = e instanceof ApiError ? e.message : 'Force torn down failed'
      setForceDownError(message)
      toast({ title: 'Force torn down failed', description: message, tone: 'danger', duration: 0 })
    }
  }

  async function onForceDelete() {
    setForceError(null)
    try {
      await forceDelete.mutateAsync()
      // The build no longer exists -- leave its page for the repo's build list.
      navigate({ to: '/repos/$repoId', params: { repoId } })
    } catch (e) {
      const message = e instanceof ApiError ? e.message : 'Force remove failed'
      setForceError(message)
      toast({ title: 'Force remove failed', description: message, tone: 'danger', duration: 0 })
    }
  }

  return (
    <>
      <PageHeader
        title={
          <span className="flex min-w-0 items-center gap-3">
            <span className="shrink-0">{build?.environment_name ?? '…'}</span>
            {build?.builder_config_name && <BuilderPill name={build.builder_config_name} color={builderColor} kind={build.builder_kind} />}
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

        {/* Buried admin escape hatch: force-remove a DEAD build from the
            database (its real hosts were deleted out-of-band, so teardown can
            never finish and it's wedged in 'tearing_down'). Instance-admins
            only, collapsed by default, and gated behind typing the exact build
            id -- it is deliberately hard to reach and hard to fire, because it
            does NOT destroy any hoster resources. */}
        {me.data?.is_instance_admin && build && (
          <details className="mt-10 border-t border-border pt-3 text-xs text-fg-subtle">
            <summary className="cursor-pointer select-none text-fg-subtle hover:text-fg-muted">Admin</summary>
            {build.status === 'tearing_down' && (
              <div className="mt-3 max-w-xl rounded-token border border-warning/30 bg-warning-soft/40 p-3.5">
                <div className="flex items-center gap-2 text-sm font-semibold text-warning">
                  <TriangleAlert size={14} /> Force Torn Down
                </div>
                <p className="mt-1.5 text-fg-muted">
                  Marks this build torn down and every remaining host, container, and network as destroyed{' '}
                  <span className="font-semibold text-fg">without</span> asking the hoster to remove anything. Use it for a teardown
                  that cannot finish when the real infrastructure is already gone or you will remove it by hand. The build and its
                  history stay.
                </p>
                <div className="mt-2 flex flex-wrap items-center gap-2">
                  <Button variant="danger" onClick={onForceTornDown} disabled={forceTornDown.isPending}>
                    <PowerOff size={12} /> {forceTornDown.isPending ? 'Marking…' : 'Force Torn Down'}
                  </Button>
                </div>
                {forceDownError && <div className="mt-2 text-danger">{forceDownError}</div>}
              </div>
            )}
            <div className="mt-3 max-w-xl rounded-token border border-danger/30 bg-danger-soft/40 p-3.5">
              <div className="flex items-center gap-2 text-sm font-semibold text-danger">
                <TriangleAlert size={14} /> Force remove from database
              </div>
              <p className="mt-1.5 text-fg-muted">
                Deletes this build and all its records from the database. It does <span className="font-semibold text-fg">not</span> tear
                down any hosts, containers, or networks — use <span className="font-medium">Tear Down</span> for that. Only use this for a
                build whose real infrastructure is already gone (so teardown can never complete). This cannot be undone.
              </p>
              <p className="mt-2 text-fg-muted">
                Type the build id <span className="font-mono text-fg-subtle">{buildId}</span> to confirm:
              </p>
              <div className="mt-1.5 flex flex-wrap items-center gap-2">
                <Input
                  value={forceConfirm}
                  onChange={(e) => setForceConfirm(e.target.value)}
                  placeholder="build id"
                  className="w-80 max-w-full font-mono"
                  aria-label="Type the build id to confirm removal"
                />
                <Button
                  variant="danger"
                  disabled={forceConfirm.trim() !== buildId || forceDelete.isPending}
                  onClick={onForceDelete}
                >
                  <Trash2 size={12} /> {forceDelete.isPending ? 'Removing…' : 'Force Remove'}
                </Button>
              </div>
              {forceError && <div className="mt-2 text-danger">{forceError}</div>}
            </div>
          </details>
        )}
      </div>
    </>
  )
}
