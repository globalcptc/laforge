import { useMemo } from 'react'
import { useParams } from '@tanstack/react-router'
import { AlertTriangle } from 'lucide-react'
import { useBuild, useDashboard, useUpcomingChanges } from '../api/hooks'
import { effectiveStatusKey } from '../components/StatusBadge'
import {
  ProvisioningProgress,
  StateByTeam,
  FailuresByCause,
  AgentActivity,
  ResourceCpu,
  ResourceMem,
  ResourceDisk,
  ResourceNet,
} from '../components/DashboardCharts'
import { StatusBadge } from '../components/StatusBadge'
import { UpcomingChangesPanel } from '../components/UpcomingChangesPanel'
import { Card, Spinner } from '../ui'

// Build → Overview. The counts (finished / in-progress / failures / teams-open)
// that used to sit across the top told you little the charts below don't; what
// actually matters when a build is unhealthy is WHAT failed and why, so instead
// of the tiles we surface the failing objects directly. The four charts
// (provisioning progress, state by team, agent check-ins, failures by cause)
// read from the real GET /builds/{id}/dashboard aggregate (internal/api/dashboard.go).
export function BuildOverview() {
  const { buildId } = useParams({ from: '/repos/$repoId/builds/$buildId/' })
  const { data: build } = useBuild(buildId)
  const { data: dash } = useDashboard(buildId)
  const { data: upcoming } = useUpcomingChanges(buildId)

  // Per-team and build-wide status counts, folding each host/container's agent
  // and infra health into its state (effectiveStatusKey) exactly as the Hosts
  // page's team bars do -- so "up but agent/infra down" shows as a problem here,
  // not as provisioned. Networks (no agent/power) are excluded, matching the
  // Hosts bars' scope. Computed from the build's objects, not the server's
  // lifecycle-only aggregate.
  const byTeam = useMemo(
    () =>
      (build?.teams ?? []).map((t) => {
        const by_status: Record<string, number> = {}
        for (const o of t.objects ?? []) {
          if (o.kind === 'network') continue
          const k = effectiveStatusKey(o)
          by_status[k] = (by_status[k] ?? 0) + 1
        }
        return { team_number: t.team_number, by_status }
      }),
    [build],
  )
  const provisioning = useMemo(() => {
    const c: Record<string, number> = {}
    for (const t of byTeam) for (const [k, n] of Object.entries(t.by_status)) c[k] = (c[k] ?? 0) + n
    return c
  }, [byTeam])

  if (!build || !dash)
    return (
      <div className="flex items-center gap-2 text-sm text-fg-muted">
        <Spinner /> Loading…
      </div>
    )

  // The actual failing objects (terminal failures at any layer), surfaced so an
  // operator sees exactly which host/container failed and why.
  const failures = (build.teams ?? []).flatMap((t) =>
    (t.objects ?? [])
      .filter((o) => o.status === 'deploy_failed' || o.status === 'build_failed' || o.status === 'invalid')
      .map((o) => ({ team: t.team_number, obj: o })),
  )

  return (
    <div className="flex flex-col gap-4">
      {upcoming && <UpcomingChangesPanel buildId={buildId} data={upcoming} />}

      {failures.length > 0 && (
        <Card className="border-danger/40 p-4">
          <div className="mb-2 flex items-center gap-2 text-sm font-semibold text-danger">
            <AlertTriangle size={14} /> {failures.length} failure{failures.length === 1 ? '' : 's'}
          </div>
          <ul className="flex flex-col gap-1.5 text-sm">
            {failures.map(({ team, obj }) => (
              <li key={obj.id} className="flex flex-wrap items-center gap-x-2 gap-y-1">
                <span className="shrink-0 font-mono text-xs text-fg-muted">Team {team}</span>
                <span className="font-medium text-fg">{obj.as_name ?? obj.object_name}</span>
                <StatusBadge status={obj.status} kind={obj.kind} tooltip={false} />
                {obj.last_error ? (
                  <span className="min-w-0 text-fg-muted">— {obj.last_error}</span>
                ) : obj.status === 'invalid' ? (
                  <span className="text-fg-muted">— a validator did not pass</span>
                ) : null}
              </li>
            ))}
          </ul>
        </Card>
      )}

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <ResourceCpu data={dash} />
        <ResourceMem data={dash} />
        <ResourceDisk data={dash} />
        <ResourceNet data={dash} />
      </div>

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <ProvisioningProgress counts={provisioning} />
        <AgentActivity data={dash} />
      </div>
      <FailuresByCause data={dash} />
      <StateByTeam teams={byTeam} />
    </div>
  )
}
