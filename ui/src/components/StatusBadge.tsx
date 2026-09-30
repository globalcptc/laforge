import {
  CircleDashed,
  CircleHelp,
  Loader,
  CircleCheck,
  CircleX,
  MinusCircle,
  Power,
  PowerOff,
  Trash2,
  ClockAlert,
  PlugZap,
  Radio,
  Cog,
  ServerCrash,
  Wrench,
  ShieldX,
  type LucideIcon,
} from 'lucide-react'
import { Badge, cn, Modal, type BadgeProps } from '../ui'

// The state vocabulary from the "Cell states" table --
// "colour + Lucide icon + label, one definition rendered identically
// everywhere." Covers deployed_object.status, build.status,
// team.access_state, and the agent check-in states (booting/healthy/late/
// missing).
//
// The host/container lifecycle (migration 00023) is the point of this map's
// colour discipline: green is reserved for exactly one state, `finished`
// ("Build Complete") -- an environment must not read as done until every step
// and validator has completed. The labels are operator-facing rather than
// literal DB values ("Creating Hosts", "Awaiting Agent", "Running Steps") so
// the stage reads plainly. Distinct hues per stage: pending=black (ink),
// deploying=dark blue (indigo), running=light blue (sky), building=purple,
// destroying=pink, torn_down=grey. The three failures stay three distinct
// hues, because each points at a different fix: deploy_failed ("Creation
// Failed", red) is infra, build_failed ("Build Failed", orange) is a failed
// step, invalid ("Validation Failed", amber) is a failed validator. Every
// badge animates while it is transitional or needs attention; the four
// terminal outcomes -- Build Complete, Creation Failed, Build Failed,
// Validation Failed -- and the other resting states (Agent Healthy, Open,
// Purged, ...) hold still. These same styles render for hosts, containers,
// networks, teams and environments, since they share one status vocabulary.
type Kind =
  | 'planned'
  | 'pending'
  | 'deploying'
  | 'running'
  | 'building'
  | 'finished'
  | 'deploy_failed'
  | 'build_failed'
  | 'invalid'
  // build.status also uses these two aggregates:
  | 'failed'
  // teardown + access + agent + scheduled-task states:
  | 'tearing_down'
  | 'torn_down'
  | 'purged'
  | 'destroying'
  | 'destroyed'
  | 'open'
  | 'closed'
  | 'booting'
  | 'healthy'
  | 'late'
  | 'missing'
  | 'fired'
  | 'canceled'
  // legacy aliases (pre-00023), kept so any stale value still renders:
  | 'deployed'

type Anim = 'spin' | 'pulse'

