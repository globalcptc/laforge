import { useParams } from '@tanstack/react-router'
import { useMemo, useState } from 'react'
import { CalendarClock, Plus, X } from 'lucide-react'
import { useBuild, useCancelScheduledTask, useCreateScheduledTask, useScheduledTasks } from '../api/hooks'
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
          <ScheduleTable title={`Upcoming (${pending.length})`} tasks={pending} onCancel={readOnly ? undefined : (id) => cancel.mutate(id)} cancelPending={cancel.isPending} />
          {past.length > 0 && <ScheduleTable title={`History (${past.length})`} tasks={past} />}
        </>
      )}
    </div>
  )
}

function ScheduleTable({
  title,
  tasks,
  onCancel,
  cancelPending,
}: {
  title: string
  tasks: ScheduledTask[]
  onCancel?: (id: string) => void
  cancelPending?: boolean
}) {
  const fmt = useTimeFormat()
  return (
    <div>
      <div className="mb-2 text-xs font-semibold uppercase tracking-wide text-fg-muted">{title}</div>
      <TableScroller contain={false}>
        <Table>
          <thead>
            <tr>
              <Th>When</Th>
              <Th>Command</Th>
              <Th>Source</Th>
              <Th>Next Fire</Th>
              <Th>Status</Th>
              {onCancel && <Th className="w-8"></Th>}
            </tr>
          </thead>
          <tbody>
            {tasks.map((t) => (
              <tr key={t.id}>
                <Td className="font-medium text-fg">{t.when_expr}</Td>
                <Td className="font-mono text-xs text-fg-muted">{t.command}</Td>
                <Td className="text-xs text-fg-muted">{t.source === 'content' ? 'Content' : t.created_by ? `Ad-hoc · ${t.created_by}` : 'Ad-hoc'}</Td>
                <Td className="text-xs text-fg-muted">{t.next_fire_at ? fmt.dateTime(t.next_fire_at) : '—'}</Td>
                <Td>
                  <StatusBadge status={t.status} />
                </Td>
                {onCancel && (
                  <Td>
                    <Button variant="ghost" size="icon" onClick={() => onCancel(t.id)} disabled={cancelPending} title="Cancel" className="text-fg-muted hover:text-danger">
                      <X size={14} />
                    </Button>
                  </Td>
                )}
              </tr>
            ))}
          </tbody>
        </Table>
      </TableScroller>
    </div>
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
