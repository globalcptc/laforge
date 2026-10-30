import {
  CircleDashed,
  CircleHelp,
  Hourglass,
  Loader,
  LoaderPinwheel,
  OctagonPause,
  OctagonX,
  SquarePause,
  Check,
  CircleCheck,
  CircleX,
  MinusCircle,
  Power,
  PowerOff,
  Trash2,
  ClockAlert,
  PlugZap,
  SatelliteDish,
  Cog,
  ServerOff,
  TrashOff,
  Wrench,
  ShieldOff,
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
  // a derived state (not stored): a 'building' object with unmet blocked_on:
  | 'dependency_blocked'
  // teardown + access + agent + scheduled-task states:
  | 'tearing_down'
  | 'torn_down'
  | 'purged'
  | 'discarded'
  | 'destroy_queued'
  | 'destroying'
  | 'destroy_failed'
  | 'destroyed'
  | 'open'
  | 'closed'
  | 'booting'
  | 'healthy'
  | 'late'
  | 'missing'
  // Composite "effective" states (not stored): a host that is up by its build
  // state but whose agent or infra is unhealthy. effectiveStatusKey folds these
  // in so the matrix/team bars show a problem instead of a reassuring green.
  | 'agent_late'
  | 'agent_missing'
  | 'powered_off'
  | 'instance_gone'
  | 'fired'
  | 'canceled'
  // agent_task run states (the Schedule page's per-team task runs + run modal):
  // a task the agent has leased, finished, skipped, or is holding on a gate.
  | 'leased'
  | 'done'
  | 'ignored'
  | 'blocked'
  // legacy aliases (pre-00023), kept so any stale value still renders:
  | 'deployed'

type Anim = 'spin' | 'pulse' | 'slow-spin' | 'steady-spin'

const STYLES: Record<Kind, { icon: LucideIcon; anim?: Anim; tone: BadgeProps['tone']; label: string }> = {
  // Planned waits for the operator to click Deploy -- a gentle pulse, not a
  // spin, so it reads as "ready for you" rather than "the system is working."
  planned: { icon: CircleDashed, anim: 'pulse', tone: 'ink', label: 'Planned' },
  // Pending is queued for the system to act on -- a spinning dashed ring reads
  // as a loading spinner, so "queued, nothing at the hoster yet" still signals
  // activity rather than looking stuck.
  pending: { icon: SquarePause, anim: 'pulse', tone: 'ink', label: 'Pending' },
  // Infra being created -- dark blue, spinning.
  deploying: { icon: LoaderPinwheel, anim: 'spin', tone: 'indigo', label: 'Creating Hosts' },
  // Infra up, agent not yet checked in -- light blue, animated. The satellite
  // dish reads as "listening for the agent's first check-in."
  running: { icon: SatelliteDish, anim: 'pulse', tone: 'sky', label: 'Awaiting Agent' },
  // Agent checked in, steps/validators executing -- purple, spinning (work in
  // progress reads better as a spin than a pulse).
  building: { icon: Cog, anim: 'spin', tone: 'purple', label: 'Running Steps' },
  // Steps are queued but held until this object's dependencies finish -- a derived
  // state (building + unmet blocked_on), amber, rotating slowly (patient "held,
  // waiting," distinct from the active spinners). Reused by StatusBadge's blocked
  // branch and by the environment board, so its look lives in one place.
  dependency_blocked: { icon: Hourglass, anim: 'slow-spin', tone: 'warning', label: 'Dependency Blocked' },
  // The only green, and a terminal outcome: holds still. A plain check (no ring)
  // marks the single "done" state.
  finished: { icon: Check, tone: 'success', label: 'Build Complete' },
  // Three distinct failure hues, one per cause -- all terminal, all still.
  deploy_failed: { icon: ServerOff, tone: 'danger', label: 'Creation Failed' },
  build_failed: { icon: Wrench, tone: 'orange', label: 'Build Failed' },
  invalid: { icon: ShieldOff, tone: 'warning', label: 'Validation Failed' },
  // Build-level aggregate failure -- terminal, still.
  failed: { icon: CircleX, tone: 'danger', label: 'Failed' },
  tearing_down: { icon: OctagonX, anim: 'steady-spin', tone: 'pink', label: 'Tearing down' },
  // Destroyed is a terminal resting state -- no animation.
  torn_down: { icon: PowerOff, tone: 'neutral', label: 'Destroyed' },
  purged: { icon: Trash2, tone: 'neutral', label: 'Purged' },
  // A planned build that was superseded by a newer commit or whose configured
  // build was removed -- never deployed, nothing was ever created. Terminal,
  // resting, grey like the other history states.
  discarded: { icon: MinusCircle, tone: 'neutral', label: 'Discarded' },
  // A destroy is queued (a rebuild, a content change) but hasn't started: the
  // instance is still there. A pale magenta, set apart from Destroying's pink,
  // and pulsing like Pending -- waiting, not working.
  destroy_queued: { icon: OctagonPause, anim: 'pulse', tone: 'magenta', label: 'Destroy Pending' },
  destroying: { icon: OctagonX, anim: 'steady-spin', tone: 'pink', label: 'Destroying' },
  // The destroy-side twin of Creation Failed: the hoster refused to remove the
  // instance for a reason retrying cannot fix, so teardown stopped here. Terminal
  // and still until an operator retries the teardown or forces the build down. A
  // distinct icon (TrashOff, not deploy_failed's ServerOff) so the two don't
  // read identically on the environment board, where color + icon carry it alone.
  destroy_failed: { icon: TrashOff, tone: 'danger', label: 'Destroy Failed' },
  destroyed: { icon: MinusCircle, tone: 'neutral', label: 'Destroyed' },
  open: { icon: CircleCheck, tone: 'success', label: 'Open' },
  closed: { icon: MinusCircle, tone: 'warning', label: 'Closed' },
  booting: { icon: Power, anim: 'pulse', tone: 'sky', label: 'Booting' },
  // Resting-good agent state: still. Late/missing need attention: animated.
  healthy: { icon: CircleCheck, tone: 'success', label: 'Agent Healthy' },
  late: { icon: ClockAlert, anim: 'pulse', tone: 'warning', label: 'Agent Late' },
  missing: { icon: PlugZap, anim: 'pulse', tone: 'danger', label: 'Agent Missing' },
  // Composite effective states: an up host whose agent or infra is unhealthy.
  agent_late: { icon: ClockAlert, anim: 'pulse', tone: 'warning', label: 'Agent Late' },
  agent_missing: { icon: PlugZap, anim: 'pulse', tone: 'danger', label: 'Agent Missing' },
  powered_off: { icon: PowerOff, tone: 'orange', label: 'Powered Off' },
  instance_gone: { icon: ServerOff, tone: 'danger', label: 'Instance Missing' },
  fired: { icon: CircleCheck, tone: 'success', label: 'Fired' },
  canceled: { icon: MinusCircle, tone: 'neutral', label: 'Canceled' },
  // agent_task run states. 'leased' is in flight (the agent has it and is
  // running it) -- sky, spinning, like the other active work. 'done' is a
  // successful run (CircleCheck/success, matching 'fired'). 'ignored' was
  // skipped (neutral, still). 'blocked' is held on a gate (wait_for / a
  // dependency) -- the same patient amber Hourglass as dependency_blocked.
  leased: { icon: Loader, anim: 'spin', tone: 'sky', label: 'Running' },
  done: { icon: CircleCheck, tone: 'success', label: 'Done' },
  ignored: { icon: MinusCircle, tone: 'neutral', label: 'Skipped' },
  blocked: { icon: Hourglass, anim: 'slow-spin', tone: 'warning', label: 'Blocked' },
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
  dependency_blocked: 'Steps are queued but held until this object’s dependencies finish.',
  finished: 'Up, every step done, every validator passed.',
  deploy_failed: 'The hoster could not create the instance.',
  build_failed: 'A build step could not be completed.',
  invalid: 'Steps ran, but a validator did not pass.',
  failed: 'Settled with at least one failed object.',
  tearing_down: 'Teardown in progress.',
  torn_down: 'Removed from the hoster.',
  purged: 'Records cleared after teardown.',
  discarded: 'Planned build set aside — a newer commit superseded it, or its configured build was removed. Nothing was ever created.',
  destroy_queued: 'A destroy is queued (a rebuild or a content change) and hasn’t started; the instance is still up. Once it’s destroyed the host is recreated.',
  destroying: 'The hoster is removing the instance — for a teardown, or before it’s recreated.',
  destroy_failed: 'The hoster could not remove the instance; teardown is stuck until retried.',
  destroyed: 'Removed from the hoster.',
  open: 'External ingress is allowed.',
  closed: 'Ingress is blocked; connections dropped.',
  booting: 'Instance is up; no agent heartbeat yet.',
  healthy: 'Agent checked in within the window.',
  late: 'Agent is overdue for a check-in.',
  missing: 'Agent is long overdue — the host is probably gone.',
  fired: 'The scheduled task fired.',
  canceled: 'The scheduled task was canceled.',
  leased: 'The agent has picked up this task and is running it.',
  done: 'The task ran and completed.',
  ignored: 'The task was skipped on this host.',
  blocked: 'Held until a gate (a dependency or wait_for) is satisfied.',
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
  blockedOn,
}: {
  status: string
  kind?: string
  className?: string
  // A hover tooltip explaining the state fades in after ~1s. Off where the
  // explanation is already on screen (the legend).
  tooltip?: boolean
  // When set and non-empty on a 'building' object, the badge reads "Dependency
  // Blocked" instead of "Running Steps": its next steps are queued but held
  // until these dependencies (depends_on, or a step's wait_for) are ready.
  blockedOn?: string[] | null
}) {
  const isBlocked = status === 'building' && !!blockedOn && blockedOn.length > 0
  const key = isBlocked ? 'dependency_blocked' : status
  const style = STYLES[key as Kind] ?? { icon: CircleHelp, tone: 'neutral' as const, label: status }
  const label = (kind && KIND_LABELS[kind]?.[status as Kind]) ?? style.label
  const desc = isBlocked
    ? `Waiting on ${blockedOn!.join(', ')} before this host’s next steps run.`
    : statusDescription(status, kind)
  const Icon = style.icon
  const animClass = animClassFor(style.anim)
  const badge = (
    <Badge tone={style.tone} className={cn('shrink-0 whitespace-nowrap rounded-full py-0.5', className)}>
      <Icon size={12} className={animClass} />
      {label}
    </Badge>
  )
  if (!tooltip || !desc) return badge
  return (
    <span className="group/sb relative inline-flex shrink-0">
      {badge}
      {/* Not rendered until hover (display, not opacity): a hidden-but-laid-out
          tooltip still counts as overflow, and near the right edge on a phone
          that made the whole page scroll sideways. lf-tooltip-in keeps the
          1s dwell before it fades in. */}
      <span
        role="tooltip"
        className="pointer-events-none absolute bottom-full left-1/2 z-50 mb-1.5 hidden w-max max-w-[15rem] -translate-x-1/2 animate-[lf-tooltip-in_150ms_ease-out_1s_both] whitespace-normal rounded-token border border-border bg-surface-raised px-2 py-1 text-left text-xs font-normal text-fg-muted shadow-overlay group-hover/sb:block"
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

// animClassFor maps a status's animation to its Tailwind class. 'slow-spin' is a
// ~7s rotation (vs animate-spin's 1s) for the patient, held "Dependency Blocked"
// state -- deliberately much slower than the active spinners. 'steady-spin' is
// a 3s rotation, for work that's under way but slow (Destroying, Tearing down).
function animClassFor(anim?: Anim): string | undefined {
  return anim === 'spin'
    ? 'animate-spin'
    : anim === 'pulse'
      ? 'animate-pulse'
      : anim === 'slow-spin'
        ? 'animate-[spin_7s_linear_infinite]'
        : anim === 'steady-spin'
          ? 'animate-[spin_3s_linear_infinite]'
          : undefined
}

// blockedStatusKey resolves the derived 'dependency_blocked' state the same way
// StatusBadge does, so the environment board and the badge never diverge: a
// 'building' object with unmet blocked_on reads as dependency-blocked.
export function blockedStatusKey(status: string, blockedOn?: string[] | null): string {
  return status === 'building' && !!blockedOn && blockedOn.length > 0 ? 'dependency_blocked' : status
}

// Lifecycle states where the instance is expected to be up at the hoster with a
// live agent -- the only states where a missing/stopped instance or a dark agent
// is a problem (a pending/deploying/destroying/torn-down object has no agent to
// miss). Mirrors the backend's effective-health gate in dashboard.go.
const UP_STATES = new Set(['running', 'building', 'finished'])

// effectiveStatusKey folds an object's build state together with its infra
// (power) and agent health into ONE status key, worst-of: a host that is
// "finished" by its build but whose agent has gone missing or whose instance is
// powered off reads as the problem, not as green. Lifecycle failures and
// gone/terminal states dominate (nothing to fold); otherwise, once the instance
// should be up, an absent/stopped instance beats a dark agent beats the base
// state. Used by the Matrix, the team bars, and the hosts State filter so all
// three agree.
export function effectiveStatusKey(obj: {
  status: string
  blocked_on?: string[] | null
  power_state?: string
  agent?: { agent_status?: string }
}): string {
  const base = blockedStatusKey(obj.status, obj.blocked_on)
  if (GONE_STATES.has(obj.status) || base === 'deploy_failed' || base === 'build_failed' || base === 'destroy_failed' || base === 'invalid') {
    return base
  }
  if (UP_STATES.has(obj.status)) {
    if (obj.power_state === 'missing') return 'instance_gone'
    if (obj.power_state === 'stopped') return 'powered_off'
    if (obj.agent?.agent_status === 'missing') return 'agent_missing'
    if (obj.agent?.agent_status === 'late') return 'agent_late'
  }
  return base
}

// statusIcon / statusAnimClass expose the badge's own icon and animation for a
// status, for surfaces that render a bare glyph instead of a full badge (the
// environment board's tiles and legend). One source of truth -- the STYLES map.
export function statusIcon(status: string): LucideIcon {
  return STYLES[status as Kind]?.icon ?? CircleHelp
}
export function statusAnimClass(status: string): string | undefined {
  return animClassFor(STYLES[status as Kind]?.anim)
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
  'destroy_queued',
  'destroying',
  'destroy_failed',
  'torn_down',
] as const
export const AGENT_STATES = ['healthy', 'late', 'missing'] as const

// The composite agent/infra problem states effectiveStatusKey can derive, in
// severity order -- for the hosts State filter and the team-bar/legend ordering.
export const EFFECTIVE_HEALTH_STATES = ['agent_late', 'agent_missing', 'powered_off', 'instance_gone'] as const

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

// powerLabel is the display name for a power state ("Running", "Stopped"…).
export function powerLabel(state: string): string {
  return POWER_STYLES[state]?.label ?? state
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
