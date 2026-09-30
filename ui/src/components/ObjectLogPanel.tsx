import { useState } from 'react'
import { Radio, ScrollText, FileCode, Maximize2, Minimize2, ListChecks, CircleCheck, CircleX, CircleDashed, Loader, ShieldCheck, ShieldX, ChevronDown, Info, ServerCog, Eye, EyeOff, Copy, Check } from 'lucide-react'
import { useObjectEvents, useObjectHeartbeats, useObjectRender, useObjectSteps, useObjectInfra } from '../api/hooks'
import { useTimeFormat } from '../lib/time'
import { ApiError } from '../api/client'
import type { RenderedStep, StepStatus, ObjectKind, DeployedObject } from '../api/types'
import { EmptyState } from './EmptyState'
import { Badge, Button, cn, InspectorPanel, Spinner } from '../ui'
import { StatusBadge, PowerBadge, GONE_STATES } from './StatusBadge'

type Tab = 'info' | 'events' | 'steps' | 'heartbeats' | 'render'

// "Per-object logs" -- one host/container's own
// slice of the journal (real data, GET
// /builds/{id}/objects/{objectId}/events) plus, as of the real
// append-only agent_heartbeat log (migrations/00007), its own real
// heartbeat history -- "live troubleshooting" for exactly this object,
// including the real remote address each check-in came from (real
// "rules checking" signal: an address change mid-competition is
// actionable, not just an oddity).
export function ObjectLogPanel({
  buildId,
  objectId,
  label,
  kind = 'host',
  object,
  teamNumber,
  onClose,
}: {
  buildId: string
  objectId: string
  label: string
  kind?: ObjectKind
  // The clicked object's own row, when the caller has it -- powers the Info
  // tab (identity + agent + power) and scopes the Findings tab to this
  // object. Optional so the panel still works from a caller that only knows
  // the id (it just shows less on Info).
  object?: DeployedObject
  teamNumber?: number
  onClose: () => void
}) {
  const [tab, setTab] = useState<Tab>('info')
  const [wide, setWide] = useState(false)
  const fmt = useTimeFormat()
  // A network is provisioned by the builder, not by an in-guest agent, so
  // it has a log (events) and findings but no steps/heartbeats/render --
  // show only what applies to what was clicked.
  const isNetwork = kind === 'network'
  const { data: events, isLoading: eventsLoading } = useObjectEvents(buildId, objectId)
  const { data: heartbeats, isLoading: heartbeatsLoading } = useObjectHeartbeats(buildId, objectId)
  const { data: buildSteps, isLoading: stepsLoading } = useObjectSteps(buildId, objectId, !isNetwork && tab === 'steps')
  const {
    data: renderedSteps,
    isLoading: renderLoading,
    error: renderError,
  } = useObjectRender(buildId, objectId, tab === 'render')

  return (
    <InspectorPanel
      open
      overlay
      onClose={onClose}
      title={label}
      subtitle={kind}
      width={wide ? 'min(80vw, 60rem)' : 'min(92vw, 28rem)'}
      headerAction={
        <Button
          variant="ghost"
          size="icon"
          onClick={() => setWide((v) => !v)}
          aria-label={wide ? 'Collapse panel' : 'Expand panel'}
          title={wide ? 'Collapse' : 'Expand'}
        >
          {wide ? <Minimize2 size={14} /> : <Maximize2 size={14} />}
        </Button>
      }
    >
      <div className="-mx-4 -mt-3 mb-3 flex flex-wrap gap-1 border-b border-border px-4 pb-2">
        <TabButton active={tab === 'info'} onClick={() => setTab('info')} icon={Info} label="Info" />
        <TabButton active={tab === 'events'} onClick={() => setTab('events')} icon={ScrollText} label="Events" />
        {!isNetwork && <TabButton active={tab === 'steps'} onClick={() => setTab('steps')} icon={ListChecks} label="Steps" />}
        {!isNetwork && <TabButton active={tab === 'heartbeats'} onClick={() => setTab('heartbeats')} icon={Radio} label="Heartbeats" />}
        {!isNetwork && <TabButton active={tab === 'render'} onClick={() => setTab('render')} icon={FileCode} label="Render" />}
      </div>

      {tab === 'info' && <InfoTab buildId={buildId} objectId={objectId} object={object} kind={kind} label={label} teamNumber={teamNumber} fmtTime={fmt.time} />}

      {tab === 'events' && (
        <>
          {eventsLoading && <LoadingRow />}
          {!eventsLoading && (!events || events.length === 0) && (
            <EmptyState
              icon={ScrollText}
              title="No events yet for this object"
              hint="Deploy, destroy, and access lifecycle events for this specific host/container will appear here."
            />
          )}
          {events && events.length > 0 && (
            <div className="flex flex-col gap-2 font-mono text-xs">
              {events.map((ev) => (
                <div key={ev.id} className="border-b border-border pb-2">
                  <div className="flex items-center gap-2">
                    <span className="text-fg-muted">{fmt.time(ev.created_at)}</span>
                    <Badge tone="accent">{ev.kind}</Badge>
                  </div>
                  <div className="mt-1 text-fg">{ev.message}</div>
                </div>
              ))}
            </div>
          )}
        </>
      )}
      {tab === 'heartbeats' && (
        <>
          {heartbeatsLoading && <LoadingRow />}
          {!heartbeatsLoading && (!heartbeats || heartbeats.length === 0) && (
            <EmptyState
              icon={Radio}
              title="No heartbeats yet for this object"
              hint="Real check-ins from this object's agent will appear here as it heartbeats."
            />
          )}
          {heartbeats && heartbeats.length > 0 && (
            <div className="flex flex-col gap-2 font-mono text-xs">
              {heartbeats.map((h, i) => {
                const addrChanged = i < heartbeats.length - 1 && h.remote_addr !== heartbeats[i + 1].remote_addr
                return (
                  <div key={h.id} className="border-b border-border pb-2">
                    <div className="flex gap-2">
                      <span className="text-fg-muted">{fmt.time(h.created_at)}</span>
                      <span className={addrChanged ? 'text-warning' : 'text-fg-muted'}>{h.remote_addr ?? '—'}</span>
                    </div>
                    {addrChanged && <div className="mt-0.5 text-warning">remote address changed from the previous check-in</div>}
                  </div>
                )
              })}
            </div>
          )}
        </>
      )}
      {tab === 'steps' && (
        <>
          {stepsLoading && <LoadingRow />}
          {!stepsLoading && (!buildSteps || buildSteps.length === 0) && (
            <EmptyState
              icon={ListChecks}
              title="No build steps yet"
              hint="Steps appear here once the agent has checked in and they're queued to run."
            />
          )}
          {buildSteps && buildSteps.length > 0 && <StepStatusList steps={buildSteps} />}
        </>
      )}
      {tab === 'render' && (
        <>
          {renderLoading && <LoadingRow />}
          {renderError && <div className="text-sm text-danger">{renderError instanceof ApiError ? renderError.message : 'Failed to render'}</div>}
          {renderedSteps && renderedSteps.length === 0 && <EmptyState icon={FileCode} title="No steps on this object" hint="Nothing to render." />}
          {renderedSteps && renderedSteps.length > 0 && <RenderedSteps steps={renderedSteps} />}
        </>
      )}
    </InspectorPanel>
  )
}

