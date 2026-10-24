import { useParams } from '@tanstack/react-router'
import { Fragment, useMemo, useState } from 'react'
import { CalendarClock, Plus, X, ChevronRight, FileText } from 'lucide-react'
import { useBuild, useCancelScheduledTask, useCreateScheduledTask, useScheduledTasks, useScheduledTaskRuns } from '../api/hooks'
import { useTimeFormat } from '../lib/time'
import { buildIsReadOnly } from '../lib/build'
import { StatusBadge } from '../components/StatusBadge'
import { EmptyState } from '../components/EmptyState'
import { ApiError } from '../api/client'
import { TaskActionFields, buildTaskPayload, taskFormValid } from '../components/TaskActionForm'
import { WhenField } from '../components/WhenField'
import type { DeployedObject, ScheduledTask } from '../api/types'
import { Button, Label, Modal, Spinner, Table, TableScroller, Td, Th, cn } from '../ui'

// "Build → Schedule: injects, upcoming and past. Each shows its cron,
// next fire, which teams, and history. Manage access can cancel or
// reschedule." -- "cron" there predates the
// natural-language `when:` grammar; real dispatch is
// internal/orchestrator.DispatchDueScheduledTasks,
// the same CreateAgentTask path immediate ad-hoc dispatch uses.
//
// This page can now create a scheduled task directly, picking hosts here
// (direct product feedback) rather than only from a host selection on the
// Hosts page -- the same useCreateScheduledTask path either way.
export function BuildSchedule() {
  const { buildId } = useParams({ from: '/repos/$repoId/builds/$buildId/schedule' })
  const { data: build } = useBuild(buildId)
  const { data: tasks, isLoading } = useScheduledTasks(buildId)
  const cancel = useCancelScheduledTask(buildId)
  const [showNew, setShowNew] = useState(false)
  const readOnly = buildIsReadOnly(build?.status)

  const pending = (tasks ?? []).filter((t) => t.status === 'pending')
  const past = (tasks ?? []).filter((t) => t.status !== 'pending')

  return (
    <div className="flex flex-col gap-6">
      <div className="flex items-center justify-between">
        <div className="text-xs font-semibold uppercase tracking-wide text-fg-muted">Scheduled tasks</div>
        {!readOnly && (
          <Button variant="brand" size="sm" onClick={() => setShowNew(true)}>
            <Plus size={12} /> New Scheduled Task
          </Button>
        )}
      </div>

      {showNew && !readOnly && <NewScheduledTaskModal buildId={buildId} onClose={() => setShowNew(false)} />}

      {isLoading ? (
        <div className="flex items-center gap-2 text-sm text-fg-muted">
          <Spinner /> Loading…
        </div>
      ) : !tasks || tasks.length === 0 ? (
        <EmptyState
          icon={CalendarClock}
          title="No scheduled tasks"
          hint="Schedule one with the button above, or content-authored schedule: entries appear here once deployed."
        />
      ) : (
        <>
          <ScheduleTable buildId={buildId} title={`Upcoming (${pending.length})`} tasks={pending} onCancel={readOnly ? undefined : (id) => cancel.mutate(id)} cancelPending={cancel.isPending} />
          {past.length > 0 && <ScheduleTable buildId={buildId} title={`History (${past.length})`} tasks={past} />}
        </>
      )}
    </div>
  )
}

// The scheduled task's human label: what it actually does, not just the raw
// command. The payload carries the specifics (script name, url, args) -- shown
// as a compact summary so "the task" is visible at a glance.
function taskLabel(t: ScheduledTask): { title: string; detail: string } {
  const p = (t.payload && typeof t.payload === 'object' ? (t.payload as Record<string, unknown>) : {}) || {}
  const parts: string[] = []
  for (const [k, v] of Object.entries(p)) {
    if (v === null || v === undefined || v === '' || (Array.isArray(v) && v.length === 0)) continue
    parts.push(`${k}=${Array.isArray(v) ? v.join(',') : typeof v === 'object' ? JSON.stringify(v) : String(v)}`)
  }
  return { title: t.command, detail: parts.join(' · ') }
}

