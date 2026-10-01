import { useParams } from '@tanstack/react-router'
import { useEffect, useState } from 'react'
import { Lock, Unlock, Clock, Infinity as InfinityIcon } from 'lucide-react'
import { useBuild, useSetTeamAccess } from '../api/hooks'
import { useTimeFormat } from '../lib/time'
import { StatusBadge } from '../components/StatusBadge'
import { AccessCountdown } from '../components/AccessCountdown'
import type { AccessWindow, Team } from '../api/types'
import { Button, Card, cn, Spinner } from '../ui'
import { buildIsReadOnly } from '../lib/build'

// currentAccessEnd is when this team's access is scheduled to end, but ONLY
// when the team is actually open right now (access_state is authoritative --
// the schedule reconciler drives it). A closed team returns no end at all: the
// countdown next to it shows when it next opens instead, so we never render an
// "ends at" that reads as if a closed team were open. When open, the end is a
// live open-direction override (an extension) if one reaches furthest, else the
// close of the current window. `unlimited` means the team is open with nothing
// scheduled to end it (no windows, no override) -- it stays open until closed
// by hand.
function currentAccessEnd(windows: AccessWindow[] | undefined, team: Team): { end: Date | null; unlimited: boolean; extended: boolean } {
  if (team.access_state !== 'open') return { end: null, unlimited: false, extended: false }

  const now = new Date()
  const override = team.access_override_until ? new Date(team.access_override_until) : null
  const overrideOpenActive = team.access_override_state === 'open' && override !== null && override > now
  const parsed = (windows ?? []).map((w) => ({ open: new Date(w.open), close: new Date(w.close) }))
  const active = parsed.find((w) => now >= w.open && now <= w.close)

  if (overrideOpenActive) {
    const extended = !active || override! > active.close
    return { end: override!, unlimited: false, extended }
  }
  if (active) return { end: active.close, unlimited: false, extended: false }
  return { end: null, unlimited: true, extended: false }
}

