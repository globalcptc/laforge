import { AlertTriangle, Boxes, Cloud, Server } from 'lucide-react'
import { Field, Input, cn } from '../../ui'
import type { Kind } from './model'

const KINDS: { kind: Kind; title: string; body: string; icon: typeof Server; draft?: boolean }[] = [
  {
    kind: 'microcloud',
    title: 'MicroCloud cluster',
    body: 'A MicroCloud: an LXD cluster with Ceph storage and OVN networking. Connect to any member; it answers for the whole cluster.',
    icon: Boxes,
  },
  {
    kind: 'incus',
    title: 'Incus hosts',
    body: 'One or more standalone Incus servers. Each team is placed on one host, spread evenly across them.',
    icon: Server,
  },
  {
    kind: 'aws',
    title: 'AWS (EC2)',
    body: 'Amazon EC2: a per-team VPC/subnet, instances from your AMIs. Credentials and region come from the runner’s AWS environment (AWS_* / AWS_REGION).',
    icon: Cloud,
    draft: true,
  },
  {
    kind: 'openstack',
    title: 'OpenStack',
    body: 'An OpenStack cloud (Nova/Neutron): instances from Glance images and Nova flavors. Credentials and region come from the runner’s OS_* environment.',
    icon: Cloud,
    draft: true,
  },
  // The 'fake' (simulated) builder stays wired up in the backend for local
  // development and tests, but is deliberately not offered here -- it isn't
  // something an operator sets up for a real event.
]

// The cloud builders are real code but have never been exercised against a live
// account, and don't yet wire external reachability (VPC egress / floating IPs)
// or terminate established connections on close -- so they're marked Draft here
// rather than presented as production-ready.
const DRAFT_KINDS = new Set<Kind>(KINDS.filter((k) => k.draft).map((k) => k.kind))

export const NAME_PATTERN = /^[a-z0-9][a-z0-9-]*$/

export function StepType({
  name,
  kind,
  editing,
  onName,
  onKind,
}: {
  name: string
  kind: Kind
  editing: boolean
  onName: (v: string) => void
  onKind: (k: Kind) => void
}) {
  const nameError = name && !NAME_PATTERN.test(name) ? 'Use lowercase letters, numbers, and dashes.' : undefined
  return (
    <div className="flex flex-col gap-5">
      <Field
        label="Name"
        hint={editing ? 'Configured builds refer to a builder by name, so it can’t change.' : 'Configured builds pick a builder by this name.'}
        error={nameError}
        className="mb-0 max-w-sm"
      >
        <Input value={name} onChange={(e) => onName(e.target.value.toLowerCase())} disabled={editing} placeholder="cptc-microcloud" autoFocus={!editing} />
      </Field>

      <div>
        <div className="mb-2 text-sm font-medium text-fg">What are you connecting to?</div>
        <div className="grid gap-3 sm:grid-cols-3">
          {KINDS.map(({ kind: k, title, body, icon: Icon, draft }) => (
            <button
              key={k}
              type="button"
              disabled={editing && k !== kind}
              onClick={() => onKind(k)}
              className={cn(
                'card flex flex-col gap-2 p-4 text-left transition-colors disabled:cursor-not-allowed disabled:opacity-40',
                kind === k ? 'border-accent ring-2 ring-accent-soft' : 'hover:border-border-strong',
              )}
            >
              <div className="flex items-center justify-between">
                <Icon size={18} className={kind === k ? 'text-accent' : 'text-fg-muted'} />
                {draft && (
                  <span className="rounded-token-sm bg-warning-soft px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-warning">Draft</span>
                )}
              </div>
              <div className="text-sm font-semibold text-fg">{title}</div>
              <div className="text-xs text-fg-muted">{body}</div>
            </button>
          ))}
        </div>
        {DRAFT_KINDS.has(kind) && (
          <div className="mt-2 flex items-start gap-2 rounded-token border border-warning/30 bg-warning-soft px-3 py-2 text-xs text-warning">
            <AlertTriangle size={14} className="mt-0.5 shrink-0" />
            <span>
              This builder is a draft — real code, but never run against a live account. It doesn’t yet wire external
              reachability or terminate established connections when access closes. Don’t rely on it for a competition
              without testing it first.
            </span>
          </div>
        )}
        {editing && <div className="mt-2 text-xs text-fg-subtle">The type of an existing builder can’t change -- create a new one instead.</div>}
      </div>
    </div>
  )
}