// A group is every team's copy of the same logical scheduled task: same host +
// content schedule entry (content), or same command+when dispatch (ad-hoc).
type Group = { key: string; rep: ScheduledTask; tasks: ScheduledTask[] }

function groupKey(t: ScheduledTask): string {
  if (t.source === 'content') return `c:${t.host_object_name ?? ''}:${t.schedule_index ?? ''}:${t.command}`
  return `a:${t.command}:${t.when_expr}:${t.created_at}`
}

function groupTasks(tasks: ScheduledTask[]): Group[] {
  const map = new Map<string, Group>()
  for (const t of tasks) {
    const key = groupKey(t)
    const g = map.get(key)
    if (g) g.tasks.push(t)
    else map.set(key, { key, rep: t, tasks: [t] })
  }
  for (const g of map.values()) g.tasks.sort((a, b) => (a.team_number ?? 0) - (b.team_number ?? 0))
  return [...map.values()]
}

function ScheduleTable({
  buildId,
  title,
  tasks,
  onCancel,
  cancelPending,
}: {
  buildId: string
  title: string
  tasks: ScheduledTask[]
  onCancel?: (id: string) => void
  cancelPending?: boolean
}) {
  const fmt = useTimeFormat()
  const [expanded, setExpanded] = useState<Set<string>>(new Set())
  const [runFor, setRunFor] = useState<ScheduledTask | null>(null)
  const groups = useMemo(() => groupTasks(tasks), [tasks])

  const toggle = (key: string) =>
    setExpanded((prev) => {
      const next = new Set(prev)
      next.has(key) ? next.delete(key) : next.add(key)
      return next
    })

  return (
    <div>
      <div className="mb-2 text-xs font-semibold uppercase tracking-wide text-fg-muted">{title}</div>
      {runFor && <RunModal buildId={buildId} task={runFor} onClose={() => setRunFor(null)} />}
      <TableScroller contain={false}>
        <Table>
          <thead>
            <tr>
              <Th className="w-8"></Th>
              <Th>Host</Th>
              <Th>Task</Th>
              <Th>When</Th>
              <Th>Next Fire</Th>
              <Th>Teams</Th>
              <Th>Runs</Th>
              {onCancel && <Th className="w-8"></Th>}
            </tr>
          </thead>
          <tbody>
            {groups.map((g) => {
              const lbl = taskLabel(g.rep)
              const open = expanded.has(g.key)
              const host = g.rep.host_as_name || g.rep.host_object_name || (g.rep.source === 'adhoc' ? 'Ad-hoc' : '—')
              const failed = g.tasks.reduce((n, t) => n + (t.run_failed || 0), 0)
              const ran = g.tasks.reduce((n, t) => n + (t.run_total || 0), 0)
              return (
                <Fragment key={g.key}>
                  <tr className="cursor-pointer hover:bg-surface-hover" onClick={() => toggle(g.key)}>
                    <Td>
                      <ChevronRight size={14} className={cn('text-fg-muted transition-transform', open && 'rotate-90')} />
                    </Td>
                    <Td className="font-medium text-fg">
                      {host}
                      {g.rep.host_kind && <span className="ml-1.5 text-xs font-normal text-fg-subtle">{g.rep.host_kind}</span>}
                    </Td>
                    <Td>
                      <span className="font-mono text-xs text-fg">{lbl.title}</span>
                      {lbl.detail && <span className="ml-1.5 font-mono text-[11px] text-fg-muted">{lbl.detail}</span>}
                      <span className="ml-1.5 text-[11px] text-fg-subtle">{g.rep.source === 'content' ? 'content' : g.rep.created_by ? `ad-hoc · ${g.rep.created_by}` : 'ad-hoc'}</span>
                    </Td>
                    <Td className="text-xs text-fg-muted">{g.rep.when_expr}</Td>
                    <Td className="text-xs text-fg-muted">{g.rep.next_fire_at ? fmt.dateTime(g.rep.next_fire_at) : '—'}</Td>
                    <Td className="text-xs text-fg-muted">{g.tasks.length}</Td>
                    <Td className="text-xs">
                      {ran === 0 ? (
                        <span className="text-fg-subtle">not yet</span>
                      ) : (
                        <span className={failed > 0 ? 'text-danger' : 'text-success'}>
                          {ran} run{ran === 1 ? '' : 's'}
                          {failed > 0 && ` · ${failed} failed`}
                        </span>
                      )}
                    </Td>
                    {onCancel && (
                      <Td>
                        <Button
                          variant="ghost"
                          size="icon"
                          onClick={(e) => {
                            e.stopPropagation()
                            for (const t of g.tasks) onCancel(t.id)
                          }}
                          disabled={cancelPending}
                          title="Cancel for all teams"
                          className="text-fg-muted hover:text-danger"
                        >
                          <X size={14} />
                        </Button>
                      </Td>
                    )}
                  </tr>
                  {open &&
                    g.tasks.map((t) => (
                      <tr key={t.id} className="bg-surface-sunken/50">
                        <Td></Td>
                        <Td className="text-xs text-fg-muted">Team {t.team_number ?? '—'}</Td>
                        <Td>
                          <StatusBadge status={t.status} />
                        </Td>
                        <Td colSpan={2} className="text-xs text-fg-muted">
                          {t.last_run_status ? (
                            <span className="inline-flex items-center gap-1.5">
                              last run <StatusBadge status={t.last_run_status} />
                              {t.last_run_finished_at && <span className="text-fg-subtle">{fmt.dateTime(t.last_run_finished_at)}</span>}
                            </span>
                          ) : (
                            <span className="text-fg-subtle">no runs yet</span>
                          )}
                        </Td>
                        <Td></Td>
                        <Td colSpan={onCancel ? 2 : 1}>
                          <Button variant="ghost" size="sm" disabled={t.run_total === 0} onClick={() => setRunFor(t)} className="text-xs">
                            <FileText size={13} /> Output
                          </Button>
                        </Td>
                      </tr>
                    ))}
                </Fragment>
              )
            })}
          </tbody>
        </Table>
      </TableScroller>
    </div>
  )
}