// "Build → Access": current state per team, and
// "extend team 4 by 30 minutes is one control, not a form." Every
// action here drives the real close_access/open_access task the runner
// already executes against the real builder -- see
// internal/api/tasks.go's own doc comment.
export function BuildAccess() {
  const { buildId } = useParams({ from: '/repos/$repoId/builds/$buildId/access' })
  const { data: build } = useBuild(buildId)
  const setAccess = useSetTeamAccess(buildId)
  const [extendFor, setExtendFor] = useState<number | null>(null)
  const [reduceFor, setReduceFor] = useState<number | null>(null)
  // The expanded +Nm / -Nm choices collapse back to the button after 30s of no
  // choice, so a half-open control doesn't linger.
  useEffect(() => {
    if (extendFor === null) return
    const t = setTimeout(() => setExtendFor(null), 30_000)
    return () => clearTimeout(t)
  }, [extendFor])
  useEffect(() => {
    if (reduceFor === null) return
    const t = setTimeout(() => setReduceFor(null), 30_000)
    return () => clearTimeout(t)
  }, [reduceFor])
  const fmt = useTimeFormat()

  if (!build)
    return (
      <div className="flex items-center gap-2 text-sm text-fg-muted">
        <Spinner /> Loading…
      </div>
    )

  const readOnly = buildIsReadOnly(build.status)

  return (
    <div className="flex flex-col gap-3">
      {readOnly && (
        <div className="rounded-token border border-border bg-surface-sunken px-3 py-2 text-xs text-fg-muted">
          This build has been torn down — access is read-only.
        </div>
      )}
      {build.access && build.access.length > 0 && (
        <Card className="px-4 py-2">
          <AccessCountdown windows={build.access} />
        </Card>
      )}
      {build.teams.map((team) => {
        const accessEnd = currentAccessEnd(build.access, team)
        const extending = extendFor === team.team_number
        const reducing = reduceFor === team.team_number
        return (
          <Card key={team.id} className="flex items-center justify-between px-4 py-3">
            <div className="flex items-center gap-3">
              <span className="font-medium text-fg">Team {team.team_number}</span>
              <StatusBadge status={team.access_state} />
              {/* Each team's own next transition -- "the current start or
                  end, depending on which is next" -- computed against the
                  same authored windows but this team's own override, so an
                  extended team shows its later close, not the schedule's. */}
              <AccessCountdown windows={build.access} overrideUntil={team.access_override_until} state={team.access_state} />
              {/* The explicit current access end: an exact time, or a note
                  that access is unlimited when nothing is scheduled to end it. */}
              {accessEnd.unlimited ? (
                <span className="flex items-center gap-1 text-xs text-fg-muted">
                  <InfinityIcon size={12} />
                  No time limit
                </span>
              ) : accessEnd.end ? (
                <span className="flex items-center gap-1 text-xs text-fg-muted">
                  <Clock size={12} />
                  ends {fmt.dateTime(accessEnd.end.toISOString())}
                  {accessEnd.extended && <span className="text-warning">(extended)</span>}
                </span>
              ) : null}
            </div>
            {!readOnly && (
              <div className="flex items-center gap-2">
                <Button variant="primary" size="sm" onClick={() => setAccess.mutate({ team: team.team_number, action: 'open' })} disabled={setAccess.isPending}>
                  <Unlock size={14} /> Open
                </Button>
                <Button variant="secondary" size="sm" onClick={() => setAccess.mutate({ team: team.team_number, action: 'close' })} disabled={setAccess.isPending}>
                  <Lock size={14} /> Close
                </Button>
                {/* Extend collapses to its +Nm choices and back, animated both
                    ways: each side transitions its own max-width + opacity so
                    one slides open as the other slides shut. */}
                <div className="flex items-center gap-1">
                  <div className={cn('overflow-hidden transition-all duration-500 ease-in-out', extending ? 'max-w-0 opacity-0' : 'max-w-[7rem] opacity-100')}>
                    <Button variant="ghost" size="sm" className="shrink-0" tabIndex={extending ? -1 : undefined} onClick={() => setExtendFor(team.team_number)}>
                      <Clock size={14} /> Extend
                    </Button>
                  </div>
                  <div className={cn('flex items-center gap-1 overflow-hidden transition-all duration-500 ease-in-out', extending ? 'max-w-md opacity-100' : 'pointer-events-none max-w-0 opacity-0')}>
                    <span className="shrink-0 whitespace-nowrap pl-0.5 text-xs text-fg-muted">Extend by</span>
                    {[1, 5, 15, 30, 60].map((mins) => (
                      <Button
                        key={mins}
                        size="sm"
                        className="shrink-0"
                        tabIndex={extending ? undefined : -1}
                        onClick={() => {
                          setAccess.mutate({ team: team.team_number, action: 'extend', extendMinutes: mins })
                          setExtendFor(null)
                        }}
                      >
                        +{mins}m
                      </Button>
                    ))}
                  </div>
                </div>
                {/* Reduce is a penalty (close the team for N more minutes),
                    always available -- a team may be penalized whether or not
                    it's currently extended. Mirrors Extend. */}
                <div className="flex items-center gap-1">
                  <div className={cn('overflow-hidden transition-all duration-500 ease-in-out', reducing ? 'max-w-0 opacity-0' : 'max-w-[7rem] opacity-100')}>
                    <Button variant="ghost" size="sm" className="shrink-0" tabIndex={reducing ? -1 : undefined} onClick={() => setReduceFor(team.team_number)}>
                      <Clock size={14} /> Reduce
                    </Button>
                  </div>
                  <div className={cn('flex items-center gap-1 overflow-hidden transition-all duration-500 ease-in-out', reducing ? 'max-w-md opacity-100' : 'pointer-events-none max-w-0 opacity-0')}>
                    <span className="shrink-0 whitespace-nowrap pl-0.5 text-xs text-fg-muted">Reduce by</span>
                    {[1, 5, 15, 30, 60].map((mins) => (
                      <Button
                        key={mins}
                        variant="secondary"
                        size="sm"
                        className="shrink-0"
                        tabIndex={reducing ? undefined : -1}
                        onClick={() => {
                          setAccess.mutate({ team: team.team_number, action: 'reduce', reduceMinutes: mins })
                          setReduceFor(null)
                        }}
                      >
                        −{mins}m
                      </Button>
                    ))}
                  </div>
                </div>
              </div>
            )}
          </Card>
        )
      })}
    </div>
  )
}
