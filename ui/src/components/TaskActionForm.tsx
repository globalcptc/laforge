import { Label, Select } from '../ui'

// Every LaForge action an operator can run ad-hoc or on a schedule -- the
// exact set the Rust agent implements (agent/src/commands.rs) and the API
// allows (internal/api/tasks.go's allowedAdHocCommands). Each action is a
// small form spec: which fields it takes, their types, and which are
// required. buildTaskPayload turns the collected values into the JSON
// payload shape agentproto expects (field name === payload key), and
// taskFormValid gates submission. One source of truth, shared by the Run
// Task dialog, the Hosts "schedule for later" dialog, and the Schedule
// page's own create form.

type FieldType = 'text' | 'password' | 'number' | 'textarea' | 'select' | 'list'

interface FieldSpec {
  name: string
  label: string
  type: FieldType
  required?: boolean
  placeholder?: string
  options?: string[]
  help?: string
  mono?: boolean
  sep?: ',' | ' ' // list separator; default whitespace
}

export interface TaskActionSpec {
  command: string
  label: string
  fields: FieldSpec[]
}

export const TASK_ACTIONS: TaskActionSpec[] = [
  {
    command: 'execute',
    label: 'Run a command',
    fields: [
      { name: 'command', label: 'Program', type: 'text', required: true, mono: true, placeholder: '/bin/sh', help: 'The program to run. For a shell command use /bin/sh with args: -c "…".' },
      { name: 'args', label: 'Arguments', type: 'list', placeholder: '-c "systemctl restart nginx"', help: 'Optional, space-separated.' },
      { name: 'working_dir', label: 'Working directory', type: 'text', mono: true, placeholder: '/tmp' },
      { name: 'timeout_sec', label: 'Timeout (seconds)', type: 'number', placeholder: '60' },
    ],
  },
  {
    command: 'service',
    label: 'Manage a service',
    fields: [
      { name: 'name', label: 'Service name', type: 'text', required: true, mono: true, placeholder: 'nginx' },
      { name: 'action', label: 'Action', type: 'select', required: true, options: ['start', 'stop', 'restart', 'enable', 'disable'] },
    ],
  },
  {
    command: 'write_file',
    label: 'Write a file',
    fields: [
      { name: 'path', label: 'Path', type: 'text', required: true, mono: true, placeholder: '/etc/motd' },
      { name: 'content', label: 'Content', type: 'textarea', required: true },
      { name: 'mode', label: 'Mode', type: 'text', mono: true, placeholder: '0644' },
    ],
  },
  {
    command: 'append_file',
    label: 'Append to a file',
    fields: [
      { name: 'path', label: 'Path', type: 'text', required: true, mono: true, placeholder: '/etc/hosts' },
      { name: 'content', label: 'Content', type: 'textarea', required: true },
    ],
  },
  { command: 'delete', label: 'Delete a path', fields: [{ name: 'path', label: 'Path', type: 'text', required: true, mono: true, placeholder: '/tmp/scratch' }] },
  {
    command: 'change_perms',
    label: 'Change permissions',
    fields: [
      { name: 'path', label: 'Path', type: 'text', required: true, mono: true, placeholder: '/opt/app/run.sh' },
      { name: 'mode', label: 'Mode', type: 'text', mono: true, placeholder: '0755' },
      { name: 'owner', label: 'Owner', type: 'text', placeholder: 'root' },
      { name: 'group', label: 'Group', type: 'text', placeholder: 'root' },
    ],
  },
  {
    command: 'create_user',
    label: 'Create a user',
    fields: [
      { name: 'username', label: 'Username', type: 'text', required: true, mono: true, placeholder: 'operator' },
      { name: 'password', label: 'Password', type: 'password' },
      { name: 'groups', label: 'Groups', type: 'list', sep: ',', placeholder: 'sudo, docker', help: 'Optional, comma-separated.' },
    ],
  },
  {
    command: 'set_password',
    label: 'Set a password',
    fields: [
      { name: 'username', label: 'Username', type: 'text', required: true, mono: true, placeholder: 'operator' },
      { name: 'password', label: 'Password', type: 'password', required: true },
    ],
  },
  {
    command: 'add_to_group',
    label: 'Add user to a group',
    fields: [
      { name: 'username', label: 'Username', type: 'text', required: true, mono: true, placeholder: 'operator' },
      { name: 'group', label: 'Group', type: 'text', required: true, mono: true, placeholder: 'sudo' },
    ],
  },
]

