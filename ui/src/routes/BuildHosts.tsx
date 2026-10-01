import { useParams } from '@tanstack/react-router'
import { Fragment as FragmentGroup, useEffect, useMemo, useState } from 'react'
import { AlertTriangle, Box, CalendarClock, CheckCircle2, ChevronDown, Container, Hammer, Info, Network, Play, Power, Radar, RotateCw, Search, Server, Square, Terminal, TriangleAlert, Users, type LucideIcon } from 'lucide-react'
import { useAdHocTask, useBuild, useCreateScheduledTask, useDetectDrift, usePowerAction, useRebuild, useTopology } from '../api/hooks'
import { StatusBadge, PowerBadge, StatusLegendDialog, GONE_STATES, statusLabel } from '../components/StatusBadge'
import { EmptyState } from '../components/EmptyState'
import { ObjectLogPanel } from '../components/ObjectLogPanel'
import { TaskActionFields, buildTaskPayload, taskFormValid } from '../components/TaskActionForm'
import { WhenField } from '../components/WhenField'
import { Boxes } from 'lucide-react'
import { ApiError } from '../api/client'
import type { DeployedObject, DriftReport, ObjectKind, RebuildAffected, TopologyMember } from '../api/types'
import { Badge, Button, cn, Label, Modal, Spinner, Table, TableScroller, Td, Th, useToast } from '../ui'
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
function InfraCell({ obj, onAction, onRebuild }: { obj: DeployedObject; onAction: (action: PowerAct, obj: DeployedObject) => void; onRebuild?: (obj: DeployedObject) => void }) {
  if (!obj.power_state || GONE_STATES.has(obj.status)) return <span className="text-fg-muted">—</span>
  const actions = POWER_ACTIONS[obj.power_state] ?? []
  const name = obj.as_name ?? obj.object_name
  if (actions.length === 0 && !onRebuild) return <PowerBadge state={obj.power_state} />
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
        {onRebuild && (
          <>
            {actions.length > 0 && <span className="mx-0.5 h-3.5 w-px bg-border" />}
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
      </div>
    </div>
  )
}

// FilterSelect is one "All X / <value>" dropdown for the Hosts toolbar
// (Type / OS / State). Highlights its border when a value is active.
function FilterSelect({ label, value, onChange, options }: { label: string; value: string; onChange: (v: string) => void; options: [string, string][] }) {
  return (
    <select
      value={value}
      onChange={(e) => onChange(e.target.value)}
      aria-label={label}
      className={cn(
        'h-8 rounded-token border bg-surface-raised px-2 text-sm outline-none focus-visible:border-accent focus-visible:outline-2 focus-visible:outline-offset-0 focus-visible:outline-accent-soft',
        value ? 'border-accent text-fg' : 'border-border text-fg-muted',
      )}
    >
      <option value="">All {label}</option>
      {options.map(([v, l]) => (
        <option key={v} value={v}>
          {l}
        </option>
      ))}
    </select>
  )
}

// "Build → Hosts": the deployed hosts/containers as a searchable, filterable
// table (the compact status matrix now lives at the bottom of the Overview).
// Real selection and real ad-hoc dispatch (internal/api/tasks.go) -- "the
// matched set is shown before anything runs, with a count" via the
// ImpactDialog below.
export function BuildHosts() {
  const { buildId } = useParams({ from: '/repos/$repoId/builds/$buildId/hosts' })
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
  const detectDrift = useDetectDrift(buildId)
  const toast = useToast()
  const [confirming, setConfirming] = useState<{ initialCommand: string; label: string } | null>(null)
  const [scheduling, setScheduling] = useState<{ initialCommand: string; label: string } | null>(null)
  const [powering, setPowering] = useState<'start' | 'stop' | 'reboot' | null>(null)
  // A single-instance power action from the Infra pill's hover controls,
  // confirmed in the same PowerDialog the bulk bar uses (one target here).
  const [powerSingle, setPowerSingle] = useState<{ action: 'start' | 'stop' | 'reboot'; id: string; name: string } | null>(null)
  // A forced rebuild (tear down + recreate) of one or more hosts plus their
  // dependents; confirmed in RebuildDialog, which previews the full blast radius.
  const [rebuilding, setRebuilding] = useState<{ ids: string[]; names: string[] } | null>(null)
  const [logPanelFor, setLogPanelFor] = useState<{ id: string; label: string; kind: ObjectKind; object?: DeployedObject; teamNumber?: number } | null>(null)
  const [driftReport, setDriftReport] = useState<DriftReport | null>(null)
  const [driftError, setDriftError] = useState<string | null>(null)
  const [filter, setFilter] = useState('')
  const [typeFilter, setTypeFilter] = useState('')
  const [osFilter, setOsFilter] = useState('')
  const [stateFilter, setStateFilter] = useState('')
  const [tagFilter, setTagFilter] = useState('')
  const [collapsedTeams, setCollapsedTeams] = useState<Set<number>>(new Set())
  const [collapsedNets, setCollapsedNets] = useState<Set<string>>(new Set())
  const [legendOpen, setLegendOpen] = useState(false)

  // Drift detection lives here, on the Hosts page, not in the build-wide
  // header -- it's a question about *these hosts* ("does the hoster still
  // have exactly what we deployed?"), so it belongs next to the host list
  // it reports on rather than floating over Overview/Access/Logs too.
  const driftable = build ? ['deploying', 'building', 'finished', 'failed'].includes(build.status) : false

  async function onDetectDrift() {
    setDriftError(null)
    setDriftReport(null)
    try {
      const report = await detectDrift.mutateAsync()
      setDriftReport(report)
      const clean = report.orphaned.length === 0 && report.missing.length === 0
      toast({
        title: clean ? 'No drift detected' : 'Drift detected',
        description: clean ? undefined : `${report.orphaned.length} orphaned, ${report.missing.length} missing`,
        tone: clean ? 'success' : 'danger',
      })
    } catch (e) {
      const message = e instanceof ApiError ? e.message : 'Detect drift failed'
      setDriftError(message)
      toast({ title: 'Detect drift failed', description: message, tone: 'danger', duration: 0 })
    }
  }

  const objects: DeployedObject[] = useMemo(() => build?.teams.flatMap((t) => t.objects) ?? [], [build])

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
  const stateOptions = useMemo(() => [...new Set(hosts.map((h) => h.status))].sort(), [hosts])
  const tagOptions = useMemo(() => [...new Set(hosts.flatMap(hostTagList))].sort(), [hosts, memberByObjId])

  // Free-text search across everything shown (name, kind, network, state,
  // agent, external ref, OS, size, tags), plus dropdown filters for
  // type/OS/state/tag -- the tag filter is the key to operating on a group:
  // filter to a tag, select all, run a task.
  const filtered = useMemo(() => {
    const q = filter.trim().toLowerCase()
    return hosts.filter((h) => {
      if (typeFilter && h.kind !== typeFilter) return false
      if (stateFilter && h.status !== stateFilter) return false
      if (osFilter && hostOs(h) !== osFilter) return false
      if (tagFilter && !hostTagList(h).includes(tagFilter)) return false
      if (!q) return true
      const mem = memberByObjId.get(h.id)
      return [h.as_name ?? h.object_name, h.kind, h.network_name ?? '', h.status, h.agent?.agent_status ?? '', h.external_ref ?? '', hostOs(h), mem?.size ?? '', ...(mem?.depends_on ?? []), ...hostTagList(h)]
        .some((v) => v.toLowerCase().includes(q))
    })
  }, [hosts, filter, typeFilter, stateFilter, osFilter, tagFilter, memberByObjId])

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
        <div className="flex items-center gap-2">
          {driftable && (
            <Button variant="secondary" size="sm" onClick={onDetectDrift} disabled={detectDrift.isPending}>
              <Radar size={14} /> {detectDrift.isPending ? 'Detecting…' : 'Detect Drift'}
            </Button>
          )}
          {driftError && <span className="text-xs text-danger">{driftError}</span>}
          <Button variant="ghost" size="sm" onClick={() => setLegendOpen(true)}>
            <Info size={14} /> State Legend
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
          <FilterSelect label="Type" value={typeFilter} onChange={setTypeFilter} options={[['host', 'Host'], ['container', 'Container']]} />
          <FilterSelect label="OS" value={osFilter} onChange={setOsFilter} options={osOptions.map((o) => [o, o])} />
          <FilterSelect label="State" value={stateFilter} onChange={setStateFilter} options={stateOptions.map((s) => [s, statusLabel(s)])} />
          {tagOptions.length > 0 && <FilterSelect label="Tag" value={tagFilter} onChange={setTagFilter} options={tagOptions.map((t) => [t, t])} />}
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
                        <button onClick={() => toggleTeamCollapse(team.teamNumber)} className="flex items-center gap-1.5 text-sm font-semibold text-fg">
                          <ChevronDown size={14} className={cn('text-fg-subtle transition-transform', teamCollapsed && '-rotate-90')} />
                          <Users size={14} className="text-fg-subtle" />
                          Team {team.teamNumber}
                          <span className="font-normal text-fg-subtle">({team.count})</span>
                        </button>
                      </Td>
                    </tr>
                    {!teamCollapsed &&
                      team.networks.map((net) => {
                        const key = `${team.teamNumber}:${net.name}`
                        const netCollapsed = collapsedNets.has(key)
                        return (
                          <FragmentGroup key={key}>
                            <tr>
                              <Td></Td>
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
                                                onClick={() => setTagFilter(tagFilter === tag ? '' : tag)}
                                                title={`Filter by ${tag}`}
                                                className="rounded-token-sm bg-surface-sunken px-1.5 py-0.5 text-[10px] font-medium text-fg-muted hover:bg-surface-hover hover:text-fg"
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
                                      <StatusBadge status={h.status} kind={h.kind} />
                                    </Td>
                                    <Td>
                                      <InfraCell
                                        obj={h}
                                        onAction={(action, obj) => setPowerSingle({ action, id: obj.id, name: obj.as_name ?? obj.object_name })}
                                        onRebuild={readOnly ? undefined : (obj) => setRebuilding({ ids: [obj.id], names: [obj.as_name ?? obj.object_name] })}
                                      />
                                    </Td>
                                    <Td>{h.agent ? <StatusBadge status={h.agent.agent_status} /> : <span className="text-fg-muted">—</span>}</Td>
                                    <Td className="font-mono text-xs text-fg-muted">{h.external_ref ?? '—'}</Td>
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
            <Power size={13} className="text-fg-subtle" />
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
              onClick={() => setRebuilding({ ids: selectedHosts.map((h) => h.id), names: selectedHosts.map((h) => h.as_name ?? h.object_name) })}
              title="Rebuild — tear down and recreate, cascading to dependents"
              aria-label="Rebuild"
            >
              <Hammer size={14} />
            </Button>
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
          matched={selectedHosts.map((h) => h.as_name ?? h.object_name)}
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
          matched={[powerSingle.name]}
          onClose={() => setPowerSingle(null)}
          onDone={() => setPowerSingle(null)}
        />
      )}

      {rebuilding && buildId && (
        <RebuildDialog
          buildId={buildId}
          targetIds={rebuilding.ids}
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
        <ObjectLogPanel buildId={buildId} objectId={logPanelFor.id} label={logPanelFor.label} kind={logPanelFor.kind} object={logPanelFor.object} teamNumber={logPanelFor.teamNumber} onClose={() => setLogPanelFor(null)} />
      )}

      <StatusLegendDialog open={legendOpen} onClose={() => setLegendOpen(false)} />

      <Modal
        open={!!driftReport}
        onClose={() => setDriftReport(null)}
        title="Drift Report"
        description="Builder.Inspect's real, live answer for this build's own builder, compared against what LaForge believes is deployed."
      >
        {driftReport && <DriftReportBody report={driftReport} />}
      </Modal>
    </div>
  )
}

// Live drift for this build's hosts: what the hoster actually has, versus
// what LaForge's records say it deployed. Moved here from the build-wide
// header (see onDetectDrift's comment) so it reports right beside the host
// list it's about.
function DriftReportBody({ report }: { report: DriftReport }) {
  if (report.orphaned.length === 0 && report.missing.length === 0) {
    return (
      <div className="flex items-center gap-2 rounded-token border border-success/30 bg-success-soft px-3 py-2 text-sm text-success">
        <CheckCircle2 size={16} /> No drift -- {report.tracked} resource{report.tracked === 1 ? '' : 's'} match exactly what the hoster has.
      </div>
    )
  }
  return (
    <div className="flex flex-col gap-4">
      <div className="text-xs text-fg-muted">{report.tracked} resource{report.tracked === 1 ? '' : 's'} confirmed matching.</div>

      {report.missing.length > 0 && (
        <div>
          <div className="mb-1 flex items-center gap-1.5 text-sm font-medium text-danger">
            <AlertTriangle size={14} /> Missing at the hoster ({report.missing.length})
          </div>
          <div className="mb-2 text-xs text-fg-muted">LaForge believes these are deployed, but the hoster no longer has them.</div>
          <TableScroller contain={false}>
            <Table>
              <thead>
                <tr>
                  <Th>Kind</Th>
                  <Th>Name</Th>
                  <Th>External ref</Th>
                </tr>
              </thead>
              <tbody>
                {report.missing.map((o) => (
                  <tr key={o.id}>
                    <Td><Badge tone="neutral">{o.kind}</Badge></Td>
                    <Td>{o.as_name ?? o.object_name}</Td>
                    <Td className="font-mono text-xs text-fg-muted">{o.external_ref}</Td>
                  </tr>
                ))}
              </tbody>
            </Table>
          </TableScroller>
        </div>
      )}

      {report.orphaned.length > 0 && (
        <div>
          <div className="mb-1 flex items-center gap-1.5 text-sm font-medium text-warning">
            <AlertTriangle size={14} /> Orphaned at the hoster ({report.orphaned.length})
          </div>
          <div className="mb-2 text-xs text-fg-muted">The hoster has these, but LaForge has no record of them for this build.</div>
          <TableScroller contain={false}>
            <Table>
              <thead>
                <tr>
                  <Th>Kind</Th>
                  <Th>External ref</Th>
                </tr>
              </thead>
              <tbody>
                {report.orphaned.map((r) => (
                  <tr key={r.external_ref}>
                    <Td><Badge tone="neutral">{r.kind}</Badge></Td>
                    <Td className="font-mono text-xs text-fg-muted">{r.external_ref}</Td>
                  </tr>
                ))}
              </tbody>
            </Table>
          </TableScroller>
        </div>
      )}
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
  matched: string[]
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
      const failed = res.failed ?? 0
      toast({
        title: failed === 0 ? `${verb} sent` : `${verb}: ${failed} failed`,
        description: failed === 0 ? `${res.results.length} host(s)` : res.results.filter((r) => r.error).map((r) => `${r.name}: ${r.error}`).join('; '),
        tone: failed === 0 ? 'success' : 'danger',
        duration: failed === 0 ? undefined : 0,
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
      <div className="mt-1 max-h-32 overflow-auto rounded-token border border-border bg-surface-sunken p-2 text-xs text-fg-muted">{matched.join(', ')}</div>
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
  onClose,
  onDone,
}: {
  buildId: string
  targetIds: string[]
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
