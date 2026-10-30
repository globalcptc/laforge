import { useNavigate, useParams, useSearch } from '@tanstack/react-router'
import { Fragment as FragmentGroup, useEffect, useMemo, useRef, useState } from 'react'
import { Box, CalendarClock, Check, ChevronDown, ShieldCheck, Container, Hammer, Info, LayoutGrid, Network, Play, RotateCcw, RotateCw, Search, Server, Square, Terminal, TriangleAlert, Users, Waypoints, type LucideIcon } from 'lucide-react'
import { useAdHocTask, useBuild, useCreateScheduledTask, usePowerAction, useRebuild, useTopology } from '../api/hooks'
import { StatusBadge, PowerBadge, powerLabel, StatusLegendDialog, GONE_STATES, statusLabel, statusFillClass, LIFECYCLE_STATES, effectiveStatusKey, EFFECTIVE_HEALTH_STATES } from '../components/StatusBadge'
import { EmptyState } from '../components/EmptyState'
import { ObjectLogPanel } from '../components/ObjectLogPanel'
import { RestartStepsDialog, canRestartSteps } from '../components/RestartStepsDialog'
import { RevalidateDialog, ValidationReportsDialog } from '../components/ValidationReport'
import { EnvironmentBoard } from '../components/EnvironmentBoard'
import { DependencyGraph } from '../components/DependencyGraph'
import { TaskActionFields, buildTaskPayload, taskFormValid } from '../components/TaskActionForm'
import { WhenField } from '../components/WhenField'
import { Boxes } from 'lucide-react'
import { ApiError } from '../api/client'
import type { DeployedObject, ObjectKind, RebuildAffected, TopologyMember } from '../api/types'
import { AnchoredPopover, Badge, Button, cn, Label, Modal, Spinner, Table, TableScroller, Td, Th, useToast } from '../ui'
import { buildIsReadOnly } from '../lib/build'

// Kind → icon + label, so a row states its type at a glance (feedback: "list
// out the type (network, team, host, container) as well as use icons to
// delineate the type for each"). One definition, used on host rows and on the
// team/network group headers.
const KIND_META: Record<string, { icon: LucideIcon; label: string }> = {
  network: { icon: Network, label: 'Network' },
  team: { icon: Users, label: 'Team' },
  host: { icon: Server, label: 'Host' },
  container: { icon: Container, label: 'Container' },
}

function KindBadge({ kind }: { kind: string }) {
  const m = KIND_META[kind] ?? { icon: Box, label: kind }
  const Icon = m.icon
  return (
    <Badge tone="neutral" className="rounded-full py-0.5">
      <Icon size={11} /> {m.label}
    </Badge>
  )
}

// Canonical order for the team-row status bars -- the same lifecycle order
// (and, via statusFillClass, the same colours) the Overview charts and host
// badges use, with any unexpected status appended so nothing is dropped.
const TEAM_BAR_ORDER: string[] = [...LIFECYCLE_STATES, ...EFFECTIVE_HEALTH_STATES, 'destroyed', 'planned']

// TeamNetworkBars renders one compact stacked status bar PER NETWORK on a team
// header row: each network's hosts/containers split by their standard status,
// in the same colours as the Overview "State by Team" bars. Deliberately tiny
// (h-2 bars, 10px labels) so it sits inline on the existing team row without
// making it any taller -- a glance-level "which network on which team is red."
function TeamNetworkBars({ networks, className }: { networks: { name: string; hosts: DeployedObject[] }[]; className?: string }) {
  const nets = networks.filter((n) => n.hosts.length > 0)
  if (nets.length === 0) return null
  return (
    <div className={cn('flex flex-nowrap items-center gap-3', className)}>
      {nets.map((net) => {
        const counts = new Map<string, number>()
        for (const h of net.hosts) {
          const k = effectiveStatusKey(h)
          counts.set(k, (counts.get(k) ?? 0) + 1)
        }
        const ordered = [...TEAM_BAR_ORDER.filter((s) => counts.has(s)), ...[...counts.keys()].filter((s) => !TEAM_BAR_ORDER.includes(s))]
        const total = net.hosts.length
        const legend = ordered.map((s) => `${statusLabel(s)}: ${counts.get(s)}`).join(' · ')
        return (
          <div key={net.name || '—'} className="flex shrink-0 items-center gap-1.5" title={`${net.name || 'no network'} — ${legend}`}>
            <span className="max-w-[6rem] truncate text-[10px] font-medium text-fg-subtle">{net.name || 'no network'}</span>
            <div className="flex h-2 w-16 shrink-0 overflow-hidden rounded-full bg-surface-sunken">
              {ordered.map((s) => (
                <div
                  key={s}
                  className={statusFillClass(s)}
                  style={{ width: `${((counts.get(s) ?? 0) / total) * 100}%` }}
                  title={`${statusLabel(s)}: ${counts.get(s)}`}
                />
              ))}
            </div>
            <span className="text-[10px] tabular-nums text-fg-subtle">{total}</span>
          </div>
        )
      })}
    </div>
  )
}

// Which power actions make sense for a given live power state -- offered as
// hover controls on the Infra pill. A running instance can be stopped or
// restarted; a stopped one started; an "other" (neither cleanly up nor down)
// gets both; a missing instance (gone at the hoster) gets none.
type PowerAct = 'start' | 'stop' | 'reboot'
const POWER_ACTIONS: Record<string, { action: PowerAct; icon: LucideIcon; label: string }[]> = {
  running: [
    { action: 'stop', icon: Square, label: 'Stop' },
    { action: 'reboot', icon: RotateCw, label: 'Restart' },
  ],
  stopped: [{ action: 'start', icon: Play, label: 'Start' }],
  other: [
    { action: 'start', icon: Play, label: 'Start' },
    { action: 'stop', icon: Square, label: 'Stop' },
  ],
}