// RunModal shows a team's run history for one scheduled task: each agent_task's
// status, timing and console output (and any error) -- the "console output and
// log for each team" the Schedule page needed.
function RunModal({ buildId, task, onClose }: { buildId: string; task: ScheduledTask; onClose: () => void }) {
  const fmt = useTimeFormat()
  const { data: runs, isLoading } = useScheduledTaskRuns(buildId, task.id)
  const host = task.host_as_name || task.host_object_name || 'host'
  return (
    <Modal open onClose={onClose} title={`Team ${task.team_number ?? '—'} · ${host} · ${task.command}`}>
      <div className="flex max-h-[70vh] flex-col gap-4 overflow-auto">
        {isLoading ? (
          <div className="flex items-center gap-2 text-sm text-fg-muted">
            <Spinner /> Loading runs…
          </div>
        ) : !runs || runs.length === 0 ? (
          <div className="text-sm text-fg-muted">This task hasn’t run on this team yet.</div>
        ) : (
          runs.map((run) => (
            <div key={run.id} className="rounded-token border border-border">
              <div className="flex items-center gap-2 border-b border-border bg-surface-sunken px-3 py-2 text-xs">
                <StatusBadge status={run.status} />
                <span className="font-mono text-fg-muted">{run.command}</span>
                <span className="ml-auto text-fg-subtle">
                  {run.finished_at ? fmt.dateTime(run.finished_at) : run.started_at ? `started ${fmt.dateTime(run.started_at)}` : fmt.dateTime(run.created_at)}
                </span>
              </div>
              {run.last_error && <div className="border-b border-border bg-danger-soft px-3 py-2 text-xs text-danger">{run.last_error}</div>}
              <pre className="max-h-64 overflow-auto whitespace-pre-wrap px-3 py-2 font-mono text-[11px] leading-relaxed text-fg">
                {run.output ? run.output : <span className="text-fg-subtle">no console output</span>}
              </pre>
            </div>
          ))
        )}
      </div>
    </Modal>
  )
}

