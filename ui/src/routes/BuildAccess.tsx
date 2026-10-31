import { useParams } from '@tanstack/react-router'
import { useEffect, useState } from 'react'
import { Lock, Unlock, Clock } from 'lucide-react'
import { useBuild, useSetTeamAccess } from '../api/hooks'
import { useTimeFormat } from '../lib/time'
import { StatusBadge } from '../components/StatusBadge'
import { AccessCountdown } from '../components/AccessCountdown'
import { Button, Card, cn, Spinner, useToast } from '../ui'
import { buildIsReadOnly } from '../lib/build'

// "Build → Access": current state per team, and
// "extend team 4 by 30 minutes is one control, not a form." Every
// action here drives the real close_access/open_access task the runner
// already executes against the real builder -- see
// internal/api/tasks.go's own doc comment.
export function BuildAccess() {
  const { buildId } = useParams({ from: '/repos/$repoId/builds/$buildId/access' })
  const { data: build } = useBuild(buildId)
  const setAccess = useSetTeamAccess(buildId)
  const toast = useToast()
  // One button per team that toggles the team's access. The real state flips
  // asynchronously (the runner applies it on the builder), so we fire an
  // immediate "changing" toast on click and a result toast when the request is
  // accepted or fails -- otherwise the click looks like it did nothing.
  const toggleAccess = (teamNumber: number, state: string) => {
    const action = state === 'open' ? 'close' : 'open'
    toast({ title: `Team ${teamNumber}: ${action === 'close' ? 'closing' : 'opening'} access…`, tone: 'info' })
    setAccess.mutate(
      { team: teamNumber, action },
      {
        onSuccess: () =>
          toast({ title: `Team ${teamNumber} ${action === 'close' ? 'closing' : 'opening'}`, description: 'Applying on the builder — the badge updates when it takes effect.', tone: 'success' }),
        onError: (e) => toast({ title: `Team ${teamNumber}: ${action} failed`, description: e instanceof Error ? e.message : 'error', tone: 'danger' }),
      },
    )
  }
  // Bulk open/close every team at once. One toast for the whole batch (N toasts
  // would be noise); each team's badge updates on its own as it takes effect.
  const toggleAll = (action: 'open' | 'close') => {
    if (!build) return
    toast({ title: `${action === 'close' ? 'Closing' : 'Opening'} all ${build.teams.length} teams…`, tone: 'info' })
    for (const team of build.teams) {
      setAccess.mutate({ team: team.team_number, action })
    }
  }
  // Which picker is open: a team number, 'all' for the All Teams row, or none.
  const [extendFor, setExtendFor] = useState<number | 'all' | null>(null)
  const [reduceFor, setReduceFor] = useState<number | 'all' | null>(null)
  // Extend All / Reduce All move every open team's OWN end by the same number
  // of minutes -- the server computes each team's current end (its override or
  // its scheduled window close), so teams already on different end times keep
  // their offsets. Closed teams are skipped: they have no end to move, and
  // extending one would reopen it. One summary toast for the batch.
  const adjustAll = async (action: 'extend' | 'reduce', mins: number) => {
    if (!build) return
    const open = build.teams.filter((t) => t.access_state === 'open')
    const skipped = build.teams.filter((t) => t.access_state !== 'open').map((t) => t.team_number)
    if (open.length === 0) {
      toast({ title: `No open teams to ${action}`, description: 'Every team is closed.', tone: 'info' })
      return
    }
    const verb = action === 'extend' ? 'Extending' : 'Reducing'
    toast({ title: `${verb} ${open.length} team${open.length === 1 ? '' : 's'} by ${mins}m…`, tone: 'info' })
    const results = await Promise.allSettled(
      open.map((t) =>
        setAccess.mutateAsync(action === 'extend' ? { team: t.team_number, action, extendMinutes: mins } : { team: t.team_number, action, reduceMinutes: mins }),
      ),
    )
    const failed = results.flatMap((r, i) => (r.status === 'rejected' ? [`team ${open[i].team_number} (${r.reason instanceof Error ? r.reason.message : 'error'})`] : []))
    const done = open.length - failed.length
    const notes = [skipped.length > 0 && `Skipped closed team${skipped.length === 1 ? '' : 's'} ${skipped.join(', ')}.`, failed.length > 0 && `Failed: ${failed.join('; ')}.`].filter(Boolean).join(' ')
    toast({
      title: `${action === 'extend' ? 'Extended' : 'Reduced'} ${done} team${done === 1 ? '' : 's'} by ${mins}m`,
      description: notes || "Each team's own end time moved.",
      tone: failed.length > 0 ? 'danger' : 'success',
    })
  }
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
          <AccessCountdown windows={build.access} formatAt={(d) => fmt.dateTime(d.toISOString())} />
        </Card>
      )}
      {!readOnly && build.teams.length > 0 && (
        <div className="flex items-center justify-between">
          <span className="text-sm font-medium text-fg">All teams</span>
          <div className="flex items-center gap-2">
            <Button variant="primary" size="sm" onClick={() => toggleAll('open')} disabled={setAccess.isPending}>
              <Unlock size={14} /> Open All
            </Button>
            <Button variant="danger" size="sm" className="bg-orange hover:brightness-95" onClick={() => toggleAll('close')} disabled={setAccess.isPending}>
              <Lock size={14} /> Close All
            </Button>
            <MinutesPicker
              label="Extend All"
              prompt="Extend all by"
              sign="+"
              open={extendFor === 'all'}
              onOpen={() => setExtendFor('all')}
              onPick={(mins) => {
                setExtendFor(null)
                void adjustAll('extend', mins)
              }}
            />
            <MinutesPicker
              label="Reduce All"
              prompt="Reduce all by"
              sign="−"
              choiceVariant="secondary"
              open={reduceFor === 'all'}
              onOpen={() => setReduceFor('all')}
              onPick={(mins) => {
                setReduceFor(null)
                void adjustAll('reduce', mins)
              }}
            />
          </div>
        </div>
      )}
      {build.teams.map((team) => {
        const extending = extendFor === team.team_number
        const reducing = reduceFor === team.team_number
        return (
          <Card key={team.id} className="flex items-center justify-between px-4 py-3">
            <div className="flex items-center gap-3">
              <span className="font-medium text-fg">Team {team.team_number}</span>
              <StatusBadge status={team.access_state} />
              {/* The next scheduled window transition as one line (datetime +
                  countdown), using this team's own override so a genuine
                  extension shows its later close. */}
              <AccessCountdown
                windows={build.access}
                overrideUntil={team.access_override_until}
                overrideState={team.access_override_state}
                formatAt={(d) => fmt.dateTime(d.toISOString())}
              />
            </div>
            {!readOnly && (
              <div className="flex items-center gap-2">
                {/* One button that toggles: an open team shows Close (danger),
                    a closed team shows Open (accent). */}
                {team.access_state === 'open' ? (
                  <Button variant="danger" size="sm" className="bg-orange hover:brightness-95" onClick={() => toggleAccess(team.team_number, team.access_state)} disabled={setAccess.isPending}>
                    <Lock size={14} /> Close
                  </Button>
                ) : (
                  <Button variant="primary" size="sm" onClick={() => toggleAccess(team.team_number, team.access_state)} disabled={setAccess.isPending}>
                    <Unlock size={14} /> Open
                  </Button>
                )}
                <MinutesPicker
                  label="Extend"
                  prompt="Extend by"
                  sign="+"
                  open={extending}
                  onOpen={() => setExtendFor(team.team_number)}
                  onPick={(mins) => {
                    setAccess.mutate({ team: team.team_number, action: 'extend', extendMinutes: mins })
                    setExtendFor(null)
                  }}
                />
                {/* Reduce is a penalty (close the team for N more minutes),
                    always available -- a team may be penalized whether or not
                    it's currently extended. Mirrors Extend. */}
                <MinutesPicker
                  label="Reduce"
                  prompt="Reduce by"
                  sign="−"
                  choiceVariant="secondary"
                  open={reducing}
                  onOpen={() => setReduceFor(team.team_number)}
                  onPick={(mins) => {
                    setAccess.mutate({ team: team.team_number, action: 'reduce', reduceMinutes: mins })
                    setReduceFor(null)
                  }}
                />
              </div>
            )}
          </Card>
        )
      })}
    </div>
  )
}