// RenderedSteps lists a host's fully-resolved steps -- shared by the narrow
// drawer preview and the wide expanded modal, so both show identical
// content, just with different room.
function RenderedSteps({ steps }: { steps: RenderedStep[] }) {
  return (
    <div className="flex flex-col gap-3">
      {steps.map((s) => (
        <div key={s.index} className="overflow-hidden rounded-token border border-border">
          <div className="flex items-center gap-2 border-b border-border bg-surface-sunken px-2 py-1 text-xs">
            <span className="text-fg-muted">#{s.index}</span>
            <span className="font-medium text-fg">{s.action}</span>
            {s.script_name && <span className="text-accent">{s.script_name}</span>}
          </div>
          {s.rendered ? (
            <pre className="overflow-x-auto whitespace-pre-wrap p-2 font-mono text-xs text-fg">{s.rendered}</pre>
          ) : (
            <pre className="overflow-x-auto whitespace-pre-wrap p-2 font-mono text-xs text-fg-muted">{JSON.stringify(s.raw, null, 2)}</pre>
          )}
        </div>
      ))}
    </div>
  )
}

// StepStatusList shows where a host is in its build: each materialized step
// with its live status (pending → running → done / failed) and, when a step
// has validators, whether each passed. This is the execution truth (the
// real agent_task rows), distinct from the Render tab's rendered script.
const STEP_STYLE: Record<string, { icon: typeof CircleCheck; tone: 'neutral' | 'info' | 'success' | 'danger'; label: string; spin?: boolean }> = {
  pending: { icon: CircleDashed, tone: 'neutral', label: 'Pending' },
  leased: { icon: Loader, tone: 'info', label: 'Running', spin: true },
  done: { icon: CircleCheck, tone: 'success', label: 'Done' },
  failed: { icon: CircleX, tone: 'danger', label: 'Failed' },
}

