import { useMemo, useState } from 'react'
import { Link } from '@tanstack/react-router'
import { ChevronDown, GitCommitHorizontal, Hammer, IterationCw, MousePointerClick, Plus, Rocket, Lock, Unlock, Trash2, RefreshCw } from 'lucide-react'
import {
  useBuilderConfigs,
  useBuilds,
  useConfiguredBuilds,
  useCreateConfiguredBuild,
  useTriggerBuild,
  useSyncConfiguredBuild,
  useDeleteConfiguredBuild,
  useSetConfiguredBuildAutoDeploy,
  useSetLock,
} from '../api/hooks'
import { ApiError } from '../api/client'
import { useTimeFormat } from '../lib/time'
import { NewBuildModal } from './NewBuildModal'
import { StatusBadge } from './StatusBadge'
import type { Build, ConfiguredBuild } from '../api/types'
import { Button, Card, CardHeader, cn, Spinner, Table, TableScroller, Td, Th } from '../ui'
import { BuilderKindBadge } from './BuilderKindBadge'

// Which branch cards are twisted open persists across navigation and
// reloads (direct product feedback), keyed by the configured-build id
// (globally unique). Per-viewer convenience only, so localStorage is the
// right home; every access is guarded since it can be blocked or throw.
const EXPANDED_KEY = 'laforge:configured-builds:expanded'

function readExpandedIds(): Set<string> {
  try {
    return new Set(JSON.parse(localStorage.getItem(EXPANDED_KEY) ?? '[]') as string[])
  } catch {
    return new Set()
  }
}

function persistExpanded(id: string, expanded: boolean) {
  try {
    const s = readExpandedIds()
    if (expanded) s.add(id)
    else s.delete(id)
    localStorage.setItem(EXPANDED_KEY, JSON.stringify([...s]))
  } catch {
    // Storage blocked (private mode): the twist state just won't persist.
  }
}

// "A build is always configured explicitly: repository, branch,
// environment file, builder config" -- the API for
// this has existed for a while (internal/api/builds.go). Auto-deploy
// (per branch) and lock toggles get their first UI surface
// here too -- "auto-build is on by default, always... both can be turned
// off per repository" and "marking the competition started locks
// automatic deploys."
//
// Builds (the actual environments) are grouped *under* the configured
// build that produced them (direct product feedback: a flat build list
// "doesn't map to configured builds") -- each branch is a collapsible
// section showing its last build's state and a small history strip inline,
// expanding to that branch's own builds.
export function ConfiguredBuilds({ repoId }: { repoId: string }) {
  const { data: builds, isLoading } = useConfiguredBuilds(repoId)
  const { data: allBuilds } = useBuilds(repoId)
  const { data: builderConfigs } = useBuilderConfigs()
  const kindByName = useMemo(() => new Map((builderConfigs ?? []).map((bc) => [bc.name, bc.kind])), [builderConfigs])
  const create = useCreateConfiguredBuild(repoId)
  const [showForm, setShowForm] = useState(false)

  // Group every build under its configured build. Builds predating a
  // configured build, or triggered outside one, have a null
  // configured_build_id and fall into the "unassociated" bucket below so
  // they never silently disappear.
  const { byConfigured, unassociated } = useMemo(() => {
    const byConfigured = new Map<string, Build[]>()
    const unassociated: Build[] = []
    for (const b of allBuilds ?? []) {
      if (b.configured_build_id) {
        const list = byConfigured.get(b.configured_build_id) ?? []
        list.push(b)
        byConfigured.set(b.configured_build_id, list)
      } else {
        unassociated.push(b)
      }
    }
    // Newest first within each group.
    const cmp = (a: Build, z: Build) => new Date(z.created_at).getTime() - new Date(a.created_at).getTime()
    for (const list of byConfigured.values()) list.sort(cmp)
    unassociated.sort(cmp)
    return { byConfigured, unassociated }
  }, [allBuilds])

  return (
    <Card className="mb-6">
      <CardHeader
        title={
          <span className="flex items-center gap-1.5">
            <Hammer size={14} /> Configured Builds
          </span>
        }
        actions={
          <Button variant="primary" size="sm" onClick={() => setShowForm(true)}>
            <Plus size={12} /> New Build
          </Button>
        }
      />

      <NewBuildModal
        repoId={repoId}
        open={showForm}
        onClose={() => setShowForm(false)}
        onCreate={async (req) => {
          await create.mutateAsync(req)
          setShowForm(false)
        }}
      />

      {isLoading && (
        <div className="flex items-center gap-2 p-4 text-sm text-fg-muted">
          <Spinner /> Loading…
        </div>
      )}

      {!isLoading && (!builds || builds.length === 0) && !showForm && (
        <div className="p-4 text-sm text-fg-muted">
          Nothing configured yet. A configured build ties a branch, an environment file, and a builder
          together -- every CI-passing push to that branch then tracks a validated commit here, ready to build.
        </div>
      )}

      {builds && builds.length > 0 && (
        <div className="flex flex-col gap-2 p-3 pt-0">
          {builds.map((cb) => (
            <ConfiguredBuildRow key={cb.id} cb={cb} repoId={repoId} builds={byConfigured.get(cb.id) ?? []} builderKind={kindByName.get(cb.builder_config_name)} />
          ))}
          {unassociated.length > 0 && <UnassociatedBuilds repoId={repoId} builds={unassociated} />}
        </div>
      )}
    </Card>
  )
}

