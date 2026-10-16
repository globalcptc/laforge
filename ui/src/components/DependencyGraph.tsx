import { useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react'
import { useQueries } from '@tanstack/react-query'
import { X, Boxes } from 'lucide-react'
import { api } from '../api/client'
import { useBuild, useLiveEvents, useTopology } from '../api/hooks'
import type { DeployedObject, ObjectKind, StepStatus } from '../api/types'
import { statusIcon, statusAnimClass, statusLabel, blockedStatusKey } from './StatusBadge'
import { STATUS_COLOR, BOARD, statusColor, isGhostStatus, BOARD_STATUS_ORDER } from './statusPalette'
import { ObjectLogPanel } from './ObjectLogPanel'
import { EmptyState } from './EmptyState'
import { cn } from '../ui'

// The Dependency Graph: a build's objects as a top-to-bottom dependency flow.
// Each node is a fixed-size status bar (icon + name, coloured like the matrix),
// with a strip of per-authored-step mini-bars inside each host. Edges follow
// depends_on (node-to-node) and wait_for (attached to the exact milestone step a
// dependent waits on), coloured by that gate's live state. Dark-committed to
// match the matrix. Hover for detail; click a host to open its panel; click a
// step to deep-link to that step in the host's Steps tab.

type StepGroup = { index: number; label?: string; status: string; color: string; running: boolean; milestone?: string; waitFor: string[] }
type GNode = { id: string; name: string; kind: ObjectKind; net: string; key: string; obj: DeployedObject; deps: string[]; groups: StepGroup[]; depth: number }
type GEdge = { fromId: string; fromMs?: number; toId: string; gate: string }

// Aggregate an authored step group's command statuses into one state + colour.
function groupStatus(steps: StepStatus[]): { status: string; color: string; running: boolean } {
  let status = 'pending'
  if (steps.some((s) => s.status === 'failed')) status = 'build_failed'
  else if (steps.some((s) => s.status === 'leased')) status = 'running'
  else if (steps.some((s) => s.status === 'blocked')) status = 'dependency_blocked'
  else if (steps.every((s) => s.status === 'done' || s.status === 'ignored')) status = 'finished'
  const color =
    status === 'finished' ? STATUS_COLOR.finished
    : status === 'running' ? STATUS_COLOR.building
    : status === 'build_failed' ? STATUS_COLOR.build_failed
    : status === 'dependency_blocked' ? STATUS_COLOR.dependency_blocked
    : '#2a2740'
  return { status, color, running: status === 'running' }
}

// The gate colour key for an edge, from what it's waiting on.
function gateFromNodeStatus(key: string): string {
  if (key === 'finished') return 'finished'
  if (['deploy_failed', 'build_failed', 'invalid', 'destroy_failed', 'failed'].includes(key)) return 'build_failed'
  if (['dependency_blocked', 'pending', 'planned'].includes(key)) return 'dependency_blocked'
  return 'building'
}
function gateFromStepStatus(status: string): string {
  return status === 'finished' ? 'finished' : status === 'build_failed' ? 'build_failed' : status === 'running' ? 'building' : 'dependency_blocked'
}

const NET_ORDER: Record<string, number> = {}

export function DependencyGraph({ repoId, buildId, onClose }: { repoId: string; buildId: string; onClose: () => void }) {
  const { data: build } = useBuild(buildId)
  const { data: topology } = useTopology(buildId)
  const { connected } = useLiveEvents(buildId)
  const [teamId, setTeamId] = useState<string | null>(null)
  const [selected, setSelected] = useState<{ object: DeployedObject; initialTab?: 'info' | 'steps'; focusGroup?: number; teamNumber: number } | null>(null)
  const [hoverId, setHoverId] = useState<string | null>(null)

  const teams = useMemo(() => [...(build?.teams ?? [])].sort((a, b) => a.team_number - b.team_number), [build])
  const team = useMemo(() => teams.find((t) => t.id === teamId) ?? teams[0], [teams, teamId])

  // depends_on + network/cidr facts per object, from the topology.
  const memberDeps = useMemo(() => {
    const m = new Map<string, string[]>()
    for (const t of topology?.teams ?? []) for (const n of t.networks) for (const mem of n.members) m.set(mem.object_id, mem.depends_on ?? [])
    return m
  }, [topology])

  // Steps for every host/container in the selected team (networks have none).
  const hostObjs = useMemo(() => (team?.objects ?? []).filter((o) => o.kind !== 'network'), [team])
  const stepResults = useQueries({
    queries: hostObjs.map((o) => ({
      queryKey: ['object-steps', buildId, o.id],
      queryFn: () => api.get<StepStatus[]>(`/builds/${buildId}/objects/${o.id}/steps`),
      enabled: !!buildId,
      refetchInterval: 8000,
    })),
  })
  const stepsSig = stepResults.map((r) => r.dataUpdatedAt ?? 0).join('|')

  const model = useMemo(() => {
    if (!team) return { levels: [] as GNode[][], edges: [] as GEdge[], nodes: new Map<string, GNode>(), dependents: new Map<string, string[]>() }
    const objects = team.objects
    const idByName = new Map<string, string>()
    for (const o of objects) idByName.set(o.object_name, o.id)
    const stepsById = new Map<string, StepStatus[]>()
    hostObjs.forEach((o, i) => { const d = stepResults[i]?.data; if (d) stepsById.set(o.id, d) })

    // Nodes.
    const nodes = new Map<string, GNode>()
    for (const o of objects) {
      const deps: string[] = []
      if (o.network_name && idByName.has(o.network_name)) deps.push(idByName.get(o.network_name)!)
      for (const dn of memberDeps.get(o.id) ?? []) { const id = idByName.get(dn); if (id) deps.push(id) }
      // Build authored-step groups.
      const raw = stepsById.get(o.id) ?? []
      const groups: StepGroup[] = []
      for (const s of raw) {
        const last = groups[groups.length - 1]
        if (last && last.index === s.group_index) {
          if (s.milestone) last.milestone = s.milestone
          if (s.wait_for) last.waitFor = Array.from(new Set([...last.waitFor, ...s.wait_for]))
        } else {
          groups.push({ index: s.group_index, label: s.group_label, status: 'pending', color: '#2a2740', running: false, milestone: s.milestone, waitFor: s.wait_for ?? [] })
        }
      }
      // Aggregate each group's status from its commands.
      const byGroup = new Map<number, StepStatus[]>()
      for (const s of raw) { const a = byGroup.get(s.group_index) ?? []; a.push(s); byGroup.set(s.group_index, a) }
      for (const g of groups) { const agg = groupStatus(byGroup.get(g.index) ?? []); g.status = agg.status; g.color = agg.color; g.running = agg.running }
      nodes.set(o.id, { id: o.id, name: o.object_name, kind: o.kind, net: o.network_name ?? '', key: blockedStatusKey(o.status, o.blocked_on), obj: o, deps: Array.from(new Set(deps)), groups, depth: 0 })
    }

    // Depth = longest dependency path.
    const depthMemo = new Map<string, number>()
    function depth(id: string, seen: Set<string>): number {
      if (depthMemo.has(id)) return depthMemo.get(id)!
      const n = nodes.get(id)
      if (!n || n.deps.length === 0 || seen.has(id)) { depthMemo.set(id, 0); return 0 }
      seen.add(id)
      let d = 0
      for (const dp of n.deps) d = Math.max(d, depth(dp, seen) + 1)
      seen.delete(id)
      depthMemo.set(id, d)
      return d
    }
    for (const n of nodes.values()) n.depth = depth(n.id, new Set())

    // Edges. wait_for (step-attached) first, recording host pairs to dedupe.
    const edges: GEdge[] = []
    const waitPairs = new Set<string>()
    for (const n of nodes.values()) {
      for (const g of n.groups) {
        for (const wf of g.waitFor) {
          const [srcName, milestone] = wf.split(' · ')
          const srcId = idByName.get((srcName ?? '').trim())
          if (!srcId) continue
          const src = nodes.get(srcId)
          if (!src) continue
          const ms = src.groups.find((x) => x.milestone === (milestone ?? '').trim())
          edges.push({ fromId: srcId, fromMs: ms?.index, toId: n.id, gate: ms ? gateFromStepStatus(ms.status) : gateFromNodeStatus(src.key) })
          waitPairs.add(srcId + '>' + n.id)
        }
      }
    }
    // Node-level depends_on / network edges (skip host pairs a wait_for already covers).
    for (const n of nodes.values()) {
      for (const dep of n.deps) {
        const src = nodes.get(dep)
        if (!src) continue
        if (src.kind !== 'network' && waitPairs.has(dep + '>' + n.id)) continue
        edges.push({ fromId: dep, toId: n.id, gate: gateFromNodeStatus(src.key) })
      }
    }

    // Dependents, for hover highlighting.
    const dependents = new Map<string, string[]>()
    for (const n of nodes.values()) dependents.set(n.id, [])
    for (const n of nodes.values()) for (const dp of n.deps) dependents.get(dp)?.push(n.id)

    const maxD = Math.max(0, ...Array.from(nodes.values()).map((n) => n.depth))
    const levels: GNode[][] = []
    for (let d = 0; d <= maxD; d++) {
      levels.push(
        Array.from(nodes.values())
          .filter((n) => n.depth === d)
          .sort((a, b) => (NET_ORDER[a.net] ?? 9) - (NET_ORDER[b.net] ?? 9) || a.name.localeCompare(b.name)),
      )
    }
    return { levels, edges, nodes, dependents }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [team, memberDeps, stepsSig, hostObjs])

  // --- Edge geometry (measured against the laid-out DOM) ---
  const graphRef = useRef<HTMLDivElement>(null)
  const nodeRefs = useRef<Map<string, HTMLElement>>(new Map())
  const msRefs = useRef<Map<string, HTMLElement>>(new Map())
  const [paths, setPaths] = useState<{ d: string; gate: string; fromId: string; toId: string }[]>([])
  const [size, setSize] = useState({ w: 0, h: 0 })
  const [tick, setTick] = useState(0)

  useLayoutEffect(() => {
    const g = graphRef.current
    if (!g) return
    const gr = g.getBoundingClientRect()
    setSize({ w: g.offsetWidth, h: g.offsetHeight })
    const out: { d: string; gate: string; fromId: string; toId: string }[] = []
    for (const e of model.edges) {
      const toEl = nodeRefs.current.get(e.toId)
      if (!toEl) continue
      const tr = toEl.getBoundingClientRect()
      const x2 = tr.left - gr.left + tr.width / 2
      const y2 = tr.top - gr.top
      let x1: number | null = null
      let y1 = 0
      if (e.fromMs != null) {
        const ms = msRefs.current.get(e.fromId + '#' + e.fromMs)
        if (ms) { const mr = ms.getBoundingClientRect(); x1 = mr.left - gr.left + mr.width / 2; y1 = mr.bottom - gr.top }
      }
      if (x1 == null) {
        const fe = nodeRefs.current.get(e.fromId)
        if (!fe) continue
        const fr = fe.getBoundingClientRect()
        x1 = fr.left - gr.left + fr.width / 2
        y1 = fr.bottom - gr.top
      }
      const k = Math.max(24, (y2 - y1) / 2)
      out.push({ d: `M${x1} ${y1} C${x1} ${y1 + k} ${x2} ${y2 - k} ${x2} ${y2}`, gate: e.gate, fromId: e.fromId, toId: e.toId })
    }
    setPaths(out)
  }, [model, tick])

  useEffect(() => {
    const g = graphRef.current
    if (!g || !window.ResizeObserver) return
    const ro = new ResizeObserver(() => setTick((t) => t + 1))
    ro.observe(g)
    return () => ro.disconnect()
  }, [])
  // One more pass after fonts/layout settle.
  useEffect(() => {
    const id = requestAnimationFrame(() => setTick((t) => t + 1))
    return () => cancelAnimationFrame(id)
  }, [model])

  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if (e.key !== 'Escape') return
      if (selected) setSelected(null)
      else onClose()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [selected, onClose])

  // Hover tooltip (imperative, to avoid a re-render per mousemove).
  const tipRef = useRef<HTMLDivElement>(null)
  function showTip(e: React.MouseEvent, html: string) {
    const t = tipRef.current
    if (!t) return
    t.innerHTML = html
    t.style.opacity = '1'
    t.style.left = Math.min(e.clientX + 14, window.innerWidth - t.offsetWidth - 14) + 'px'
    t.style.top = e.clientY + 16 + 'px'
  }
  function hideTip() { if (tipRef.current) tipRef.current.style.opacity = '0' }

  const related = useMemo(() => {
    if (!hoverId) return null
    const s = new Set<string>([hoverId])
    model.nodes.get(hoverId)?.deps.forEach((d) => s.add(d))
    model.dependents.get(hoverId)?.forEach((d) => s.add(d))
    return s
  }, [hoverId, model])

  function nodeTip(n: GNode) {
    const waiting = n.obj.blocked_on ?? []
    const deps = n.deps.map((id) => model.nodes.get(id)?.name).filter(Boolean)
    return (
      `<b>${n.name}</b> · ${n.kind}` +
      `<div class="r">state <b>${statusLabel(n.key)}</b></div>` +
      `<div class="r">depends on <b>${deps.length ? deps.join(', ') : '—'}</b></div>` +
      (waiting.length ? `<div class="r">waiting on <b>${waiting.join(', ')}</b></div>` : '')
    )
  }
  function stepTip(n: GNode, g: StepGroup) {
    return (
      `<b>${n.name}</b> · ${g.label ?? 'step ' + g.index}` +
      `<div class="r">state <b>${statusLabel(g.status === 'running' ? 'building' : g.status)}</b></div>` +
      (g.milestone ? `<div class="r">milestone <b>${g.milestone}</b></div>` : '') +
      (g.waitFor.length ? `<div class="r">waits for <b>${g.waitFor.join(', ')}</b></div>` : '')
    )
  }

  const empty = !team || model.nodes.size === 0

  return (
    <>
      <div className="fixed inset-0 z-50 flex flex-col" style={{ background: BOARD.bg, color: BOARD.fg }}>
        <header className="flex shrink-0 flex-wrap items-center gap-x-4 gap-y-2 px-4 py-2.5" style={{ borderBottom: `1px solid ${BOARD.line}` }}>
          <div className="flex items-center gap-2.5">
            <span className="h-2.5 w-2.5 rounded-full" style={{ background: connected ? STATUS_COLOR.finished : BOARD.faint, animation: connected ? 'pulse 1.8s ease-in-out infinite' : undefined }} title={connected ? 'Live' : 'Reconnecting…'} />
            <span className="text-base font-semibold tracking-tight">{build?.environment_name ?? 'Environment'}</span>
            <span className="text-sm" style={{ color: BOARD.muted }}>· dependency graph</span>
          </div>
          <select
            value={team?.id ?? ''}
            onChange={(e) => setTeamId(e.target.value)}
            className="rounded-token px-2.5 py-1.5 text-sm"
            style={{ background: BOARD.panel, border: `1px solid ${BOARD.line2}`, color: BOARD.fg }}
          >
            {teams.map((t) => (
              <option key={t.id} value={t.id}>Team {t.team_number}</option>
            ))}
          </select>
          <button
            onClick={onClose}
            aria-label="Close dependency graph"
            className="ml-auto flex h-8 w-8 items-center justify-center rounded-token"
            style={{ color: BOARD.muted }}
          >
            <X size={18} />
          </button>
        </header>

        {empty ? (
          <div className="flex flex-1 items-center justify-center" style={{ color: BOARD.muted }}>
            <EmptyState icon={Boxes} title="Nothing deployed yet" hint="Objects appear here as the build runs." />
          </div>
        ) : (
          <div className="min-h-0 flex-1 overflow-auto" style={{ background: '#0d0a17' }}>
            <div ref={graphRef} className="relative" style={{ minWidth: 'min-content' }}>
              <svg className="pointer-events-none absolute inset-0 overflow-visible" width={size.w} height={size.h} viewBox={`0 0 ${size.w} ${size.h}`}>
                {paths.map((p, i) => {
                  const on = !hoverId || p.fromId === hoverId || p.toId === hoverId
                  return (
                    <path
                      key={i}
                      d={p.d}
                      fill="none"
                      stroke={statusColor(p.gate)}
                      strokeWidth={on && hoverId ? 2.3 : 1.6}
                      strokeDasharray={p.gate === 'build_failed' ? '5 3' : undefined}
                      opacity={on ? (hoverId ? 0.95 : 0.72) : 0.07}
                    />
                  )
                })}
              </svg>
              <div className="relative flex flex-col gap-[62px] px-8 pb-4 pt-6">
                {model.levels.map((row, li) => (
                  <div key={li} className="flex justify-center gap-10">
                    {row.map((n) => (
                      <GraphNode
                        key={n.id}
                        node={n}
                        dim={!!hoverId && !related?.has(n.id)}
                        setNodeRef={(el) => { if (el) nodeRefs.current.set(n.id, el); else nodeRefs.current.delete(n.id) }}
                        setMsRef={(idx, el) => { const k = n.id + '#' + idx; if (el) msRefs.current.set(k, el); else msRefs.current.delete(k) }}
                        onEnter={() => setHoverId(n.id)}
                        onLeave={() => { setHoverId(null); hideTip() }}
                        onNodeMove={(e) => showTip(e, nodeTip(n))}
                        onStepMove={(e, g) => showTip(e, stepTip(n, g))}
                        onOpenNode={() => setSelected({ object: n.obj, teamNumber: team!.team_number })}
                        onOpenStep={(g) => setSelected({ object: n.obj, initialTab: 'steps', focusGroup: g.index, teamNumber: team!.team_number })}
                      />
                    ))}
                  </div>
                ))}
              </div>
            </div>
          </div>
        )}

        {/* Legend — matches the matrix. */}
        <div className="flex shrink-0 flex-wrap items-center gap-x-4 gap-y-1.5 px-4 py-2 text-xs" style={{ borderTop: `1px solid ${BOARD.line}`, color: BOARD.muted }}>
          {BOARD_STATUS_ORDER.map((k) => {
            const Icon = statusIcon(k)
            const ghost = isGhostStatus(k)
            return (
              <span key={k} className="flex items-center gap-1.5">
                <span className="flex h-4 w-4 items-center justify-center rounded" style={ghost ? { border: `1px dashed ${BOARD.line2}` } : { background: statusColor(k) }}>
                  <Icon size={10} color="#fff" />
                </span>
                {statusLabel(k)}
              </span>
            )
          })}
          <span className="h-3.5 w-px" style={{ background: BOARD.line2 }} />
          {([['finished', 'satisfied'], ['building', 'gate running'], ['dependency_blocked', 'blocked'], ['build_failed', 'failed']] as const).map(([k, lbl]) => (
            <span key={k} className="flex items-center gap-1.5"><span className="inline-block w-4" style={{ borderTop: `2px solid ${statusColor(k)}` }} />{lbl}</span>
          ))}
        </div>
      </div>

      <div ref={tipRef} className="pointer-events-none fixed z-[70] max-w-[270px] rounded-token px-2.5 py-2 text-xs opacity-0 transition-opacity" style={{ background: '#060410', border: `1px solid ${BOARD.line2}`, color: BOARD.fg, boxShadow: '0 8px 24px rgba(0,0,0,.5)' }} />

      {selected && (
        <div className="relative z-[60]">
          <ObjectLogPanel
            repoId={repoId}
            buildId={buildId}
            objectId={selected.object.id}
            label={selected.object.as_name ?? selected.object.object_name}
            kind={selected.object.kind}
            object={selected.object}
            teamNumber={selected.teamNumber}
            initialTab={selected.initialTab}
            focusGroup={selected.focusGroup}
            onClose={() => setSelected(null)}
          />
        </div>
      )}
    </>
  )
}

function GraphNode({
  node: n,
  dim,
  setNodeRef,
  setMsRef,
  onEnter,
  onLeave,
  onNodeMove,
  onStepMove,
  onOpenNode,
  onOpenStep,
}: {
  node: GNode
  dim: boolean
  setNodeRef: (el: HTMLElement | null) => void
  setMsRef: (idx: number, el: HTMLElement | null) => void
  onEnter: () => void
  onLeave: () => void
  onNodeMove: (e: React.MouseEvent) => void
  onStepMove: (e: React.MouseEvent, g: StepGroup) => void
  onOpenNode: () => void
  onOpenStep: (g: StepGroup) => void
}) {
  const ghost = isGhostStatus(n.key)
  const Icon = statusIcon(n.key)
  return (
    <div
      ref={setNodeRef}
      onMouseEnter={onEnter}
      onMouseLeave={onLeave}
      className="transition-opacity"
      style={{ width: 198, background: BOARD.panel, border: `1px solid ${BOARD.line}`, borderRadius: 9, opacity: dim ? 0.26 : 1 }}
    >
      <button
        onClick={onOpenNode}
        onMouseMove={onNodeMove}
        className="flex w-full items-center gap-2 px-2.5 py-1.5 text-left text-[13px] font-medium"
        style={ghost
          ? { background: 'transparent', color: BOARD.muted, border: `1px dashed #352f4d`, borderRadius: 9 }
          : { background: statusColor(n.key), color: '#fff', borderRadius: n.groups.length ? '9px 9px 0 0' : 9 }}
        title=""
      >
        <Icon size={15} className={statusAnimClass(n.key)} />
        <span className="truncate font-mono">{n.name}</span>
        <span className="ml-auto text-[10px] opacity-80">{n.kind}</span>
      </button>
      {n.groups.length > 0 && (
        <div className="flex gap-0.5 px-2 pb-2 pt-2.5" style={{ background: BOARD.panel2, borderRadius: '0 0 9px 9px' }}>
          {n.groups.map((g) => (
            <button
              key={g.index}
              ref={g.milestone ? (el) => setMsRef(g.index, el) : undefined}
              onClick={(e) => { e.stopPropagation(); onOpenStep(g) }}
              onMouseMove={(e) => onStepMove(e, g)}
              className={cn('relative h-1.5 min-w-[3px] flex-1 rounded-sm', g.running && 'animate-pulse')}
              style={{ background: g.color, boxShadow: g.milestone ? `0 0 0 1.5px #0d0a17, 0 0 0 2.5px ${BOARD.fg}` : undefined }}
              title=""
            >
              {g.milestone && (
                <span className="absolute left-1/2 top-[-9px] h-[7px] w-[7px] -translate-x-1/2 rounded-full" style={{ background: g.color, border: `1.5px solid ${BOARD.fg}` }} />
              )}
            </button>
          ))}
        </div>
      )}
    </div>
  )
}