function specFor(command: string): TaskActionSpec | undefined {
  return TASK_ACTIONS.find((a) => a.command === command)
}

export function buildTaskPayload(command: string, values: Record<string, string>): unknown {
  const spec = specFor(command)
  if (!spec) return {}
  const payload: Record<string, unknown> = {}
  for (const f of spec.fields) {
    const raw = (values[f.name] ?? '').trim()
    if (raw === '') continue
    if (f.type === 'number') {
      const n = parseInt(raw, 10)
      if (!Number.isNaN(n)) payload[f.name] = n
    } else if (f.type === 'list') {
      payload[f.name] = raw
        .split(f.sep === ',' ? ',' : /\s+/)
        .map((s) => s.trim())
        .filter(Boolean)
    } else {
      payload[f.name] = raw
    }
  }
  return payload
}

export function taskFormValid(command: string, values: Record<string, string>): boolean {
  const spec = specFor(command)
  if (!spec) return false
  return spec.fields.every((f) => !f.required || (values[f.name] ?? '').trim() !== '')
}

const inputClass =
  'h-9 w-full rounded-token border border-border bg-surface-raised px-2.5 text-sm text-fg placeholder:text-fg-subtle outline-none focus-visible:border-accent focus-visible:outline-2 focus-visible:outline-offset-0 focus-visible:outline-accent-soft'

// Presentational: the action picker plus the selected action's fields.
// State lives in the caller so each dialog can wire submit/validation its
// own way. Changing the action clears the field values (they don't carry
// meaning across actions), which is the caller's responsibility via
// onCommandChange.
export function TaskActionFields({
  command,
  values,
  onCommandChange,
  onValueChange,
}: {
  command: string
  values: Record<string, string>
  onCommandChange: (command: string) => void
  onValueChange: (name: string, value: string) => void
}) {
  const spec = specFor(command)
  return (
    <div>
      <Label>Action</Label>
      <Select value={command} onChange={(e) => onCommandChange(e.target.value)} className="mb-3">
        {TASK_ACTIONS.map((a) => (
          <option key={a.command} value={a.command}>
            {a.label}
          </option>
        ))}
      </Select>

      {spec?.fields.map((f) => (
        <div key={f.name} className="mb-3">
          <Label>
            {f.label}
            {!f.required && <span className="ml-1 font-normal text-fg-subtle">(optional)</span>}
          </Label>
          {f.type === 'textarea' ? (
            <textarea
              value={values[f.name] ?? ''}
              onChange={(e) => onValueChange(f.name, e.target.value)}
              placeholder={f.placeholder}
              rows={4}
              className={inputClass.replace('h-9', 'min-h-[5rem] py-1.5') + ' font-mono'}
            />
          ) : f.type === 'select' ? (
            <Select value={values[f.name] ?? ''} onChange={(e) => onValueChange(f.name, e.target.value)}>
              <option value="" disabled>
                Choose…
              </option>
              {f.options?.map((o) => (
                <option key={o} value={o}>
                  {o}
                </option>
              ))}
            </Select>
          ) : (
            <input
              type={f.type === 'password' ? 'password' : f.type === 'number' ? 'number' : 'text'}
              value={values[f.name] ?? ''}
              onChange={(e) => onValueChange(f.name, e.target.value)}
              placeholder={f.placeholder}
              className={f.mono ? inputClass + ' font-mono' : inputClass}
            />
          )}
          {f.help && <div className="mt-1 text-xs text-fg-subtle">{f.help}</div>}
        </div>
      ))}
    </div>
  )
}