// InfraCell shows the live power pill, and on hover (after a ~0.5s dwell, so it
// doesn't flicker while scanning the list) reveals same-size Start/Stop/Restart
// controls just past the pill. Each opens the confirm modal for this one host.
function InfraCell({ obj, onAction, onRebuild, onRevalidate, onRestartSteps }: { obj: DeployedObject; onAction: (action: PowerAct, obj: DeployedObject) => void; onRebuild?: (obj: DeployedObject) => void; onRevalidate?: (obj: DeployedObject) => void; onRestartSteps?: (obj: DeployedObject) => void }) {
  if (!obj.power_state || GONE_STATES.has(obj.status)) return <span className="text-fg-muted">—</span>
  const actions = POWER_ACTIONS[obj.power_state] ?? []
  const name = obj.as_name ?? obj.object_name
  const restartable = !!onRestartSteps && canRestartSteps(obj)
  if (actions.length === 0 && !onRebuild && !onRevalidate && !restartable) return <PowerBadge state={obj.power_state} />
  return (
    <div className="group/pw relative inline-flex items-center">
      <PowerBadge state={obj.power_state} />
      <div className="pointer-events-none absolute left-full top-1/2 z-20 ml-1 flex -translate-y-1/2 -translate-x-1 items-center gap-0.5 rounded-full border border-border bg-surface-raised px-1 py-0.5 opacity-0 shadow-raised transition delay-500 duration-150 group-hover/pw:pointer-events-auto group-hover/pw:translate-x-0 group-hover/pw:opacity-100">
        {actions.map((a) => {
          const Icon = a.icon
          return (
            <button
              key={a.action}
              onClick={() => onAction(a.action, obj)}
              title={`${a.label} ${name}`}
              aria-label={`${a.label} ${name}`}
              className="flex h-5 w-5 items-center justify-center rounded-full text-fg-muted transition-colors hover:bg-surface-hover hover:text-fg"
            >
              <Icon size={12} />
            </button>
          )
        })}
        {restartable && (
          <>
            {actions.length > 0 && <span className="mx-0.5 h-3.5 w-px bg-border" />}
            <button
              onClick={() => onRestartSteps!(obj)}
              title={`Restart ${name}'s build from a step, on the box as it is`}
              aria-label={`Restart ${name} from a step`}
              className="flex h-5 w-5 items-center justify-center rounded-full text-fg-muted transition-colors hover:bg-warning-soft hover:text-warning"
            >
              <RotateCcw size={12} />
            </button>
          </>
        )}
        {onRebuild && (
          <>
            {(actions.length > 0 || restartable) && <span className="mx-0.5 h-3.5 w-px bg-border" />}
            <button
              onClick={() => onRebuild(obj)}
              title={`Rebuild ${name} and everything that depends on it`}
              aria-label={`Rebuild ${name}`}
              className="flex h-5 w-5 items-center justify-center rounded-full text-fg-muted transition-colors hover:bg-danger-soft hover:text-danger"
            >
              <Hammer size={12} />
            </button>
          </>
        )}
        {onRevalidate && (
          <button
            onClick={() => onRevalidate(obj)}
            title={`Revalidate ${name} — re-run every validator and report, changing nothing else`}
            aria-label={`Revalidate ${name}`}
            className="flex h-5 w-5 items-center justify-center rounded-full text-fg-muted transition-colors hover:bg-accent-soft hover:text-accent-fg"
          >
            <ShieldCheck size={12} />
          </button>
        )}
      </div>
    </div>
  )
}

// toggleSet returns a new Set with v added or removed -- how the multi-select
// filters below flip a value on and off.
function toggleSet(s: Set<string>, v: string): Set<string> {
  const next = new Set(s)
  next.has(v) ? next.delete(v) : next.add(v)
  return next
}

// The Status filter's three columns, and a host's value in each, as
// "<column>:<value>" keys -- exactly what its State / Infra / Agent cells show
// ('' where the cell shows a dash).
type StatusColumn = 'state' | 'infra' | 'agent'
function statusKeys(h: DeployedObject): string[] {
  const infra = !h.power_state || GONE_STATES.has(h.status) ? '' : h.power_state
  return [`state:${h.status}`, `infra:${infra}`, `agent:${h.agent?.agent_status ?? ''}`]
}

// matchesStatusFilter: OR within a column, AND across the columns that have
// a pick.
function matchesStatusFilter(h: DeployedObject, picked: Set<string>): boolean {
  const keys = statusKeys(h)
  return (['state', 'infra', 'agent'] as StatusColumn[]).every((col) => {
    const inCol = [...picked].filter((k) => k.startsWith(col + ':'))
    return inCol.length === 0 || inCol.some((k) => keys.includes(k))
  })
}

// MultiFilter is one checkbox dropdown for the Hosts toolbar (Type / OS / Status /
// Tag). Each filter is multi-select: pick any number of values and a host matches
// when its value is among those picked (OR within a filter, AND across filters).
// An option's optional third element is a section heading (Status splits into
// State / Infra / Agent); a heading is drawn wherever it changes.
// The count badge and accent border show when a filter is narrowing the list.
function MultiFilter({ label, selected, onToggle, onClear, options }: { label: string; selected: Set<string>; onToggle: (v: string) => void; onClear: () => void; options: [string, string, string?][] }) {
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
  const count = selected.size
  return (
    <>
      <button
        ref={anchorRef}
        type="button"
        onClick={() => setOpen((o) => !o)}
        aria-label={label}
        className={cn(
          'flex h-8 items-center gap-1.5 rounded-token border bg-surface-raised px-2.5 text-sm outline-none focus-visible:border-accent focus-visible:outline-2 focus-visible:outline-offset-0 focus-visible:outline-accent-soft',
          count ? 'border-accent text-fg' : 'border-border text-fg-muted',
        )}
      >
        {label}
        {count > 0 && <span className="rounded-full bg-accent px-1.5 text-[11px] font-medium text-fg-inverted">{count}</span>}
        <ChevronDown size={13} className={cn('text-fg-subtle transition-transform', open && 'rotate-180')} />
      </button>
      {open && (
        <AnchoredPopover anchorRef={anchorRef} panelRef={panelRef} minWidth={180} maxHeight={300}>
          <div className="py-1">
            {count > 0 && (
              <button type="button" onClick={onClear} className="flex w-full items-center px-2.5 py-1.5 text-left text-xs text-fg-muted hover:bg-surface-hover">
                Clear {label.toLowerCase()}
              </button>
            )}
            {options.map(([v, l, group], i) => {
              const on = selected.has(v)
              const heading = group && group !== options[i - 1]?.[2] ? group : null
              return (
                <FragmentGroup key={v}>
                {heading && <div className="px-2.5 pb-0.5 pt-2 text-[11px] font-semibold uppercase tracking-wide text-fg-subtle">{heading}</div>}
                <button type="button" onClick={() => onToggle(v)} className="flex w-full items-center gap-2 px-2.5 py-1.5 text-left text-sm text-fg hover:bg-surface-hover">
                  <span className={cn('flex h-4 w-4 shrink-0 items-center justify-center rounded border', on ? 'border-accent bg-accent text-fg-inverted' : 'border-border')}>{on && <Check size={11} />}</span>
                  <span className="truncate">{l}</span>
                </button>
                </FragmentGroup>
              )
            })}
          </div>
        </AnchoredPopover>
      )}
    </>
  )
}

