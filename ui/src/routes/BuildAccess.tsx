import { useParams } from '@tanstack/react-router'
import { useEffect, useRef, useState, type ReactNode } from 'react'
import { ChevronDown, Clock, Lock, Unlock } from 'lucide-react'
import { useBuild, useSetTeamAccess } from '../api/hooks'
import { useTimeFormat } from '../lib/time'
import { StatusBadge } from '../components/StatusBadge'
import { AccessCountdown } from '../components/AccessCountdown'
import { AnchoredPopover, Button, buttonClass, Card, cn, Spinner, useToast } from '../ui'
import { buildIsReadOnly } from '../lib/build'
import type { AccessChange, Team } from '../api/types'

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
  // Which team's Extend / Reduce picker is open, if any.
  const [extendFor, setExtendFor] = useState<number | null>(null)
  const [reduceFor, setReduceFor] = useState<number | null>(null)
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
  const changing = build.teams.flatMap((t) => (t.access_change && t.access_change.state !== 'failed' ? [t.access_change] : []))

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
          <span className="flex items-center gap-2 text-sm font-medium text-fg">
            All Teams
            {changing.length > 0 && (
              <span className="text-xs font-normal text-fg-muted">
                {summarizeChanges(changing)}
              </span>
            )}
          </span>
          <div className="flex items-center gap-2">
            <ActionMenu label="Access" icon={<Unlock size={14} />} disabled={setAccess.isPending}>
              {(close) => (
                <>
                  <MenuItem
                    onClick={() => {
                      close()
                      toggleAll('open')
                    }}
                  >
                    <Unlock size={14} className="text-accent" /> Open All
                  </MenuItem>
                  <MenuItem
                    onClick={() => {
                      close()
                      toggleAll('close')
                    }}
                  >
                    <Lock size={14} className="text-orange" /> Close All
                  </MenuItem>
                </>
              )}
            </ActionMenu>
            <ActionMenu label="Time" icon={<Clock size={14} />}>
              {(close) => (
                <>
                  <MinutesRow
                    heading="Extend All"
                    sign="+"
                    onPick={(mins) => {
                      close()
                      void adjustAll('extend', mins)
                    }}
                  />
                  <MinutesRow
                    heading="Reduce All"
                    sign="−"
                    variant="secondary"
                    onPick={(mins) => {
                      close()
                      void adjustAll('reduce', mins)
                    }}
                  />
                </>
              )}
            </ActionMenu>
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
              <StatusBadge status={accessBadge(team)} />
              {team.access_change && <AccessChangeNote change={team.access_change} />}
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
        {MINUTE_CHOICES.map((mins) => (
          <Button key={mins} variant={choiceVariant} size="sm" className="shrink-0" tabIndex={open ? undefined : -1} onClick={() => onPick(mins)}>
            {sign}
            {mins}m
          </Button>
        ))}
      </div>
    </div>
  )
}

const MINUTE_CHOICES = [1, 5, 15, 30, 60]

// ActionMenu is a dropdown of actions for the All Teams row: a coloured trigger
// that opens a panel below it, closed by a click away or by an action (each
// action gets `close`).
function ActionMenu({ label, icon, disabled, children }: { label: string; icon: ReactNode; disabled?: boolean; children: (close: () => void) => ReactNode }) {
  const [open, setOpen] = useState(false)
  const anchorRef = useRef<HTMLButtonElement>(null)
  const panelRef = useRef<HTMLDivElement>(null)
  useEffect(() => {
    if (!open) return
    function onDown(e: MouseEvent) {
      const t = e.target as Node
      if (anchorRef.current?.contains(t) || panelRef.current?.contains(t)) return
      setOpen(false)
    }
    window.addEventListener('mousedown', onDown)
    return () => window.removeEventListener('mousedown', onDown)
  }, [open])
  return (
    <>
      <button ref={anchorRef} type="button" className={buttonClass({ variant: 'primary', size: 'sm' })} onClick={() => setOpen((o) => !o)} disabled={disabled} aria-expanded={open} aria-haspopup="menu">
        {icon} {label}
        <ChevronDown size={13} className={cn('transition-transform', open && 'rotate-180')} />
      </button>
      {open && (
        <AnchoredPopover anchorRef={anchorRef} panelRef={panelRef} minWidth={180} maxHeight={320}>
          <div className="py-1" role="menu">
            {children(() => setOpen(false))}
          </div>
        </AnchoredPopover>
      )}
    </>
  )
}

function MenuItem({ onClick, children }: { onClick: () => void; children: ReactNode }) {
  return (
    <button type="button" role="menuitem" onClick={onClick} className="flex w-full items-center gap-2 px-2.5 py-1.5 text-left text-sm text-fg hover:bg-surface-hover">
      {children}
    </button>
  )
}

// MinutesRow is one labelled row of +Nm / −Nm choices inside the Time menu.
function MinutesRow({ heading, sign, variant, onPick }: { heading: string; sign: string; variant?: 'secondary'; onPick: (mins: number) => void }) {
  return (
    <div className="px-2.5 py-1.5">
      <div className="pb-1 text-[11px] font-semibold uppercase tracking-wide text-fg-subtle">{heading}</div>
      <div className="flex items-center gap-1">
        {MINUTE_CHOICES.map((mins) => (
          <Button key={mins} variant={variant} size="sm" role="menuitem" onClick={() => onPick(mins)}>
            {sign}
            {mins}m
          </Button>
        ))}
      </div>
    </div>
  )
}

// accessBadge is the state a team's badge shows: Opening / Closing while a change
// is queued or running on the builder (access_state only flips once it
// finishes), Open Failed / Close Failed when the last attempt gave up, else the
// settled Open / Closed.
function accessBadge(team: Team): 'open' | 'closed' | 'opening' | 'closing' | 'open_failed' | 'close_failed' {
  const c = team.access_change
  if (!c) return team.access_state
  if (c.state === 'failed') return c.direction === 'opening' ? 'open_failed' : 'close_failed'
  return c.direction
}

function ago(iso: string): string {
  const secs = Math.max(0, Math.round((Date.now() - new Date(iso).getTime()) / 1000))
  if (secs < 60) return `${secs}s`
  const mins = Math.floor(secs / 60)
  return mins < 60 ? `${mins}m` : `${Math.floor(mins / 60)}h ${mins % 60}m`
}

// AccessChangeNote says where a change stands: waiting for a runner, being
// applied (and for how long -- a big team takes minutes), or why it failed.
function AccessChangeNote({ change }: { change: AccessChange }) {
  if (change.state === 'failed') {
    return (
      <span className="max-w-md truncate text-xs text-danger" title={change.error}>
        Failed {ago(change.since)} ago{change.error ? `: ${change.error}` : ''}
      </span>
    )
  }
  const retry = change.attempts && change.attempts > 1 ? `, attempt ${change.attempts}` : ''
  return (
    <span className="text-xs text-fg-muted">
      {change.state === 'running' ? `Applying on the builder for ${ago(change.since)}${retry}` : `Queued for ${ago(change.since)}`}
    </span>
  )
}

function summarizeChanges(changes: AccessChange[]): string {
  const closing = changes.filter((c) => c.direction === 'closing').length
  const opening = changes.length - closing
  return [closing > 0 && `${closing} closing`, opening > 0 && `${opening} opening`].filter(Boolean).join(', ')
}