function ConfiguredBuildRow({ cb, repoId, builds, builderKind }: { cb: ConfiguredBuild; repoId: string; builds: Build[]; builderKind?: string }) {
  const fmt = useTimeFormat()
  const trigger = useTriggerBuild(repoId)
  const sync = useSyncConfiguredBuild(repoId)
  const remove = useDeleteConfiguredBuild(repoId)
  const setAutoDeploy = useSetConfiguredBuildAutoDeploy(repoId)
  const setLock = useSetLock(repoId)

  // A configured build can't be removed while any of its builds owns or is
  // acting on real infrastructure -- the same "live" set the server enforces.
  const hasLiveBuild = builds.some((b) => ['deploying', 'building', 'finished', 'tearing_down'].includes(b.status))
  const [error, setError] = useState<string | null>(null)
  const [justBuilt, setJustBuilt] = useState<string | null>(null)
  const [expanded, setExpanded] = useState(() => readExpandedIds().has(cb.id))

  function toggleExpanded() {
    setExpanded((v) => {
      persistExpanded(cb.id, !v)
      return !v
    })
  }

  const lastBuild = builds[0] ?? null

  async function onSync() {
    setError(null)
    try {
      await sync.mutateAsync(cb.id)
    } catch (e) {
      setError(e instanceof ApiError ? e.message : 'Sync failed')
    }
  }

  async function onRemove() {
    if (!window.confirm(`Remove the configured build for ${cb.environment_path} on ${cb.branch}? Existing builds are kept as history.`)) return
    setError(null)
    try {
      await remove.mutateAsync(cb.id)
    } catch (e) {
      setError(e instanceof ApiError ? e.message : 'Remove failed')
    }
  }

  async function onBuild() {
    setError(null)
    try {
      const build = await trigger.mutateAsync(cb.id)
      setJustBuilt(build.id)
    } catch (e) {
      setError(e instanceof ApiError ? e.message : 'Build failed')
    }
  }

  async function onToggleAutoDeploy() {
    setError(null)
    try {
      await setAutoDeploy.mutateAsync({ configuredBuildId: cb.id, enabled: !cb.auto_deploy_enabled })
    } catch (e) {
      setError(e instanceof ApiError ? e.message : 'Failed to change auto-deploy')
    }
  }

  async function onToggleLock() {
    setError(null)
    try {
      await setLock.mutateAsync({ configuredBuildId: cb.id, started: !cb.competition_started })
    } catch (e) {
      setError(e instanceof ApiError ? e.message : 'Failed to change lock')
    }
  }

  return (
    <div className="overflow-hidden rounded-token border border-border">
      <div className="flex items-center gap-3 px-3 py-2.5">
        <button
          onClick={toggleExpanded}
          className="flex min-w-0 flex-1 items-center gap-2 text-left"
          aria-expanded={expanded}
          title={builds.length > 0 ? `${builds.length} build${builds.length === 1 ? '' : 's'}` : 'No builds yet'}
        >
          <ChevronDown size={14} className={cn('shrink-0 text-fg-subtle transition-transform', !expanded && '-rotate-90')} />
          <div className="min-w-0">
            <div className="flex items-center gap-2">
              <span className="font-mono text-xs font-medium text-fg">{cb.branch}</span>
              {lastBuild ? <StatusBadge status={lastBuild.status} /> : <span className="text-xs italic text-fg-subtle">no builds yet</span>}
            </div>
            <div className="mt-0.5 flex items-center gap-1.5 text-xs text-fg-muted">
              <span className="truncate">
                {cb.environment_path} · {cb.builder_config_name}
              </span>
              <BuilderKindBadge kind={builderKind} name={cb.builder_config_name} />
            </div>
          </div>
        </button>

        <BuildHistoryStrip builds={builds} repoId={repoId} />

        <button
          onClick={onToggleAutoDeploy}
          disabled={setAutoDeploy.isPending}
          title={
            cb.auto_deploy_enabled
              ? 'Auto-deploy on: every CI-passing push to this branch builds and deploys. Click for build-only.'
              : 'Build-only: pushes still build (validated, free) but wait for a manual deploy. Click to auto-deploy.'
          }
          className={cn(
            'flex shrink-0 items-center gap-1 rounded-token-sm border px-2 py-1 text-xs font-medium disabled:cursor-not-allowed disabled:opacity-40',
            cb.auto_deploy_enabled ? 'border-accent text-accent' : 'border-border text-fg-muted hover:bg-surface-hover',
          )}
        >
          <Rocket size={12} /> {cb.auto_deploy_enabled ? 'Auto-deploy' : 'Build only'}
        </button>

        <button
          onClick={onToggleLock}
          disabled={setLock.isPending}
          title={
            cb.competition_started
              ? 'Protected: automatic deploys are locked. Manual deploys, rebuilds, and ad-hoc tasks still work. Click to unprotect.'
              : 'Not protected: automatic deploys are allowed. Click to protect before a live event.'
          }
          className={cn(
            'flex shrink-0 items-center gap-1 rounded-token-sm border px-2 py-1 text-xs font-medium disabled:cursor-not-allowed disabled:opacity-40',
            cb.competition_started ? 'border-warning text-warning' : 'border-border text-fg-muted hover:bg-surface-hover',
          )}
        >
          {cb.competition_started ? <Lock size={12} /> : <Unlock size={12} />}
          {cb.competition_started ? 'Protected' : 'Not Protected'}
        </button>

        <div className="shrink-0 text-right">
          {justBuilt ? (
            <Link to="/repos/$repoId/builds/$buildId" params={{ repoId, buildId: justBuilt }} className="text-xs font-medium text-accent hover:underline">
              View build →
            </Link>
          ) : cb.current_content_revision_id ? (
            <Button variant="coral" size="sm" onClick={onBuild} disabled={trigger.isPending}>
              {trigger.isPending ? 'Building…' : 'Build Now'}
            </Button>
          ) : null}
        </div>

        {/* Force Pull: re-ingest the branch HEAD on demand (content is auto-pulled
            when a build is added; this re-pulls after a later commit, or when a push
            webhook hasn't landed). Always available; surfaces validation errors. */}
        <button
          onClick={onSync}
          disabled={sync.isPending}
          title="Force Pull: re-ingest the latest commit on this branch (no push needed)"
          className="flex shrink-0 items-center rounded-token-sm border border-border p-1 text-fg-muted hover:border-accent hover:text-accent disabled:cursor-not-allowed disabled:opacity-40 disabled:hover:border-border disabled:hover:text-fg-muted"
          aria-label="Force pull latest commit"
        >
          <RefreshCw size={14} className={sync.isPending ? 'animate-spin' : undefined} />
        </button>

        <button
          onClick={onRemove}
          disabled={hasLiveBuild || remove.isPending}
          title={
            hasLiveBuild
              ? 'Cannot remove: this build has live builds (deploying, building, finished, or tearing down). Tear them down first.'
              : 'Remove this configured build. Existing builds are kept as history.'
          }
          className="flex shrink-0 items-center rounded-token-sm border border-border p-1 text-fg-muted hover:border-danger hover:text-danger disabled:cursor-not-allowed disabled:opacity-40 disabled:hover:border-border disabled:hover:text-fg-muted"
          aria-label="Remove configured build"
        >
          <Trash2 size={14} />
        </button>
      </div>

      {error && <div className="border-t border-border px-3 py-2 text-xs text-danger">{error}</div>}

      {expanded && (
        <div className="border-t border-border bg-surface-sunken/40">
          {builds.length === 0 ? (
            <div className="px-3 py-3 text-xs text-fg-muted">No builds yet for this branch.</div>
          ) : (
            <TableScroller contain={false}>
              <Table>
                <thead>
                  <tr>
                    <Th className="pl-4">Environment</Th>
                    <Th>Commit</Th>
                    <Th>State</Th>
                    <Th className="pr-4">Created</Th>
                  </tr>
                </thead>
                <tbody>
                  {builds.map((b) => (
                    <tr key={b.id}>
                      <Td className="pl-4">
                        <span className="flex items-center gap-1.5">
                          <Link to="/repos/$repoId/builds/$buildId" params={{ repoId, buildId: b.id }} className="font-medium text-accent hover:underline">
                            {b.environment_name}
                          </Link>
                          {b.auto_built ? (
                            <span title="Auto-built from a CI-passing push" className="inline-flex text-fg-subtle">
                              <IterationCw size={13} />
                            </span>
                          ) : (
                            <span title="Built manually (Build Now)" className="inline-flex text-fg-subtle">
                              <MousePointerClick size={13} />
                            </span>
                          )}
                        </span>
                      </Td>
                      <Td>
                        {b.commit_sha ? (
                          <span className="flex items-center gap-1.5 text-xs" title={b.commit_message ?? undefined}>
                            <GitCommitHorizontal size={13} className="shrink-0 text-fg-subtle" />
                            {b.commit_message ? (
                              <span className="max-w-[36ch] truncate text-fg">
                                {b.commit_message} <span className="font-mono text-fg-muted">({b.commit_sha.slice(0, 7)})</span>
                              </span>
                            ) : (
                              <span className="font-mono text-fg-muted">{b.commit_sha.slice(0, 7)}</span>
                            )}
                          </span>
                        ) : (
                          <span className="text-fg-subtle">—</span>
                        )}
                      </Td>
                      <Td>
                        <StatusBadge status={b.status} />
                      </Td>
                      <Td className="pr-4 text-fg-muted">{fmt.dateTime(b.created_at)}</Td>
                    </tr>
                  ))}
                </tbody>
              </Table>
            </TableScroller>
          )}
        </div>
      )}
    </div>
  )
}

