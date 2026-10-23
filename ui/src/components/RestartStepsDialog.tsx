import { useEffect, useState } from 'react'
import { AlertTriangle, Check, RotateCcw } from 'lucide-react'
import { useRestartSteps } from '../api/hooks'
import { ApiError } from '../api/client'
import type { DeployedObject, RestartHost } from '../api/types'
import { Badge, Button, Modal, Spinner, cn, useToast } from '../ui'
import { statusLabel } from './StatusBadge'

// canRestartSteps: the states "Restart from step" applies to -- a host whose
// build stopped, or a finished one whose steps changed in the content while
// the build holds hosts on step-only changes.
export function canRestartSteps(obj: Pick<DeployedObject, 'kind' | 'status' | 'steps_changed_at'>): boolean {
  if (obj.kind === 'network') return false
  if (obj.status === 'build_failed' || obj.status === 'invalid') return true
  return obj.status === 'finished' && !!obj.steps_changed_at
}

// RestartStepsDialog is the confirm step for "Restart from step"
// (docs/plans/restart-from-step.md): the dry run first -- which steps run
// again, every team's copy of this host with its state (the restartable ones
// ticked), the dependents that resume, and whether the content has changed
// since the hosts were built -- then the restart of the ticked instances.
export function RestartStepsDialog({
  buildId,
  seedIds,
  fromGroup,
  onClose,
  onDone,
}: {
  buildId: string
  // The host(s) the operator started from; the dialog widens to every
  // instance of that host definition and pre-ticks the restartable ones
  // among these (or all restartable ones when they're all restartable).
  seedIds: string[]
  fromGroup: number
  onClose: () => void
  onDone: () => void
}) {
  const restart = useRestartSteps(buildId)
  const toast = useToast()
  const [hosts, setHosts] = useState<RestartHost[] | null>(null)
  const [previewErr, setPreviewErr] = useState<string | null>(null)
  const [picked, setPicked] = useState<Set<string>>(new Set())
  const [force, setForce] = useState(false)
  const [resumeDependents, setResumeDependents] = useState(true)

  const seedKey = seedIds.join(',')
  useEffect(() => {
    let cancelled = false
    setHosts(null)
    setPreviewErr(null)
    restart
      .mutateAsync({ target: { ids: seedIds }, from_group: fromGroup, dry_run: true })
      .then((res) => {
        if (cancelled) return
        setHosts(res.hosts)
        const seeds = new Set(seedIds)
        const restartable = res.hosts.filter((h) => h.eligible && !h.skipped)
        const fromSeeds = restartable.filter((h) => seeds.has(h.id))
        setPicked(new Set((fromSeeds.length > 0 ? fromSeeds : restartable).map((h) => h.id)))
      })
      .catch((e) => {
        if (!cancelled) setPreviewErr(e instanceof ApiError ? e.message : 'Could not work out what would restart')
      })
    return () => {
      cancelled = true
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [buildId, seedKey, fromGroup])

  const steps = hosts?.find((h) => h.steps.length > 0)?.steps ?? []
  const name = hosts?.[0]?.as_name || hosts?.[0]?.object_name || ''
  const contentChanged = hosts?.some((h) => picked.has(h.id) && h.content_changed) ?? false
  const dependents = hosts?.filter((h) => picked.has(h.id)).reduce((n, h) => n + h.dependents.length, 0) ?? 0
  const runningPicked = hosts?.filter((h) => picked.has(h.id) && h.running).length ?? 0

  function toggle(id: string) {
    setPicked((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  async function run() {
    try {
      const res = await restart.mutateAsync({ target: { ids: [...picked] }, from_group: fromGroup, resume_dependents: resumeDependents, force })
      const skipped = res.hosts.filter((h) => picked.has(h.id) && !h.restarted)
      toast({
        title: `Restarted ${res.restarted} host${res.restarted === 1 ? '' : 's'} from step ${fromGroup}`,
        description: skipped.length > 0 ? `Skipped: ${skipped.map((h) => `${h.as_name} (${h.skipped})`).join('; ')}` : `${name}: steps run again from "${steps[0]?.label ?? fromGroup}"`,
        tone: skipped.length > 0 ? 'info' : 'success',
      })
      onDone()
    } catch (e) {
      toast({ title: 'Restart failed', description: e instanceof ApiError ? e.message : undefined, tone: 'danger', duration: 0 })
    }
  }

  const label = steps[0]?.label ? `"${steps[0].label}"` : `step ${fromGroup}`
  return (
    <Modal
      open
      onClose={onClose}
      title={`Restart ${name || 'host'} from ${label}`}
      footer={
        <>
          <Button variant="ghost" onClick={onClose}>
            Cancel
          </Button>
          <Button variant="brand" onClick={run} disabled={restart.isPending || !hosts || picked.size === 0 || (runningPicked > 0 && !force)}>
            {restart.isPending && !hosts ? <Spinner /> : <RotateCcw size={14} />} Restart {picked.size} host{picked.size === 1 ? '' : 's'}
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-4 text-sm">
        {!hosts && !previewErr && (
          <div className="flex items-center gap-2 text-fg-muted">
            <Spinner /> Working out what would run again…
          </div>
        )}
        {previewErr && <div className="text-danger">{previewErr}</div>}
        {hosts && (
          <>
            <div>
              <div className="mb-1 text-xs font-semibold uppercase tracking-wide text-fg-subtle">Steps that run again, in full</div>
              {steps.length === 0 ? (
                <div className="text-fg-muted">Step {fromGroup} is past the last step on this host.</div>
              ) : (
                <ol className="flex flex-col gap-0.5">
                  {steps.map((s) => (
                    <li key={s.group} className="flex items-center gap-2">
                      <span className="w-6 text-right font-mono text-xs text-fg-subtle">{s.group}</span>
                      <span className="text-fg">{s.label || '(unnamed step)'}</span>
                      <span className="text-xs text-fg-subtle">
                        {s.commands} command{s.commands === 1 ? '' : 's'}
                      </span>
                    </li>
                  ))}
                </ol>
              )}
              <div className="mt-1.5 text-xs text-fg-muted">
                Downloads fetch again, scripts write and run again, validators re-check — nothing from the earlier attempt is kept. The boxes stay as they are;
                steps should be safe to run twice.
              </div>
            </div>

            <div>
              <div className="mb-1 text-xs font-semibold uppercase tracking-wide text-fg-subtle">Every team’s {hosts[0]?.object_name}</div>
              <ul className="flex flex-col gap-1">
                {hosts.map((h) => {
                  const restartable = h.eligible && !h.skipped
                  const on = picked.has(h.id)
                  return (
                    <li key={h.id}>
                      <button
                        type="button"
                        onClick={() => restartable && toggle(h.id)}
                        disabled={!restartable}
                        className={cn('flex w-full items-center gap-2 rounded-token px-1.5 py-1 text-left', restartable ? 'hover:bg-surface-hover' : 'opacity-70')}
                      >
                        <span className={cn('flex h-4 w-4 shrink-0 items-center justify-center rounded border', on ? 'border-accent bg-accent text-fg-inverted' : 'border-border')}>
                          {on && <Check size={11} />}
                        </span>
                        <span className="w-14 shrink-0 text-xs text-fg-subtle">team {h.team_number}</span>
                        <span className="text-fg">{h.as_name}</span>
                        <Badge tone={h.status === 'build_failed' || h.status === 'invalid' ? 'danger' : h.status === 'finished' ? 'success' : 'neutral'}>{statusLabel(h.status)}</Badge>
                        {h.content_changed && <Badge tone="warning">steps changed</Badge>}
                        {h.running && <Badge tone="warning">a step is running</Badge>}
                        {h.skipped && <span className="min-w-0 flex-1 truncate text-xs text-fg-muted" title={h.skipped}>{h.skipped}</span>}
                      </button>
                    </li>
                  )
                })}
              </ul>
            </div>

            {contentChanged && (
              <div className="flex items-start gap-2 rounded-token border border-warning/40 bg-warning-soft p-2 text-xs text-fg">
                <AlertTriangle size={14} className="mt-0.5 shrink-0 text-warning" />
                <span>The content has changed since these hosts were built; the steps come from the current revision.</span>
              </div>
            )}

            <label className="flex items-center gap-2 text-xs text-fg-muted">
              <input type="checkbox" checked={resumeDependents} onChange={(e) => setResumeDependents(e.target.checked)} />
              Resume hosts that were failed for depending on these ({dependents} would resume once these finish)
            </label>
            {runningPicked > 0 && (
              <label className="flex items-center gap-2 text-xs text-warning">
                <input type="checkbox" checked={force} onChange={(e) => setForce(e.target.checked)} />
                A step is running on {runningPicked} of the picked hosts right now — restart anyway (its result is dropped)
              </label>
            )}
          </>
        )}
      </div>
    </Modal>
  )
}