// "Build → Hosts": the deployed hosts/containers as a searchable, filterable
// table (the compact status matrix now lives at the bottom of the Overview).
// Real selection and real ad-hoc dispatch (internal/api/tasks.go) -- "the
// matched set is shown before anything runs, with a count" via the
// ImpactDialog below.
export function BuildHosts() {
  const { repoId, buildId } = useParams({ from: '/repos/$repoId/builds/$buildId/hosts' })
  const { inspect } = useSearch({ from: '/repos/$repoId/builds/$buildId/hosts' })
  const navigate = useNavigate()
  const { data: build, isLoading } = useBuild(buildId)
  // A torn-down build is history: show it, but offer no actions (no selection,
  // no ad-hoc/power tasks). See buildIsReadOnly.
  const readOnly = buildIsReadOnly(build?.status)
  // Topology's content facts (CIDR, OS/image, size, dependencies) are folded
  // onto the host lines here now that the standalone Topology view is gone --
  // keyed by object_id (== deployed_object id) and network object_id.
  const { data: topology } = useTopology(buildId)
  const [selected, setSelected] = useState<Set<string>>(new Set())
  const adHoc = useAdHocTask(buildId)
  const scheduleTask = useCreateScheduledTask(buildId)
  const [confirming, setConfirming] = useState<{ initialCommand: string; label: string } | null>(null)
  const [scheduling, setScheduling] = useState<{ initialCommand: string; label: string } | null>(null)
  const [powering, setPowering] = useState<'start' | 'stop' | 'reboot' | null>(null)
  // A single-instance power action from the Infra pill's hover controls,
  // confirmed in the same PowerDialog the bulk bar uses (one target here).
  const [powerSingle, setPowerSingle] = useState<{ action: 'start' | 'stop' | 'reboot'; id: string; name: string; team: number | null } | null>(null)
  // A forced rebuild (tear down + recreate) of one or more hosts plus their
  // dependents; confirmed in RebuildDialog, which previews the full blast radius.
  const [rebuilding, setRebuilding] = useState<{ ids: string[]; names: string[]; closedTeams: number[] } | null>(null)
  // Revalidate: re-run the hosts' validators only, and show the report.
  const [revalidating, setRevalidating] = useState<{ id: string; name: string }[] | null>(null)
  const [reportsOpen, setReportsOpen] = useState(false)
  // Restart from step: the host(s) it was started from and the authored step;
  // RestartStepsDialog widens to every team's copy and confirms.
  const [restarting, setRestarting] = useState<{ ids: string[]; fromGroup: number } | null>(null)
  const [logPanelFor, setLogPanelFor] = useState<{ id: string; label: string; kind: ObjectKind; object?: DeployedObject; teamNumber?: number; initialTab?: 'info' | 'steps' } | null>(null)
  const [boardOpen, setBoardOpen] = useState(false)
  const [graphOpen, setGraphOpen] = useState(false)
  const [filter, setFilter] = useState('')
  const [typeFilter, setTypeFilter] = useState<Set<string>>(new Set())
  const [osFilter, setOsFilter] = useState<Set<string>>(new Set())
  // The Status filter covers all three status columns. Values are
  // "<column>:<value>" (state / infra / agent); see statusKeys.
  const [statusFilter, setStatusFilter] = useState<Set<string>>(new Set())
  const [tagFilter, setTagFilter] = useState<Set<string>>(new Set())
  const [collapsedTeams, setCollapsedTeams] = useState<Set<number>>(new Set())
  const [collapsedNets, setCollapsedNets] = useState<Set<string>>(new Set())
  const [legendOpen, setLegendOpen] = useState(false)

  const objects: DeployedObject[] = useMemo(() => build?.teams.flatMap((t) => t.objects) ?? [], [build])

  // team_id -> team_number, so the power/rebuild confirmations can show which
  // team each host belongs to (hostnames repeat across teams -- every team gets
  // the same network -- so a bare name list is ambiguous about what's affected).
  const teamNumberById = useMemo(() => {
    const m = new Map<string, number>()
    for (const t of build?.teams ?? []) m.set(t.id, t.team_number)
    return m
  }, [build])

  // Deep-open the inspector on the Info tab when arrived at with ?inspect=<id>
  // (the standalone terminal window's "Host Info" button). Fires once the object
  // is known; the param is cleared when the panel closes so it won't reopen.
  useEffect(() => {
    if (!inspect) return
    if (logPanelFor?.id === inspect) return
    const obj = objects.find((o) => o.id === inspect)
    if (!obj) return
    setLogPanelFor({
      id: obj.id,
      label: obj.as_name ?? obj.object_name,
      kind: obj.kind,
      object: obj,
      teamNumber: teamNumberById.get(obj.team_id),
      initialTab: 'info',
    })
  }, [inspect, objects, teamNumberById, logPanelFor?.id])

  // team_id -> whether that team's access is currently closed. A closed team is
  // firewalled down to the gateway (no general egress), so a rebuilt host can't
  // fetch its agent/step downloads -- the rebuild will likely hang or fail. The
  // rebuild confirmation warns when any target is on a closed team.
  const closedTeamById = useMemo(() => {
    const m = new Map<string, boolean>()
    for (const t of build?.teams ?? []) m.set(t.id, t.access_state === 'closed')
    return m
  }, [build])
  const closedTeamsForHosts = (hs: DeployedObject[]): number[] => {
    const nums = new Set<number>()
    for (const h of hs) {
      if (closedTeamById.get(h.team_id)) nums.add(teamNumberById.get(h.team_id) ?? 0)
    }
    return [...nums].sort((a, b) => a - b)
  }

  // Router doesn't remount this component when navigating between two
  // builds that hit the same route (only buildId's param value changes),
  // so a selection made on build A's table would otherwise still be
  // "selected" in React state after switching to build B's. Every
  // selected id already belongs to build A specifically (deployed_object
  // ids are real, globally unique Postgres UUIDs -- see
  // internal/api/tasks.go's own hardening for the server-side half of
  // this), so it could never match one of build B's real rows and
  // silently dispatch to the wrong build; this is purely about not
  // showing a stale "N selected" bar with nothing visibly checked.
  useEffect(() => {
    setSelected(new Set())
  }, [buildId])

  const hosts = useMemo(() => (objects ?? []).filter((o) => o.kind !== 'network'), [objects])

  // Topology lookups: a member's content facts by its object_id, and a
  // network's CIDR by the network deployed_object's id.
  const memberByObjId = useMemo(() => {
    const m = new Map<string, TopologyMember>()
    for (const t of topology?.teams ?? []) for (const n of t.networks) for (const mem of n.members) m.set(mem.object_id, mem)
    return m
  }, [topology])
  const cidrByNetObjId = useMemo(() => {
    const m = new Map<string, string>()
    for (const t of topology?.teams ?? []) for (const n of t.networks) if (n.object_id && n.cidr) m.set(n.object_id, n.cidr)
    return m
  }, [topology])

  // Content facts for a host, from the folded-in topology.
  const hostOs = (h: DeployedObject) => {
    const mem = memberByObjId.get(h.id)
    return mem?.os ?? mem?.image ?? ''
  }
  // A host's content tags as "key=value" strings -- what the Tags filter and
  // search match on, and what a tag chip shows.
  const hostTagList = (h: DeployedObject) => Object.entries(memberByObjId.get(h.id)?.tags ?? {}).map(([k, v]) => `${k}=${v}`)
  // Distinct values present, for the Type/OS/State/Tag filter dropdowns.
  const osOptions = useMemo(() => [...new Set(hosts.map(hostOs).filter(Boolean))].sort(), [hosts, memberByObjId])
  // The Status filter has one section per status column -- State, Infra and
  // Agent -- each holding the values that actually occur. Within a section a
  // host matches any value picked; across sections it must match each section
  // that has a pick, so "State: Build Failed + Agent: Healthy" finds failed
  // hosts whose agent is fine.
  const statusOptions = useMemo(() => {
    const seen = new Set(hosts.flatMap(statusKeys))
    const section = (col: StatusColumn, heading: string, label: (v: string) => string): [string, string, string][] =>
      [...seen].filter((k) => k.startsWith(col + ':')).map((k) => k.slice(col.length + 1)).sort().map((v) => [`${col}:${v}`, v === '' ? 'None' : label(v), heading])
    return [
      ...section('state', 'State', statusLabel),
      ...section('infra', 'Infra', powerLabel),
      ...section('agent', 'Agent', (v) => statusLabel(v).replace(/^Agent /, '')),
    ]
  }, [hosts])
  const tagOptions = useMemo(() => [...new Set(hosts.flatMap(hostTagList))].sort(), [hosts, memberByObjId])

  // Free-text search across everything shown (name, kind, network, state,
  // agent, external ref, OS, size, tags), plus dropdown filters for
  // type/OS/state/tag -- the tag filter is the key to operating on a group:
  // filter to a tag, select all, run a task.
  const filtered = useMemo(() => {
    const q = filter.trim().toLowerCase()
    return hosts.filter((h) => {
      if (typeFilter.size && !typeFilter.has(h.kind)) return false
      if (statusFilter.size && !matchesStatusFilter(h, statusFilter)) return false
      if (osFilter.size && !osFilter.has(hostOs(h))) return false
      if (tagFilter.size && !hostTagList(h).some((t) => tagFilter.has(t))) return false
      if (!q) return true
      const mem = memberByObjId.get(h.id)
      return [h.as_name ?? h.object_name, h.kind, h.network_name ?? '', h.status, h.agent?.agent_status ?? '', h.external_ref ?? '', hostOs(h), mem?.size ?? '', ...(mem?.depends_on ?? []), ...hostTagList(h)]
        .some((v) => v.toLowerCase().includes(q))
    })
  }, [hosts, filter, typeFilter, statusFilter, osFilter, tagFilter, memberByObjId])

  // Subdivide by team, then by network within each team -- the shape an
  // operator reasons about ("team 4's DMZ"), collapsible so a 10-team build
  // isn't one endless list. Networkless hosts sort last under a "—" group.
  // The network's own deployed_object per team+name, so a network header can
  // open its detail panel (a network is provisioned too -- it has a log and
  // a status worth seeing, even though it has no agent).
  const networkByKey = useMemo(() => {
    const m = new Map<string, DeployedObject>()
    for (const o of objects) {
      if (o.kind === 'network') m.set(`${o.team_id}:${o.object_name}`, o)
    }
    return m
  }, [objects])

  const grouped = useMemo(() => {
    const byTeamId = new Map<string, DeployedObject[]>()
    for (const h of filtered) {
      const list = byTeamId.get(h.team_id) ?? []
      list.push(h)
      byTeamId.set(h.team_id, list)
    }
    return (build?.teams ?? [])
      .map((t) => {
        const teamHosts = byTeamId.get(t.id) ?? []
        const netNames = Array.from(new Set(teamHosts.map((h) => h.network_name ?? ''))).sort((a, b) => {
          if (a === '') return 1
          if (b === '') return -1
          return a.localeCompare(b)
        })
        const networks = netNames.map((name) => ({
          name,
          hosts: teamHosts.filter((h) => (h.network_name ?? '') === name),
          object: name ? networkByKey.get(`${t.id}:${name}`) : undefined,
        }))
        return { teamNumber: t.team_number, count: teamHosts.length, networks }
      })
      .filter((t) => t.count > 0)
  }, [filtered, build, networkByKey])

  function toggle(id: string) {
    setSelected((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  function toggleAll() {
    // Select-all operates on the currently filtered set, so it means "all
    // the hosts I can see," not a hidden global toggle.
    setSelected((prev) => {
      const allShown = filtered.length > 0 && filtered.every((h) => prev.has(h.id))
      if (allShown) {
        const next = new Set(prev)
        for (const h of filtered) next.delete(h.id)
        return next
      }
      const next = new Set(prev)
      for (const h of filtered) next.add(h.id)
      return next
    })
  }

  function toggleTeam(hostsInTeam: DeployedObject[]) {
    setSelected((prev) => {
      const allOn = hostsInTeam.length > 0 && hostsInTeam.every((h) => prev.has(h.id))
      const next = new Set(prev)
      for (const h of hostsInTeam) (allOn ? next.delete(h.id) : next.add(h.id))
      return next
    })
  }

  function toggleTeamCollapse(teamNumber: number) {
    setCollapsedTeams((prev) => {
      const next = new Set(prev)
      next.has(teamNumber) ? next.delete(teamNumber) : next.add(teamNumber)
      return next
    })
  }

  function toggleNetCollapse(key: string) {
    setCollapsedNets((prev) => {
      const next = new Set(prev)
      next.has(key) ? next.delete(key) : next.add(key)
      return next
    })
  }

  const selectedHosts = hosts.filter((h) => selected.has(h.id))
  // Networks can be selected too, but only for rebuild (they have no agent or
  // power state): rebuilding a failed network recreates it and its guests.
  const selectedNetworks = useMemo(() => (objects ?? []).filter((o) => o.kind === 'network' && selected.has(o.id)), [objects, selected])
  const allFilteredSelected = filtered.length > 0 && filtered.every((h) => selected.has(h.id))

  async function runOnSelected(command: string, payload: unknown) {
    // Target by exact deployed_object id, not a name search -- an
    // earlier version of this used `search: h.as_name`, which silently
    // matched every OTHER team's identically-named host too (every team
    // gets the same network, so hostnames are never unique build-wide),
    // caught by hand in the browser: selecting one "web01" actually
    // rebooted two. See tasks.go's adHocTarget.IDs doc comment.
    await adHoc.mutateAsync({ target: { ids: selectedHosts.map((h) => h.id) }, command, payload })
    setSelected(new Set())
    setConfirming(null)
  }

  async function scheduleOnSelected(command: string, payload: unknown, when: string) {
    // Same exact-id targeting as runOnSelected, for the same reason --
    // see its own comment above.
    await scheduleTask.mutateAsync({ target: { ids: selectedHosts.map((h) => h.id) }, when, command, payload })
    setSelected(new Set())
    setScheduling(null)
  }

  if (isLoading)
    return (
      <div className="flex items-center gap-2 text-sm text-fg-muted">
        <Spinner /> Loading…
      </div>
    )

  if (hosts.length === 0) {
    return <EmptyState icon={Boxes} title="No hosts or containers deployed yet" hint="They'll appear here once a deploy runs." />
  }

  return (
    <div>
      {readOnly && (
        <div className="mb-3 rounded-token border border-border bg-surface-sunken px-3 py-2 text-xs text-fg-muted">
          This build has been torn down — it’s read-only.
        </div>
      )}
      <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
        <div className="flex flex-wrap items-center gap-2">
          <Button variant="secondary" size="sm" onClick={() => setBoardOpen(true)} title="Open the full-screen status matrix for this build">
            <LayoutGrid size={14} /> Matrix
          </Button>
          <Button variant="secondary" size="sm" onClick={() => setGraphOpen(true)} title="Open the full-screen dependency graph for this build">
            <Waypoints size={14} /> Graph
          </Button>
          <Button variant="ghost" size="sm" onClick={() => setLegendOpen(true)}>
            <Info size={14} /> State Legend
          </Button>
          <Button variant="ghost" size="sm" onClick={() => setReportsOpen(true)} title="Past revalidations on this build and their results">
            <ShieldCheck size={14} /> Validation Reports
          </Button>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <div className="relative">
            <Search size={13} className="pointer-events-none absolute left-2 top-1/2 -translate-y-1/2 text-fg-subtle" />
            <input
              value={filter}
              onChange={(e) => setFilter(e.target.value)}
              placeholder="Search hosts…"
              className="h-8 w-48 rounded-token border border-border bg-surface-raised pl-7 pr-2 text-sm text-fg placeholder:text-fg-subtle outline-none focus-visible:border-accent focus-visible:outline-2 focus-visible:outline-offset-0 focus-visible:outline-accent-soft"
            />
          </div>
          <MultiFilter label="Type" selected={typeFilter} onToggle={(v) => setTypeFilter((p) => toggleSet(p, v))} onClear={() => setTypeFilter(new Set())} options={[['host', 'Host'], ['container', 'Container']]} />
          <MultiFilter label="OS" selected={osFilter} onToggle={(v) => setOsFilter((p) => toggleSet(p, v))} onClear={() => setOsFilter(new Set())} options={osOptions.map((o) => [o, o])} />
          <MultiFilter label="Status" selected={statusFilter} onToggle={(v) => setStatusFilter((p) => toggleSet(p, v))} onClear={() => setStatusFilter(new Set())} options={statusOptions} />
          {tagOptions.length > 0 && <MultiFilter label="Tag" selected={tagFilter} onToggle={(v) => setTagFilter((p) => toggleSet(p, v))} onClear={() => setTagFilter(new Set())} options={tagOptions.map((t) => [t, t])} />}
        </div>
      </div>

      {filtered.length === 0 ? (
        <EmptyState icon={Search} title="No hosts match this filter" hint="Clear the filter to see every host again." />
      ) : (
        <TableScroller contain={false}>
          <Table>
            <thead>
              <tr>
                <Th className="w-8">
                  {!readOnly && <input type="checkbox" aria-label="Select all shown" checked={allFilteredSelected} onChange={toggleAll} />}
                </Th>
                <Th>Name</Th>
                <Th>Kind</Th>
                <Th>Spec</Th>
                <Th>State</Th>
                <Th>Infra</Th>
                <Th>Agent</Th>
                <Th>External Ref</Th>
              </tr>
            </thead>
            <tbody>
              {grouped.map((team) => {
                const teamCollapsed = collapsedTeams.has(team.teamNumber)
                const teamHosts = team.networks.flatMap((n) => n.hosts)
                const teamAllOn = teamHosts.every((h) => selected.has(h.id))
                return (
                  <FragmentGroup key={team.teamNumber}>
                    <tr className="bg-surface-sunken/60">
                      <Td>
                        {!readOnly && (
                          <input
                            type="checkbox"
                            aria-label={`Select team ${team.teamNumber}`}
                            checked={teamAllOn}
                            onChange={() => toggleTeam(teamHosts)}
                          />
                        )}
                      </Td>
                      <Td colSpan={7}>
                        <div className="flex items-center gap-4">
                          <button onClick={() => toggleTeamCollapse(team.teamNumber)} className="flex shrink-0 items-center gap-1.5 text-sm font-semibold text-fg">
                            <ChevronDown size={14} className={cn('text-fg-subtle transition-transform', teamCollapsed && '-rotate-90')} />
                            <Users size={14} className="text-fg-subtle" />
                            Team {team.teamNumber}
                            <span className="font-normal text-fg-subtle">({team.count})</span>
                          </button>
                          {/* Per-network status bars, standard statuses, inline so the row
                              stays one line tall -- see TeamNetworkBars. */}
                          <TeamNetworkBars networks={team.networks} className="ml-auto overflow-x-auto" />
                        </div>
                      </Td>
                    </tr>
                    {!teamCollapsed &&
                      team.networks.map((net) => {
                        const key = `${team.teamNumber}:${net.name}`
                        const netCollapsed = collapsedNets.has(key)
                        return (
                          <FragmentGroup key={key}>
                            <tr>
                              <Td>
                                {!readOnly && net.object && (
                                  <input
                                    type="checkbox"
                                    aria-label={`Select network ${net.name || 'network'} for rebuild`}
                                    checked={selected.has(net.object.id)}
                                    onChange={() => toggle(net.object!.id)}
                                  />
                                )}
                              </Td>
                              <Td colSpan={7}>
                                <div className="flex items-center gap-1.5 pl-4 text-xs font-medium text-fg-muted">
                                  <button onClick={() => toggleNetCollapse(key)} aria-label="Collapse network" className="flex items-center">
                                    <ChevronDown size={12} className={cn('text-fg-subtle transition-transform', netCollapsed && '-rotate-90')} />
                                  </button>
                                  <Network size={13} className="text-fg-subtle" />
                                  {net.object ? (
                                    <button
                                      onClick={() => setLogPanelFor({ id: net.object!.id, label: net.name || 'network', kind: 'network', object: net.object, teamNumber: team.teamNumber })}
                                      className="text-fg-muted hover:text-accent hover:underline"
                                    >
                                      {net.name || 'no network'}
                                    </button>
                                  ) : (
                                    <span>{net.name || 'no network'}</span>
                                  )}
                                  <span className="text-fg-subtle">({net.hosts.length})</span>
                                  {net.object && cidrByNetObjId.get(net.object.id) && (
                                    <span className="font-mono text-fg-subtle">{cidrByNetObjId.get(net.object.id)}</span>
                                  )}
                                  {net.object && <StatusBadge status={net.object.status} kind="network" />}
                                </div>
                              </Td>
                            </tr>
                            {!netCollapsed &&
                              net.hosts.map((h) => {
                                const mem = memberByObjId.get(h.id)
                                const spec = mem?.os ?? mem?.image
                                return (
                                  <tr key={h.id}>
                                    <Td>
                                      {!readOnly && <input type="checkbox" checked={selected.has(h.id)} onChange={() => toggle(h.id)} />}
                                    </Td>
                                    <Td className="font-medium">
                                      <button onClick={() => setLogPanelFor({ id: h.id, label: h.as_name ?? h.object_name, kind: h.kind, object: h, teamNumber: team.teamNumber })} className="pl-8 text-left text-fg hover:text-accent hover:underline">
                                        {h.as_name ?? h.object_name}
                                      </button>
                                      {mem?.depends_on && mem.depends_on.length > 0 && (
                                        <div className="pl-8 text-[11px] text-fg-subtle">depends on {mem.depends_on.join(', ')}</div>
                                      )}
                                      {mem?.tags && Object.keys(mem.tags).length > 0 && (
                                        <div className="flex flex-wrap gap-1 pl-8 pt-0.5">
                                          {Object.entries(mem.tags).map(([k, v]) => {
                                            const tag = `${k}=${v}`
                                            return (
                                              <button
                                                key={k}
                                                onClick={() => setTagFilter((p) => toggleSet(p, tag))}
                                                title={`Filter by ${tag}`}
                                                className={cn(
                                                  'rounded-token-sm px-1.5 py-0.5 text-[10px] font-medium',
                                                  tagFilter.has(tag) ? 'bg-accent-soft text-accent-fg' : 'bg-surface-sunken text-fg-muted hover:bg-surface-hover hover:text-fg',
                                                )}
                                              >
                                                {k}={v}
                                              </button>
                                            )
                                          })}
                                        </div>
                                      )}
                                    </Td>
                                    <Td>
                                      <KindBadge kind={h.kind} />
                                    </Td>
                                    <Td className="text-xs text-fg-muted">
                                      {spec || mem?.size ? (
                                        <span className="flex items-center gap-1.5">
                                          {spec && <span className="font-mono">{spec}</span>}
                                          {mem?.size && <Badge tone="neutral" className="rounded-full py-0">{mem.size}</Badge>}
                                        </span>
                                      ) : (
                                        <span className="text-fg-subtle">—</span>
                                      )}
                                    </Td>
                                    <Td>
                                      <span className="flex flex-wrap items-center gap-1">
                                        <StatusBadge status={h.status} kind={h.kind} blockedOn={h.blocked_on} />
                                        {h.steps_changed_at && (
                                          <button
                                            type="button"
                                            onClick={() => !readOnly && setRestarting({ ids: [h.id], fromGroup: 0 })}
                                            title="Its steps or scripts changed in the content; this build holds hosts instead of recreating them. Restart it from a step to apply the change."
                                          >
                                            <Badge tone="warning">steps changed</Badge>
                                          </button>
                                        )}
                                      </span>
                                    </Td>
                                    <Td>
                                      <InfraCell
                                        obj={h}
                                        onAction={(action, obj) => setPowerSingle({ action, id: obj.id, name: obj.as_name ?? obj.object_name, team: teamNumberById.get(obj.team_id) ?? null })}
                                        onRebuild={readOnly ? undefined : (obj) => setRebuilding({ ids: [obj.id], names: [obj.as_name ?? obj.object_name], closedTeams: closedTeamsForHosts([obj]) })}
                                        onRevalidate={readOnly ? undefined : (obj) => setRevalidating([{ id: obj.id, name: obj.as_name ?? obj.object_name }])}
                                        onRestartSteps={readOnly ? undefined : (obj) => setLogPanelFor({ id: obj.id, label: obj.as_name ?? obj.object_name, kind: obj.kind, object: obj, teamNumber: teamNumberById.get(obj.team_id), initialTab: 'steps' })}
                                      />
                                    </Td>
                                    <Td>{h.agent ? <StatusBadge status={h.agent.agent_status} /> : <span className="text-fg-muted">—</span>}</Td>
                                    <Td className="whitespace-nowrap font-mono text-xs text-fg-muted">{h.external_ref ?? '—'}</Td>
                                  </tr>
                                )
                              })}
                          </FragmentGroup>
                        )
                      })}
                  </FragmentGroup>
                )
              })}
            </tbody>
          </Table>
        </TableScroller>
      )}

      {selected.size > 0 && !readOnly && (
        <div className="fixed bottom-6 left-1/2 flex -translate-x-1/2 items-center gap-3 rounded-token-lg border border-border bg-surface-raised px-4 py-2 shadow-panel">
          <span className="text-sm font-medium text-fg">{selected.size} selected</span>

          {/* Infrastructure power -- acts on the instances at the hoster,
             independent of the guest agent. Distinct from the LaForge agent
             tasks on the right. */}
          <span className="flex items-center gap-1" title="Infrastructure power — acts on the VM/container at the hoster, even if its agent is down">
            <Button variant="secondary" size="icon" onClick={() => setPowering('start')} title="Start" aria-label="Start">
              <Play size={14} />
            </Button>
            <Button variant="secondary" size="icon" onClick={() => setPowering('stop')} title="Stop" aria-label="Stop">
              <Square size={14} />
            </Button>
            <Button variant="secondary" size="icon" onClick={() => setPowering('reboot')} title="Reboot" aria-label="Reboot">
              <RotateCw size={14} />
            </Button>
            <Button
              variant="secondary"
              size="icon"
              onClick={() => {
                const objs = [...selectedHosts, ...selectedNetworks]
                setRebuilding({ ids: objs.map((o) => o.id), names: objs.map((o) => o.as_name ?? o.object_name), closedTeams: closedTeamsForHosts(objs) })
              }}
              title="Rebuild — tear down and recreate, cascading to dependents (a network also recreates its guests)"
              aria-label="Rebuild"
            >
              <Hammer size={14} />
            </Button>
            {selectedHosts.length > 0 && (
              <Button
                variant="secondary"
                size="icon"
                onClick={() => setRevalidating(selectedHosts.map((h) => ({ id: h.id, name: h.as_name ?? h.object_name })))}
                title="Revalidate — re-run every validator on these hosts and report; nothing else runs or changes, except Finished ↔ Invalid"
                aria-label="Revalidate"
              >
                <ShieldCheck size={14} />
              </Button>
            )}
            {new Set(selectedHosts.map((h) => h.object_name)).size === 1 && selectedHosts.some(canRestartSteps) && (
              <Button
                variant="secondary"
                size="icon"
                onClick={() => setRestarting({ ids: selectedHosts.map((h) => h.id), fromGroup: 0 })}
                title="Restart from step — run these hosts' builds again from a step, on the boxes as they are"
                aria-label="Restart from step"
              >
                <RotateCcw size={14} />
              </Button>
            )}
          </span>

          <span className="h-4 w-px bg-border" />

          {/* LaForge tasks -- run through each host's own agent. */}
          <Button variant="primary" size="sm" onClick={() => setConfirming({ initialCommand: 'execute', label: `Run a task on ${selected.size} host(s)` })}>
            <Terminal size={14} /> Run Task
          </Button>
          <Button variant="brand" size="sm" onClick={() => setScheduling({ initialCommand: 'execute', label: `Schedule a task on ${selected.size} host(s)` })}>
            <CalendarClock size={14} /> Schedule…
          </Button>
        </div>
      )}

      {powering && buildId && (
        <PowerDialog
          buildId={buildId}
          action={powering}
          targetIds={selectedHosts.map((h) => h.id)}
          matched={selectedHosts.map((h) => ({ name: h.as_name ?? h.object_name, team: teamNumberById.get(h.team_id) ?? null }))}
          onClose={() => setPowering(null)}
          onDone={() => {
            setPowering(null)
            setSelected(new Set())
          }}
        />
      )}

      {powerSingle && buildId && (
        <PowerDialog
          buildId={buildId}
          action={powerSingle.action}
          targetIds={[powerSingle.id]}
          matched={[{ name: powerSingle.name, team: powerSingle.team }]}
          onClose={() => setPowerSingle(null)}
          onDone={() => setPowerSingle(null)}
        />
      )}

      {restarting && buildId && (
        <RestartStepsDialog
          buildId={buildId}
          seedIds={restarting.ids}
          fromGroup={restarting.fromGroup}
          onClose={() => setRestarting(null)}
          onDone={() => {
            setRestarting(null)
            setSelected(new Set())
          }}
        />
      )}

      {revalidating && buildId && (
        <RevalidateDialog
          buildId={buildId}
          targets={revalidating}
          onClose={() => {
            setRevalidating(null)
            setSelected(new Set())
          }}
        />
      )}
      {reportsOpen && buildId && <ValidationReportsDialog buildId={buildId} onClose={() => setReportsOpen(false)} />}

      {rebuilding && buildId && (
        <RebuildDialog
          buildId={buildId}
          targetIds={rebuilding.ids}
          closedTeams={rebuilding.closedTeams}
          onClose={() => setRebuilding(null)}
          onDone={() => {
            setRebuilding(null)
            setSelected(new Set())
          }}
        />
      )}

      {confirming && (
        <ImpactDialog
          title={confirming.label}
          initialCommand={confirming.initialCommand}
          matched={selectedHosts.map((h) => h.as_name ?? h.object_name)}
          onCancel={() => setConfirming(null)}
          onConfirm={(command, payload) => runOnSelected(command, payload)}
          pending={adHoc.isPending}
        />
      )}

      {scheduling && (
        <ScheduleDialog
          buildId={buildId}
          title={scheduling.label}
          initialCommand={scheduling.initialCommand}
          matched={selectedHosts.map((h) => h.as_name ?? h.object_name)}
          onCancel={() => setScheduling(null)}
          onConfirm={(command, payload, when) => scheduleOnSelected(command, payload, when)}
          pending={scheduleTask.isPending}
          error={scheduleTask.error instanceof ApiError ? scheduleTask.error.message : null}
        />
      )}

      {logPanelFor && buildId && (
        <ObjectLogPanel
          repoId={repoId}
          buildId={buildId}
          objectId={logPanelFor.id}
          label={logPanelFor.label}
          kind={logPanelFor.kind}
          object={logPanelFor.object}
          teamNumber={logPanelFor.teamNumber}
          initialTab={logPanelFor.initialTab}
          onRestartFrom={!readOnly && logPanelFor.object && canRestartSteps(logPanelFor.object) ? (group) => setRestarting({ ids: [logPanelFor.id], fromGroup: group }) : undefined}
          onClose={() => {
            setLogPanelFor(null)
            if (inspect) navigate({ to: '/repos/$repoId/builds/$buildId/hosts', params: { repoId, buildId }, search: {}, replace: true })
          }}
        />
      )}

      <StatusLegendDialog open={legendOpen} onClose={() => setLegendOpen(false)} />

      {boardOpen && buildId && <EnvironmentBoard repoId={repoId} buildId={buildId} onClose={() => setBoardOpen(false)} />}
      {graphOpen && buildId && <DependencyGraph repoId={repoId} buildId={buildId} onClose={() => setGraphOpen(false)} />}
    </div>
  )
}

// PowerDialog is the infrastructure counterpart to the task dialogs: a
// hoster-level start/stop/reboot, with the hard-vs-graceful choice made
// explicit for stop and reboot (a hard action cuts power immediately; a
// graceful one asks the guest OS to shut down or reboot cleanly). Start has
// no such distinction. Results are per host, since one wedged instance can
// fail while the rest succeed.
const POWER_LABELS: Record<'start' | 'stop' | 'reboot', string> = { start: 'Start', stop: 'Stop', reboot: 'Reboot' }

function PowerDialog({
  buildId,
  action,
  targetIds,
  matched,
  onClose,
  onDone,
}: {
  buildId: string
  action: 'start' | 'stop' | 'reboot'
  targetIds: string[]
  matched: { name: string; team: number | null }[]
  onClose: () => void
  onDone: () => void
}) {
  const power = usePowerAction(buildId)
  const toast = useToast()
  const [force, setForce] = useState(false)
  const hasForceChoice = action !== 'start'
  const verb = POWER_LABELS[action]

  async function run() {
    try {
      const res = await power.mutateAsync({ target: { ids: targetIds }, action, force: hasForceChoice && force })
      // The action runs in the background; this only says it started. Each
      // host's result shows on its Events tab and in the Infra column.
      const skipped = res.results.filter((r) => r.error)
      toast({
        title: `${verb} started on ${res.started} host(s)`,
        description:
          (skipped.length > 0 ? `Skipped ${skipped.map((r) => `${r.name} (${r.error})`).join(', ')}. ` : '') +
          'Each host’s result appears on its Events tab, and the Infra column updates as hosts finish.',
        tone: skipped.length > 0 ? 'info' : 'success',
        duration: skipped.length > 0 ? 0 : undefined,
      })
      onDone()
    } catch (e) {
      toast({ title: `${verb} failed`, description: e instanceof ApiError ? e.message : undefined, tone: 'danger', duration: 0 })
    }
  }

  return (
    <Modal
      open
      onClose={onClose}
      title={`${verb} ${matched.length} host(s)`}
      description="Infrastructure power, run at the hoster — it acts on the instance itself, so it works even when the guest agent is down."
      footer={
        <>
          <Button variant="ghost" onClick={onClose}>
            Cancel
          </Button>
          <Button variant={hasForceChoice && force ? 'danger' : 'primary'} onClick={run} disabled={power.isPending}>
            {power.isPending ? 'Working…' : hasForceChoice && force ? `Force ${verb}` : verb}
          </Button>
        </>
      }
    >
      {hasForceChoice && (
        <div className="mb-3">
          <Label>Method</Label>
          <div className="mt-1 flex gap-2">
            <button
              onClick={() => setForce(false)}
              className={cn(
                'flex-1 rounded-token border px-3 py-2 text-left text-sm',
                !force ? 'border-accent bg-accent-soft text-accent-fg' : 'border-border text-fg-muted hover:bg-surface-hover',
              )}
            >
              <div className="font-medium">Graceful (OS)</div>
              <div className="text-xs text-fg-subtle">Ask the guest to {action === 'reboot' ? 'reboot' : 'shut down'} cleanly.</div>
            </button>
            <button
              onClick={() => setForce(true)}
              className={cn(
                'flex-1 rounded-token border px-3 py-2 text-left text-sm',
                force ? 'border-danger bg-danger-soft text-danger' : 'border-border text-fg-muted hover:bg-surface-hover',
              )}
            >
              <div className="font-medium">Hard (force)</div>
              <div className="text-xs text-fg-subtle">Cut power immediately.</div>
            </button>
          </div>
          {force && (
            <div className="mt-2 flex items-start gap-1.5 text-xs text-danger">
              <TriangleAlert size={13} className="mt-0.5 shrink-0" />
              <span>A hard {action} cuts the instance immediately — the guest can't flush disks or exit cleanly.</span>
            </div>
          )}
        </div>
      )}

      <Label>Affects</Label>
      <div className="mt-1 max-h-32 overflow-auto rounded-token border border-border bg-surface-sunken p-2 text-xs text-fg-muted">
        {matched.map((h, i) => (
          <div key={`${h.name}-${i}`} className="flex items-center justify-between gap-2 py-0.5">
            <span className="truncate">{h.name}</span>
            {h.team !== null && <span className="shrink-0 text-fg-subtle">team {h.team}</span>}
          </div>
        ))}
      </div>
    </Modal>
  )
}

// RebuildDialog confirms a forced rebuild: tear the selected host(s) down at the
// hoster and recreate them fresh, cascading to everything in their team that
// depends on them. It opens by asking the server for the full blast radius
// (a dry run) so the operator sees exactly what will be destroyed before
// committing -- a rebuild re-runs every step and loses any in-place state.
function RebuildDialog({
  buildId,
  targetIds,
  closedTeams,
  onClose,
  onDone,
}: {
  buildId: string
  targetIds: string[]
  // Team numbers among the targets whose access is currently closed -- a rebuilt
  // host on a closed team can't reach anything but the gateway, so its agent
  // install and step downloads will likely fail. Surfaced as a warning.
  closedTeams: number[]
  onClose: () => void
  onDone: () => void
}) {
  const rebuild = useRebuild(buildId)
  const toast = useToast()
  const [preview, setPreview] = useState<RebuildAffected[] | null>(null)
  const [previewErr, setPreviewErr] = useState<string | null>(null)

  const idsKey = targetIds.join(',')
  useEffect(() => {
    let cancelled = false
    setPreview(null)
    setPreviewErr(null)
    rebuild
      .mutateAsync({ target: { ids: targetIds }, include_dependents: true, dry_run: true })
      .then((res) => {
        if (!cancelled) setPreview(res.affected)
      })
      .catch((e) => {
        if (!cancelled) setPreviewErr(e instanceof ApiError ? e.message : 'Could not compute the rebuild set')
      })
    return () => {
      cancelled = true
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [buildId, idsKey])

  async function run() {
    try {
      const res = await rebuild.mutateAsync({ target: { ids: targetIds }, include_dependents: true, dry_run: false })
      toast({
        title: 'Rebuild started',
        description: `${res.count} host(s) will be torn down and recreated`,
        tone: 'success',
      })
      onDone()
    } catch (e) {
      toast({ title: 'Rebuild failed', description: e instanceof ApiError ? e.message : undefined, tone: 'danger', duration: 0 })
    }
  }

  const count = preview?.length ?? 0
  const extra = preview ? Math.max(0, count - targetIds.length) : 0

  return (
    <Modal
      open
      onClose={onClose}
      title={`Rebuild ${targetIds.length} host(s)`}
      description="Tears the selected host(s) down at the hoster and deploys them fresh, re-running their steps — cascading to everything in their team that depends on them."
      footer={
        <>
          <Button variant="ghost" onClick={onClose}>
            Cancel
          </Button>
          <Button variant="danger" onClick={run} disabled={rebuild.isPending || preview === null || previewErr !== null}>
            {rebuild.isPending && preview !== null ? 'Working…' : count > 0 ? `Rebuild ${count}` : 'Rebuild'}
          </Button>
        </>
      }
    >
      <div className="mb-3 flex items-start gap-1.5 text-xs text-danger">
        <TriangleAlert size={13} className="mt-0.5 shrink-0" />
        <span>This destroys the instances at the hoster and recreates them. Anything changed on them since deploy is lost.</span>
      </div>

      {closedTeams.length > 0 && (
        <div className="mb-3 flex items-start gap-1.5 rounded-token border border-warning bg-warning-soft p-2 text-xs text-fg">
          <TriangleAlert size={13} className="mt-0.5 shrink-0 text-warning" />
          <span>
            Access is <span className="font-medium">closed</span> for team{closedTeams.length > 1 ? 's' : ''} {closedTeams.join(', ')}. A
            rebuilt host can only reach the gateway while access is closed, so its agent install and step downloads will likely fail — open
            access for {closedTeams.length > 1 ? 'those teams' : 'that team'} before rebuilding, or expect the rebuild to hang.
          </span>
        </div>
      )}

      <Label>
        Will rebuild
        {extra > 0 ? ` (${targetIds.length} selected + ${extra} dependent${extra === 1 ? '' : 's'})` : ''}
      </Label>
      {previewErr !== null ? (
        <div className="mt-1 rounded-token border border-danger bg-danger-soft p-2 text-xs text-danger">{previewErr}</div>
      ) : preview === null ? (
        <div className="mt-1 flex items-center gap-2 text-xs text-fg-muted">
          <Spinner /> Computing the rebuild set…
        </div>
      ) : (
        <div className="mt-1 max-h-48 overflow-auto rounded-token border border-border bg-surface-sunken p-2 text-xs text-fg-muted">
          {preview.map((a) => (
            <div key={a.id} className="flex items-center justify-between gap-2 py-0.5">
              <span className="truncate">{a.as_name || a.object_name}</span>
              <span className="shrink-0 text-fg-subtle">
                team {a.team_number} · {a.kind}
              </span>
            </div>
          ))}
        </div>
      )}
    </Modal>
  )
}

// "Show the blast radius before the action... a full list of what will
// be touched." For an `execute` action
// it also takes the actual command to run -- the real ad-hoc command entry
// the UI was missing (direct product feedback: "there is no way to run
// ad-hoc commands"); reboot needs no input.
function ImpactDialog({
  title,
  initialCommand,
  matched,
  onCancel,
  onConfirm,
  pending,
}: {
  title: string
  initialCommand: string
  matched: string[]
  onCancel: () => void
  onConfirm: (command: string, payload: unknown) => void
  pending: boolean
}) {
  const [command, setCommand] = useState(initialCommand)
  const [values, setValues] = useState<Record<string, string>>({})
  const valid = taskFormValid(command, values)
  return (
    <Modal
      open
      onClose={onCancel}
      title={title}
      footer={
        <>
          <Button variant="ghost" onClick={onCancel}>
            Cancel
          </Button>
          <Button variant="primary" onClick={() => onConfirm(command, buildTaskPayload(command, values))} disabled={pending || !valid}>
            {pending ? 'Running…' : 'Run Task'}
          </Button>
        </>
      }
    >
      <TaskActionFields
        command={command}
        values={values}
        onCommandChange={(c) => {
          setCommand(c)
          setValues({})
        }}
        onValueChange={(name, v) => setValues((prev) => ({ ...prev, [name]: v }))}
      />
      <Label>Affects</Label>
      <div className="mt-1 max-h-32 overflow-auto rounded-token border border-border bg-surface-sunken p-2 text-xs text-fg-muted">{matched.join(', ')}</div>
    </Modal>
  )
}

// ScheduleDialog is ImpactDialog's own sibling for "schedule this for
// later" instead of "run it now" -- same target selection, same impact
// confirmation, same command choice, plus the one real difference: a
// `when:` expression (internal/schedule's grammar),
// with the datalist offering
// real example phrases rather than requiring the grammar to be
// memorized. The server validates `when` for real (internal/schedule.Parse)
// and refuses an empty match ("no hosts match this target"), surfaced
// here as `error` rather than a silent failure.
function ScheduleDialog({
  buildId,
  title,
  initialCommand,
  matched,
  onCancel,
  onConfirm,
  pending,
  error,
}: {
  buildId: string
  title: string
  initialCommand: string
  matched: string[]
  onCancel: () => void
  onConfirm: (command: string, payload: unknown, when: string) => void
  pending: boolean
  error: string | null
}) {
  const [command, setCommand] = useState(initialCommand)
  const [values, setValues] = useState<Record<string, string>>({})
  const [when, setWhen] = useState('')
  const [whenValid, setWhenValid] = useState(false)
  const canSchedule = whenValid && taskFormValid(command, values)

  return (
    <Modal
      open
      onClose={onCancel}
      title={title}
      footer={
        <>
          <Button variant="ghost" onClick={onCancel}>
            Cancel
          </Button>
          <Button variant="brand" onClick={() => onConfirm(command, buildTaskPayload(command, values), when)} disabled={pending || !canSchedule}>
            {pending ? 'Scheduling…' : 'Schedule'}
          </Button>
        </>
      }
    >
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

      <Label>Affects</Label>
      <div className="mb-3 mt-1 max-h-28 overflow-auto rounded-token border border-border bg-surface-sunken p-2 text-xs text-fg-muted">{matched.join(', ')}</div>

      {error && <div className="rounded-token border border-danger/30 bg-danger-soft p-2 text-xs text-danger">{error}</div>}
    </Modal>
  )
}