// MinutesPicker is the Extend / Reduce control: a button that slides open into
// its +Nm / −Nm choices and back, animated both ways -- each side transitions
// its own max-width + opacity so one slides open as the other slides shut.
function MinutesPicker({
  label,
  prompt,
  sign,
  open,
  onOpen,
  onPick,
  choiceVariant,
}: {
  label: string
  prompt: string
  sign: string
  open: boolean
  onOpen: () => void
  onPick: (mins: number) => void
  choiceVariant?: 'secondary'
}) {
  return (
    <div className="flex items-center gap-1">
      <div className={cn('overflow-hidden transition-all duration-500 ease-in-out', open ? 'max-w-0 opacity-0' : 'max-w-[9rem] opacity-100')}>
        <Button variant="ghost" size="sm" className="shrink-0" tabIndex={open ? -1 : undefined} onClick={onOpen}>
          <Clock size={14} /> {label}
        </Button>
      </div>
      <div className={cn('flex items-center gap-1 overflow-hidden transition-all duration-500 ease-in-out', open ? 'max-w-md opacity-100' : 'pointer-events-none max-w-0 opacity-0')}>
        <span className="shrink-0 whitespace-nowrap pl-0.5 text-xs text-fg-muted">{prompt}</span>
        {[1, 5, 15, 30, 60].map((mins) => (
          <Button key={mins} variant={choiceVariant} size="sm" className="shrink-0" tabIndex={open ? undefined : -1} onClick={() => onPick(mins)}>
            {sign}
            {mins}m
          </Button>
        ))}
      </div>
    </div>
  )
}