// The Schedule page's own create flow: pick the hosts here (grouped by
// team), a command, and a natural-language `when:` -- then the same
// useCreateScheduledTask dispatch the Hosts page uses. The server
// validates `when` for real (internal/schedule.Parse) and refuses an empty
// target, both surfaced here rather than silently failing.
function NewScheduledTaskModal({ buildId, onClose }: { buildId: string; onClose: () => void }) {
  const { data: build } = useBuild(buildId)
  const create = useCreateScheduledTask(buildId)
  const [selected, setSelected] = useState<Set<string>>(new Set())
  const [command, setCommand] = useState('reboot')
  const [values, setValues] = useState<Record<string, string>>({})
  const [when, setWhen] = useState('')
  const [whenValid, setWhenValid] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const teams = useMemo(
    () =>
      (build?.teams ?? []).map((t) => ({
        team_number: t.team_number,
        hosts: t.objects.filter((o) => o.kind !== 'network'),
      })),
    [build],
  )

  function toggle(id: string) {
    setSelected((prev) => {
      const next = new Set(prev)
      next.has(id) ? next.delete(id) : next.add(id)
      return next
    })
  }

  function toggleTeam(hosts: DeployedObject[]) {
    setSelected((prev) => {
      const allOn = hosts.length > 0 && hosts.every((h) => prev.has(h.id))
      const next = new Set(prev)
      for (const h of hosts) allOn ? next.delete(h.id) : next.add(h.id)
      return next
    })
  }

  const canSubmit = selected.size > 0 && whenValid && taskFormValid(command, values)

  async function submit() {
    setError(null)
    try {
      await create.mutateAsync({ target: { ids: Array.from(selected) }, when, command, payload: buildTaskPayload(command, values) })
      onClose()
    } catch (e) {
      setError(e instanceof ApiError ? e.message : 'Failed to schedule')
    }
  }

  return (
    <Modal
      open
      onClose={onClose}
      title="New Scheduled Task"
      footer={
        <>
          <Button variant="ghost" onClick={onClose}>
            Cancel
          </Button>
          <Button variant="brand" onClick={submit} disabled={create.isPending || !canSubmit}>
            {create.isPending ? 'Scheduling…' : 'Schedule'}
          </Button>
        </>
      }
    >
      <Label>Hosts ({selected.size} selected)</Label>
      <div className="mb-3 max-h-52 overflow-auto rounded-token border border-border">
        {teams.length === 0 && <div className="px-3 py-3 text-xs text-fg-muted">No hosts deployed yet.</div>}
        {teams.map((team) => {
          const allOn = team.hosts.length > 0 && team.hosts.every((h) => selected.has(h.id))
          return (
            <div key={team.team_number} className="border-b border-border last:border-b-0">
              <label className="flex items-center gap-2 bg-surface-sunken/60 px-3 py-1.5 text-xs font-semibold text-fg">
                <input type="checkbox" checked={allOn} onChange={() => toggleTeam(team.hosts)} disabled={team.hosts.length === 0} />
                Team {team.team_number}
                <span className="font-normal text-fg-subtle">({team.hosts.length})</span>
              </label>
              {team.hosts.map((h) => (
                <label key={h.id} className={cn('flex items-center gap-2 px-3 py-1.5 pl-6 text-sm hover:bg-surface-hover', selected.has(h.id) && 'bg-surface-accent')}>
                  <input type="checkbox" checked={selected.has(h.id)} onChange={() => toggle(h.id)} />
                  <span className="text-fg">{h.as_name ?? h.object_name}</span>
                  {h.network_name && <span className="text-xs text-fg-subtle">· {h.network_name}</span>}
                </label>
              ))}
            </div>
          )
        })}
      </div>

      <TaskActionFields
        command={command}
        values={values}
        onCommandChange={(c) => {
          setCommand(c)
          setValues({})
        }}
        onValueChange={(name, v) => setValues((prev) => ({ ...prev, [name]: v }))}
      />

      <WhenField buildId={buildId} when={when} onWhenChange={setWhen} onValidityChange={setWhenValid} />

      {error && <div className="mt-3 rounded-token border border-danger/30 bg-danger-soft p-2 text-xs text-danger">{error}</div>}
    </Modal>
  )
}
