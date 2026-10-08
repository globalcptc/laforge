import { useParams } from '@tanstack/react-router'
import { AlertTriangle } from 'lucide-react'
import { useBuild, useDashboard, useUpcomingChanges } from '../api/hooks'
import { ProvisioningProgress, StateByTeam, FailuresByCause, AgentActivity } from '../components/DashboardCharts'
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

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <ProvisioningProgress data={dash} />
        <AgentActivity data={dash} />
      </div>
      <FailuresByCause data={dash} />
      <StateByTeam data={dash} />
    </div>
  )
}
