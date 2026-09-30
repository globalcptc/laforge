import { useParams } from '@tanstack/react-router'
import { useBuild, useDashboard, useUpcomingChanges } from '../api/hooks'
import { ProvisioningProgress, StateByTeam, FailuresByCause, AgentActivity } from '../components/DashboardCharts'
import { UpcomingChangesPanel } from '../components/UpcomingChangesPanel'
import { Card, cn, Spinner } from '../ui'

// "Build → Overview: health band -- a row of stat tiles and a few
// charts, all fed by server-side aggregates rather than counted in the
// browser." The stat tiles and all four named charts
// (provisioning progress, agent check-ins, state by team, failures by
// cause) now read from the real GET /builds/{id}/dashboard aggregate
// (internal/api/dashboard.go) -- an earlier version of this screen
// counted client-side from the whole build payload, which
// the spec's own "Performance" section calls out as the wrong long-term
// answer; still not built here are the top bar's own countdown and
// true heartbeat-history charting.
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

  const s = dash.provisioning_by_status
  const total = Object.values(s).reduce((a, b) => a + b, 0)
  const finished = s.finished ?? 0
  // Every terminal failure, regardless of which layer it failed at.
  const failed = (s.deploy_failed ?? 0) + (s.build_failed ?? 0) + (s.invalid ?? 0)
  // Everything not yet at a terminal state: infra still coming up, or hosts
  // still building. Deliberately kept apart from "finished" -- an object that
  // is up but still building is not done.
  const inProgress = (s.pending ?? 0) + (s.deploying ?? 0) + (s.running ?? 0) + (s.building ?? 0)
  const teamsOpen = build.teams.filter((t) => t.access_state === 'open').length

  return (
    <div className="flex flex-col gap-4">
      {upcoming && <UpcomingChangesPanel buildId={buildId} data={upcoming} />}

      <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
        <StatTile label="Finished" value={`${finished} / ${total}`} />
        <StatTile label="In progress" value={inProgress} />
        <StatTile label="Failures" value={failed} tone={failed > 0 ? 'bad' : 'good'} />
        <StatTile label="Teams open" value={`${teamsOpen} / ${build.teams.length}`} />
      </div>

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <ProvisioningProgress data={dash} />
        <StateByTeam data={dash} />
        <AgentActivity data={dash} />
        <FailuresByCause data={dash} />
      </div>

    </div>
  )
}

function StatTile({ label, value, tone }: { label: string; value: string | number; tone?: 'good' | 'bad' }) {
  return (
    <Card className="px-4 py-3">
      <div className="text-xs font-medium uppercase tracking-wide text-fg-muted">{label}</div>
      <div className={cn('mt-1 text-2xl font-semibold text-fg', tone === 'bad' && 'text-danger', tone === 'good' && 'text-success')}>{value}</div>
    </Card>
  )
}