function StepBadge({ status }: { status: string }) {
  const s = STEP_STYLE[status] ?? { icon: CircleDashed, tone: 'neutral' as const, label: status }
  const Icon = s.icon
  return (
    <Badge tone={s.tone} className="rounded-full py-0.5">
      <Icon size={11} className={s.spin ? 'animate-spin' : undefined} /> {s.label}
    </Badge>
  )
}

function StepStatusList({ steps }: { steps: StepStatus[] }) {
  const [open, setOpen] = useState<Set<number>>(new Set())
  function toggle(i: number) {
    setOpen((prev) => {
      const next = new Set(prev)
      next.has(i) ? next.delete(i) : next.add(i)
      return next
    })
  }
  // Group consecutive steps that came from the same authored step, so the
  // expanded sub-commands (write file, execute, validate) read as one step
  // under a named heading rather than a flat wall of rows.
  const groups: { key: number; label?: string; steps: StepStatus[] }[] = []
  for (const s of steps) {
    const last = groups[groups.length - 1]
    if (last && last.key === s.group_index) last.steps.push(s)
    else groups.push({ key: s.group_index, label: s.group_label, steps: [s] })
  }
  return (
    <div className="flex flex-col gap-3">
      {groups.map((g) => (
        <div key={g.key} className="flex flex-col gap-1.5">
          {g.label && (
            <div className="flex items-center gap-2 px-0.5">
              <span className="text-[11px] font-semibold uppercase tracking-wide text-fg-subtle">{g.label}</span>
              <span className="h-px flex-1 bg-border" />
            </div>
          )}
          {g.steps.map((s) => (
            <StepRow key={s.step_index} step={s} expanded={open.has(s.step_index)} onToggle={() => toggle(s.step_index)} />
          ))}
        </div>
      ))}
    </div>
  )
}