const STYLES: Record<Kind, { icon: LucideIcon; anim?: Anim; tone: BadgeProps['tone']; label: string }> = {
  // Planned waits for the operator to click Deploy -- a gentle pulse, not a
  // spin, so it reads as "ready for you" rather than "the system is working."
  planned: { icon: CircleDashed, anim: 'pulse', tone: 'ink', label: 'Planned' },
  // Pending is queued for the system to act on -- a spinning dashed ring reads
  // as a loading spinner, so "queued, nothing at the hoster yet" still signals
  // activity rather than looking stuck.
  pending: { icon: CircleDashed, anim: 'spin', tone: 'ink', label: 'Pending' },
  // Infra being created -- dark blue, spinning.
  deploying: { icon: Loader, anim: 'spin', tone: 'indigo', label: 'Creating Hosts' },
  // Infra up, agent not yet checked in -- light blue, animated.
  running: { icon: Radio, anim: 'pulse', tone: 'sky', label: 'Awaiting Agent' },
  // Agent checked in, steps/validators executing -- purple, spinning (work in
  // progress reads better as a spin than a pulse).
  building: { icon: Cog, anim: 'spin', tone: 'purple', label: 'Running Steps' },
  // The only green, and a terminal outcome: holds still.
  finished: { icon: CircleCheck, tone: 'success', label: 'Build Complete' },
  // Three distinct failure hues, one per cause -- all terminal, all still.
  deploy_failed: { icon: ServerCrash, tone: 'danger', label: 'Creation Failed' },
  build_failed: { icon: Wrench, tone: 'orange', label: 'Build Failed' },
  invalid: { icon: ShieldX, tone: 'warning', label: 'Validation Failed' },
  // Build-level aggregate failure -- terminal, still.
  failed: { icon: CircleX, tone: 'danger', label: 'Failed' },
  tearing_down: { icon: Loader, anim: 'spin', tone: 'pink', label: 'Tearing down' },
  // Destroyed is a terminal resting state -- no animation.
  torn_down: { icon: PowerOff, tone: 'neutral', label: 'Destroyed' },
  purged: { icon: Trash2, tone: 'neutral', label: 'Purged' },
  destroying: { icon: Loader, anim: 'spin', tone: 'pink', label: 'Destroying' },
  destroyed: { icon: MinusCircle, tone: 'neutral', label: 'Destroyed' },
  open: { icon: CircleCheck, tone: 'success', label: 'Open' },
  closed: { icon: MinusCircle, tone: 'warning', label: 'Closed' },
  booting: { icon: Power, anim: 'pulse', tone: 'sky', label: 'Booting' },
  // Resting-good agent state: still. Late/missing need attention: animated.
  healthy: { icon: CircleCheck, tone: 'success', label: 'Agent Healthy' },
  late: { icon: ClockAlert, anim: 'pulse', tone: 'warning', label: 'Agent Late' },
  missing: { icon: PlugZap, anim: 'pulse', tone: 'danger', label: 'Agent Missing' },
  fired: { icon: CircleCheck, tone: 'success', label: 'Fired' },
  canceled: { icon: MinusCircle, tone: 'neutral', label: 'Canceled' },
  deployed: { icon: CircleCheck, tone: 'success', label: 'Deployed' },
}

// Per-kind label overrides. The shared lifecycle labels are written for a
// host/container ("Creating Hosts", "Awaiting Agent", "Running Steps"), which
// read wrong on a network -- it has no agent to await and runs no steps.
// Only the differing words are listed here; the colour, icon and animation
// always come from STYLES, so a network's sky-blue "Provisioning" is the same
// state, in the same colour, as a host's "Awaiting Agent" -- just worded for
// what it is. A kind with no entry (host/container/environment) uses the
// shared label unchanged.
const KIND_LABELS: Partial<Record<string, Partial<Record<Kind, string>>>> = {
  network: {
    deploying: 'Creating Network',
    running: 'Provisioning',
    building: 'Provisioning',
    finished: 'Ready',
  },
}

// One-line explanation per state, shown in the legend and the hover tooltip.
const STATUS_DESC: Partial<Record<Kind, string>> = {
  planned: 'Resolved and inspectable — click Deploy to create it at the hoster.',
  pending: 'Queued for deploy; nothing created at the hoster yet.',
  deploying: 'The hoster is creating the instance.',
  running: 'Instance is up; waiting for the agent’s first check-in.',
  building: 'The agent is running this host’s steps and validators.',
  finished: 'Up, every step done, every validator passed.',
  deploy_failed: 'The hoster could not create the instance.',
  build_failed: 'A build step could not be completed.',
  invalid: 'Steps ran, but a validator did not pass.',
  failed: 'Settled with at least one failed object.',
  tearing_down: 'Teardown in progress.',
  torn_down: 'Removed from the hoster.',
  purged: 'Records cleared after teardown.',
  destroying: 'Being removed during teardown.',
  destroyed: 'Removed from the hoster.',
  open: 'External ingress is allowed.',
  closed: 'Ingress is blocked; connections dropped.',
  booting: 'Instance is up; no agent heartbeat yet.',
  healthy: 'Agent checked in within the window.',
  late: 'Agent is overdue for a check-in.',
  missing: 'Agent is long overdue — the host is probably gone.',
  fired: 'The scheduled task fired.',
  canceled: 'The scheduled task was canceled.',
  deployed: 'Deployed.',
}
// Kind-aware description overrides, mirroring KIND_LABELS so a network's
// tooltip matches its wording rather than the host-centric default.
const KIND_DESC: Partial<Record<string, Partial<Record<Kind, string>>>> = {
  network: {
    deploying: 'The network is being created at the hoster.',
    running: 'Created; finishing up (a network has no agent to wait on).',
    building: 'Created; finishing up (a network has no agent to wait on).',
    finished: 'Provisioned and ready.',
  },
}