// A compact per-branch history strip: one cell per build, newest on the
// right, coloured by outcome -- the "inline chart per branch" that turns a
// branch's build record into an at-a-glance trend (a run of red means this
// branch keeps failing) without opening anything.
function BuildHistoryStrip({ builds, repoId }: { builds: Build[]; repoId: string }) {
  const fmt = useTimeFormat()
  if (builds.length === 0) return null
  // Oldest -> newest, capped, so the strip reads left-to-right like a timeline.
  const recent = builds.slice(0, 16).reverse()
  return (
    <div className="hidden shrink-0 items-end gap-0.5 sm:flex" aria-hidden={false}>
      {recent.map((b) => (
        <Link
          key={b.id}
          to="/repos/$repoId/builds/$buildId"
          params={{ repoId, buildId: b.id }}
          title={`${b.environment_name} — ${b.status} — ${fmt.dateTime(b.created_at)}`}
          className={cn('h-5 w-1.5 rounded-sm transition-opacity hover:opacity-70', buildStatusColor(b.status))}
        />
      ))}
    </div>
  )
}

function buildStatusColor(status: Build['status']): string {
  switch (status) {
    case 'finished':
      return 'bg-success' // the only green: everything done
    case 'failed':
      return 'bg-danger'
    case 'deploying':
    case 'building':
      return 'bg-accent' // in progress -- infra or agents, never green yet
    case 'planned':
      return 'bg-fg-subtle'
    case 'tearing_down':
    case 'torn_down':
    case 'purged':
    default:
      return 'bg-border-strong'
  }
}

