import { useMemo, useState } from 'react'
import { ChevronDown, ChevronRight, GitCommitHorizontal } from 'lucide-react'
import type { UpcomingChanges } from '../api/types'
import { useApplyUpcoming } from '../api/hooks'
import { ApiError } from '../api/client'
import { Badge, Button, Card, cn, useToast, type BadgeProps } from '../ui'

const CHANGE_STYLE: Record<string, { label: string; tone: BadgeProps['tone'] }> = {
  new: { label: 'would create', tone: 'success' },
  changed: { label: 'would rebuild', tone: 'warning' },
  removed: { label: 'would destroy', tone: 'danger' },
}

// "If a newer commit has built, what deploying it would alter, at
// environment, team, network, and host level" -- real output of
// GET /builds/{id}/upcoming
// (internal/orchestrator.DiffUpcoming), grouped here by team since the
// API itself returns a flat list at the same granularity Reconcile
// operates at. Renders nothing at all when there's nothing pending, so
// it never clutters a build that's simply up to date. The "Apply now"
// button is the deliberate action this preview exists to inform (see
// POST /builds/{id}/apply-upcoming's own doc comment) -- reuses this
// build's own row in place, not a new build.
export function UpcomingChangesPanel({ buildId, data }: { buildId: string; data: UpcomingChanges }) {
  const [expanded, setExpanded] = useState<Set<number>>(new Set())
  const apply = useApplyUpcoming(buildId)
  const [applyError, setApplyError] = useState<string | null>(null)
  const toast = useToast()

  async function onApply() {
    setApplyError(null)
    try {
      await apply.mutateAsync()
      toast({ title: 'Changes applied', tone: 'success' })
    } catch (e) {
      const message = e instanceof ApiError ? e.message : 'Apply failed'
      setApplyError(message)
      toast({ title: 'Apply failed', description: message, tone: 'danger', duration: 0 })
    }
  }

  const byTeam = useMemo(() => {
    const m = new Map<number, UpcomingChanges['changes']>()
    for (const c of data.changes) {
      const arr = m.get(c.team) ?? []
      arr.push(c)
      m.set(c.team, arr)
    }
    return [...m.entries()].sort((a, b) => a[0] - b[0])
  }, [data.changes])

  if (!data.pending) return null

  if (data.changes.length === 0) {
    return (
      <Card className="p-4">
        <div className="flex items-center gap-2 text-sm text-fg">
          <GitCommitHorizontal size={14} className="text-accent" />
          A newer commit has been validated for this build's branch, but it wouldn't change anything currently
          deployed.
          <ApplyButton apply={apply} onApply={onApply} className="ml-auto" />
        </div>
        {applyError && <div className="mt-2 text-xs text-danger">{applyError}</div>}
      </Card>
    )
  }

  return (
    <Card className="border-warning/30 p-4">
      <div className="mb-3 flex items-center gap-2 text-sm font-medium text-fg">
        <GitCommitHorizontal size={14} className="text-warning" />
        Upcoming changes -- a newer commit would alter {data.changes.length} object{data.changes.length === 1 ? '' : 's'}
        <ApplyButton apply={apply} onApply={onApply} className="ml-auto" />
      </div>
      {applyError && <div className="mb-2 text-xs text-danger">{applyError}</div>}
      <div className="flex flex-col gap-1">
        {byTeam.map(([team, changes]) => {
          const isOpen = expanded.has(team)
          return (
            <div key={team}>
              <button
                onClick={() =>
                  setExpanded((prev) => {
                    const next = new Set(prev)
                    next.has(team) ? next.delete(team) : next.add(team)
                    return next
                  })
                }
                className="flex w-full items-center gap-2 rounded-token px-1 py-1 text-left text-xs hover:bg-surface-hover"
              >
                {isOpen ? <ChevronDown size={12} /> : <ChevronRight size={12} />}
                <span className="font-medium text-fg">Team {team}</span>
                <span className="text-fg-muted">
                  {changes.length} object{changes.length === 1 ? '' : 's'}
                </span>
              </button>
              {isOpen && (
                <div className="ml-5 flex flex-col gap-1 py-1">
                  {changes.map((c, i) => {
                    const style = CHANGE_STYLE[c.change] ?? { label: c.change, tone: 'neutral' as const }
                    return (
                      <div key={i} className="flex items-center gap-2 text-xs">
                        <Badge tone={style.tone} className="rounded-full">
                          {style.label}
                        </Badge>
                        <span className="font-mono text-fg">{c.as_name || c.object_name}</span>
                        {c.network_name && <span className="text-fg-muted">on {c.network_name}</span>}
                      </div>
                    )
                  })}
                </div>
              )}
            </div>
          )
        })}
      </div>
    </Card>
  )
}

function ApplyButton({ apply, onApply, className }: { apply: { isPending: boolean }; onApply: () => void; className?: string }) {
  return (
    <Button size="sm" onClick={onApply} disabled={apply.isPending} className={cn(className)}>
      {apply.isPending ? 'Applying…' : 'Apply Now'}
    </Button>
  )
}