// statusDescription is the plain-language meaning of a state, kind-aware where
// it matters -- the single source for both the legend and the hover tooltip.
export function statusDescription(status: string, kind?: string): string {
  return (kind && KIND_DESC[kind]?.[status as Kind]) ?? STATUS_DESC[status as Kind] ?? ''
}

export function StatusBadge({
  status,
  kind,
  className,
  tooltip = true,
}: {
  status: string
  kind?: string
  className?: string
  // A hover tooltip explaining the state fades in after ~1s. Off where the
  // explanation is already on screen (the legend).
  tooltip?: boolean
}) {
  const style = STYLES[status as Kind] ?? { icon: CircleHelp, tone: 'neutral' as const, label: status }
  const label = (kind && KIND_LABELS[kind]?.[status as Kind]) ?? style.label
  const desc = statusDescription(status, kind)
  const Icon = style.icon
  const animClass = style.anim === 'spin' ? 'animate-spin' : style.anim === 'pulse' ? 'animate-pulse' : undefined
  const badge = (
    <Badge tone={style.tone} className={cn('rounded-full py-0.5', className)}>
      <Icon size={12} className={animClass} />
      {label}
    </Badge>
  )
  if (!tooltip || !desc) return badge
  return (
    <span className="group/sb relative inline-flex">
      {badge}
      <span
        role="tooltip"
        className="pointer-events-none absolute bottom-full left-1/2 z-50 mb-1.5 w-max max-w-[15rem] -translate-x-1/2 translate-y-1 whitespace-normal rounded-token border border-border bg-surface-raised px-2 py-1 text-left text-xs font-normal text-fg-muted opacity-0 shadow-overlay transition delay-1000 duration-150 group-hover/sb:translate-y-0 group-hover/sb:opacity-100"
      >
        {desc}
      </span>
    </span>
  )
}

// TONE_FILL is the solid background class per badge tone, for the places that
// show a status as a solid swatch (dashboard bars, host-matrix cells) instead
// of a soft badge. Kept here so those track the badge colours from this one
// map rather than each keeping a copy that drifts out of sync.
const TONE_FILL: Record<string, string> = {
  neutral: 'bg-border-strong',
  ink: 'bg-ink',
  indigo: 'bg-indigo',
  sky: 'bg-sky',
  purple: 'bg-purple',
  success: 'bg-success',
  danger: 'bg-danger',
  orange: 'bg-orange',
  warning: 'bg-warning',
  pink: 'bg-pink',
  info: 'bg-info',
  accent: 'bg-accent',
  brand: 'bg-brand',
}

// statusLabel is the operator-facing label for a status ('building' ->
// 'Running Steps'), the exact text StatusBadge shows.
export function statusLabel(status: string): string {
  return STYLES[status as Kind]?.label ?? status
}

// statusFillClass is the solid background class for a status swatch, matching
// that status's badge tone.
export function statusFillClass(status: string): string {
  const tone = STYLES[status as Kind]?.tone
  return (tone && TONE_FILL[tone]) || 'bg-border-strong'
}

// The canonical host/container lifecycle order and the agent check-in states,
// for the legend. One source of truth -- StatusLegend renders real
// StatusBadges from these, so the legend always matches what a row shows.
export const LIFECYCLE_STATES = [
  'pending',
  'deploying',
  'running',
  'building',
  'finished',
  'deploy_failed',
  'build_failed',
  'invalid',
  'destroying',
  'torn_down',
] as const
export const AGENT_STATES = ['healthy', 'late', 'missing'] as const