// Builds with no configured build behind them (older builds, or ones
// triggered before the branch was configured) -- kept visible in their own
// collapsible bucket rather than dropped.
function UnassociatedBuilds({ builds, repoId }: { builds: Build[]; repoId: string }) {
  const fmt = useTimeFormat()
  const key = `unassociated:${repoId}`
  const [expanded, setExpanded] = useState(() => readExpandedIds().has(key))
  return (
    <div className="overflow-hidden rounded-token border border-dashed border-border">
      <button
        onClick={() => setExpanded((v) => { persistExpanded(key, !v); return !v })}
        className="flex w-full items-center gap-2 px-3 py-2.5 text-left"
        aria-expanded={expanded}
      >
        <ChevronDown size={14} className={cn('shrink-0 text-fg-subtle transition-transform', !expanded && '-rotate-90')} />
        <span className="text-xs font-medium text-fg-muted">Unassociated builds</span>
        <span className="text-xs text-fg-subtle">({builds.length})</span>
      </button>
      {expanded && (
        <div className="border-t border-border bg-surface-sunken/40">
          <TableScroller contain={false}>
            <Table>
              <thead>
                <tr>
                  <Th className="pl-4">Environment</Th>
                  <Th>State</Th>
                  <Th className="pr-4">Created</Th>
                </tr>
              </thead>
              <tbody>
                {builds.map((b) => (
                  <tr key={b.id}>
                    <Td className="pl-4">
                      <Link to="/repos/$repoId/builds/$buildId" params={{ repoId, buildId: b.id }} className="font-medium text-accent hover:underline">
                        {b.environment_name}
                      </Link>
                    </Td>
                    <Td>
                      <StatusBadge status={b.status} />
                    </Td>
                    <Td className="pr-4 text-fg-muted">{fmt.dateTime(b.created_at)}</Td>
                  </tr>
                ))}
              </tbody>
            </Table>
          </TableScroller>
        </div>
      )}
    </div>
  )
}