function StepRow({ step: s, expanded, onToggle }: { step: StepStatus; expanded: boolean; onToggle: () => void }) {
  const hasDetail = !!s.output || !!s.last_error || s.validators.length > 0 || s.attempts > 1
  return (
    <div className="overflow-hidden rounded-token border border-border">
      <button
        onClick={() => hasDetail && onToggle()}
        disabled={!hasDetail}
        className={cn('flex w-full items-center gap-2 bg-surface-sunken px-2 py-1.5 text-left text-xs', hasDetail && 'hover:bg-surface-hover')}
      >
        {hasDetail ? (
          <ChevronDown size={12} className={cn('shrink-0 text-fg-subtle transition-transform', !expanded && '-rotate-90')} />
        ) : (
          <span className="w-3 shrink-0" />
        )}
        <span className="text-fg-muted">#{s.step_index}</span>
        <span className="truncate font-mono font-medium text-fg">{s.command}</span>
        <span className="ml-auto shrink-0">
          <StepBadge status={s.status} />
        </span>
      </button>
      {expanded && hasDetail && (
        <div className="flex flex-col gap-2 border-t border-border px-2 py-2 text-xs">
          {s.attempts > 1 && <div className="text-fg-subtle">{s.attempts} attempts</div>}
          {s.output && (
            <div>
              <div className="mb-1 text-[11px] font-semibold uppercase tracking-wide text-fg-subtle">Output</div>
              <pre className="max-h-72 overflow-auto whitespace-pre-wrap rounded-token bg-surface-sunken p-2 font-mono text-fg">{s.output}</pre>
            </div>
          )}
          {s.last_error && (
            <div>
              <div className="mb-1 text-[11px] font-semibold uppercase tracking-wide text-danger">Error</div>
              <pre className="max-h-48 overflow-auto whitespace-pre-wrap rounded-token border border-danger/30 bg-danger-soft p-2 font-mono text-danger">{s.last_error}</pre>
            </div>
          )}
          {s.validators.length > 0 && (
            <div className="flex flex-col gap-0.5">
              {s.validators.map((v, i) => (
                <div key={i} className={cn('flex items-start gap-1.5', v.passed ? 'text-success' : 'text-danger')}>
                  {v.passed ? <ShieldCheck size={12} className="mt-0.5 shrink-0" /> : <ShieldX size={12} className="mt-0.5 shrink-0" />}
                  <span className="font-medium">{v.kind}</span>
                  {v.message && <span className="text-fg-muted">— {v.message}</span>}
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  )
}

// InfoTab is the host-detail "Identity" + "Agent" summary.
// It renders from the DeployedObject row the caller already
// has -- lifecycle status, live power, agent check-in, network, and the
// hoster's own external ref. Address/image/size are content-resolved and not
// stored on the row, so they're deliberately omitted here rather than shown
// blank; the Render tab shows exactly what was delivered.
function InfoTab({
  buildId,
  objectId,
  object,
  kind,
  label,
  teamNumber,
  fmtTime,
}: {
  buildId: string
  objectId: string
  object?: DeployedObject
  kind: ObjectKind
  label: string
  teamNumber?: number
  fmtTime: (iso: string) => string
}) {
  // Builder placement + the root password for hand access -- resolved
  // server-side from the builder-config chain (varies by builder).
  const { data: infra } = useObjectInfra(buildId, objectId, true)
  return (
    <div className="flex flex-col gap-4">
      <dl className="flex flex-col divide-y divide-border text-sm">
        <InfoRow label="Object" value={object?.object_name ?? label} mono />
        {object?.as_name && <InfoRow label="Name" value={object.as_name} mono />}
        {teamNumber != null && <InfoRow label="Team" value={String(teamNumber)} />}
        <InfoRow label="Kind" value={kind} />
        {object?.network_name && <InfoRow label="Network" value={object.network_name} mono />}
        {object && (
          <InfoRow label="State" valueNode={<StatusBadge status={object.status} kind={kind} />} />
        )}
        {object?.power_state && !GONE_STATES.has(object.status) ? <InfoRow label="Infra" valueNode={<PowerBadge state={object.power_state} />} /> : null}
        {object?.agent && (
          <InfoRow label="Agent" valueNode={<StatusBadge status={object.agent.agent_status} />} />
        )}
        {object?.agent?.last_heartbeat_at && (
          <InfoRow label="Last check-in" value={fmtTime(object.agent.last_heartbeat_at)} />
        )}
        {object?.external_ref && <InfoRow label="Hoster ref" value={object.external_ref} mono />}
        {object?.last_error && <InfoRow label="Last error" value={object.last_error} mono danger />}
      </dl>

      {infra && (infra.placement.length > 0 || infra.password) && (
        <div>
          <div className="mb-1.5 flex items-center gap-1.5 text-[11px] font-semibold uppercase tracking-wide text-fg-subtle">
            <ServerCog size={12} /> Builder &amp; access
          </div>
          <dl className="flex flex-col divide-y divide-border text-sm">
            {infra.placement.map((f) => (
              <InfoRow key={f.label} label={f.label} value={f.value} mono />
            ))}
            {infra.password && <PasswordRow password={infra.password} />}
          </dl>
        </div>
      )}
    </div>
  )
}

// PasswordRow shows the host's root/admin password for hand login, hidden by
// default with a reveal toggle and a one-click copy -- it's a real secret, so
// it isn't rendered in the clear until asked for.
function PasswordRow({ password }: { password: string }) {
  const [shown, setShown] = useState(false)
  const [copied, setCopied] = useState(false)
  async function copy() {
    try {
      await navigator.clipboard.writeText(password)
      setCopied(true)
      setTimeout(() => setCopied(false), 1500)
    } catch {
      /* clipboard blocked -- reveal is still available */
    }
  }
  return (
    <div className="flex items-start justify-between gap-3 py-2">
      <dt className="shrink-0 text-xs font-medium text-fg-muted">Root password</dt>
      <dd className="flex min-w-0 items-center gap-1.5">
        <span className="min-w-0 break-all text-right font-mono text-xs">{shown ? password : '••••••••••'}</span>
        <Button variant="ghost" size="icon" onClick={() => setShown((v) => !v)} aria-label={shown ? 'Hide password' : 'Show password'} title={shown ? 'Hide' : 'Show'}>
          {shown ? <EyeOff size={13} /> : <Eye size={13} />}
        </Button>
        <Button variant="ghost" size="icon" onClick={copy} aria-label="Copy password" title="Copy">
          {copied ? <Check size={13} className="text-success" /> : <Copy size={13} />}
        </Button>
      </dd>
    </div>
  )
}

function InfoRow({
  label,
  value,
  valueNode,
  mono,
  danger,
}: {
  label: string
  value?: string
  valueNode?: React.ReactNode
  mono?: boolean
  danger?: boolean
}) {
  return (
    <div className="flex items-start justify-between gap-3 py-2">
      <dt className="shrink-0 text-xs font-medium text-fg-muted">{label}</dt>
      <dd className={cn('min-w-0 break-words text-right', mono && 'font-mono text-xs', danger && 'text-danger')}>
        {valueNode ?? value}
      </dd>
    </div>
  )
}

function TabButton({ active, onClick, icon: Icon, label }: { active: boolean; onClick: () => void; icon: typeof ScrollText; label: string }) {
  return (
    <Button variant="ghost" size="sm" onClick={onClick} className={cn(active && 'bg-surface-hover text-fg')}>
      <Icon size={12} /> {label}
    </Button>
  )
}

function LoadingRow() {
  return (
    <div className="flex items-center gap-2 text-sm text-fg-muted">
      <Spinner /> Loading…
    </div>
  )
}