// Lifecycle states where the instance no longer exists at the hoster, so its
// infra/power state is meaningless -- a destroyed object has no VM to be
// "running". Callers use this to drop a stale power badge next to a Destroyed
// lifecycle badge.
export const GONE_STATES = new Set(['destroyed', 'torn_down'])

// StatusLegend is the "place to review" what each colour/icon/label means --
// the same StatusBadges the rows use, laid out as a key. Shown wherever these
// colours appear (the Hosts screen, the matrix). Because it renders real
// badges, it can never fall out of step with them.
export function StatusLegend({ className, agent = true }: { className?: string; agent?: boolean }) {
  return (
    <div className={cn('flex flex-col gap-2 text-xs', className)}>
      <div className="flex flex-wrap items-center gap-1.5">
        <span className="mr-1 font-medium text-fg-muted">State</span>
        {LIFECYCLE_STATES.map((s) => (
          <StatusBadge key={s} status={s} />
        ))}
      </div>
      {agent && (
        <div className="flex flex-wrap items-center gap-1.5">
          <span className="mr-1 font-medium text-fg-muted">Agent</span>
          {AGENT_STATES.map((s) => (
            <StatusBadge key={s} status={s} />
          ))}
        </div>
      )}
    </div>
  )
}

// StatusLegendDialog is the reviewable key as a modal: every state with its
// badge and a one-line explanation. Tooltips are off on these badges since the
// explanation sits right beside each one.
function LegendRows({ title, states }: { title: string; states: readonly string[] }) {
  return (
    <div>
      <div className="mb-1 text-[11px] font-semibold uppercase tracking-wide text-fg-subtle">{title}</div>
      <dl className="flex flex-col divide-y divide-border">
        {states.map((s) => (
          <div key={s} className="flex items-center justify-between gap-4 py-1.5">
            <dt className="shrink-0">
              <StatusBadge status={s} tooltip={false} />
            </dt>
            <dd className="text-right text-xs text-fg-muted">{statusDescription(s)}</dd>
          </div>
        ))}
      </dl>
    </div>
  )
}

export function StatusLegendDialog({ open, onClose }: { open: boolean; onClose: () => void }) {
  return (
    <Modal
      open={open}
      onClose={onClose}
      title="State Legend"
      description="What each state means. These same colours and icons render across hosts, containers, networks, teams, and the environment."
    >
      <div className="flex flex-col gap-4">
        <LegendRows title="Lifecycle" states={LIFECYCLE_STATES} />
        <LegendRows title="Agent Check-In" states={AGENT_STATES} />
        <LegendRows title="Team Access" states={['open', 'closed']} />
      </div>
    </Modal>
  )
}

// PowerBadge is the instance's live power state at the hoster (builder
// truth), kept a separate control from StatusBadge so "the VM is up" reads
// distinctly from "the agent checked in" -- the two are independent, and
// conflating them hides the difference between an agent problem and an
// infra one. Its own map because 'missing' means something different here
// (the instance is gone) than the agent 'missing' above.
const POWER_STYLES: Record<string, { icon: LucideIcon; tone: BadgeProps['tone']; label: string }> = {
  running: { icon: Power, tone: 'success', label: 'Running' },
  stopped: { icon: PowerOff, tone: 'warning', label: 'Stopped' },
  other: { icon: CircleHelp, tone: 'neutral', label: 'Other' },
  missing: { icon: CircleX, tone: 'danger', label: 'Missing' },
}

export function PowerBadge({ state, className }: { state: string; className?: string }) {
  // Empty (not yet polled) renders nothing -- no state to claim yet.
  const style = POWER_STYLES[state]
  if (!style) return null
  const Icon = style.icon
  return (
    <Badge tone={style.tone} className={cn('rounded-full py-0.5', className)}>
      <Icon size={12} />
      {style.label}
    </Badge>
  )
}
