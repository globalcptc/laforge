import { useEffect, useMemo, useRef, useState } from 'react'
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
// Dependency lines connect directly (a line per edge, from a milestone step or a
// source's left edge), and every node is placed to the RIGHT of whatever lines
// still pass its row -- so lines never cross a card and the layout stays narrow.
// Each host/container carries a strip of per-authored-step mini-bars. Identical
// per-person instances collapse into one stacked node with a count. Dark-committed
// to match the matrix. Hover for detail; click a host for its panel, a step to
// deep-link to that step.

type SG = { index: number; label?: string; key: string; running: boolean; milestone?: string; waitFor: string[] }
type Dep = { on: string; milestone?: string; msIdx?: number }
type GNode = {
  id: string; name: string; objName: string; kind: ObjectKind; net: string; key: string
  count: number; obj: DeployedObject; groups: SG[]; deps: Dep[]
  depth: number; y: number; h: number; cy: number; x: number; lineX: number; src: boolean; i: number
}
type GEdge = { from: string; to: string; gate: string; msIdx?: number }

const CARD_W = 272, NET_H = 38, HOST_H = 58, ROW_GAP = 14, BAR = 17, LINE_OFF = 16, NODE_GAP = 12, BASE_X = 12, TOP_PAD = 16
const stepCenterX = (k: number, i: number) => 9 + (i + 0.5) * ((CARD_W - 16) / Math.max(1, k))

const PRI: Record<string, number> = { deploy_failed: 5, build_failed: 5, invalid: 5, destroy_failed: 5, failed: 5, dependency_blocked: 4, building: 3, running: 3, deploying: 3, destroying: 3, booting: 3, pending: 2, planned: 2, finished: 1, open: 1, deployed: 1 }
function worst(keys: string[]): string {
  return keys.reduce((b, k) => ((PRI[k] ?? 1) > (PRI[b] ?? 1) ? k : b), keys[0] ?? 'finished')
}
function groupAgg(steps: StepStatus[]): { key: string; running: boolean } {
  if (steps.some((s) => s.status === 'failed')) return { key: 'build_failed', running: false }
  if (steps.some((s) => s.status === 'leased')) return { key: 'running', running: true }
  if (steps.some((s) => s.status === 'blocked')) return { key: 'dependency_blocked', running: false }
  if (steps.length && steps.every((s) => s.status === 'done' || s.status === 'ignored')) return { key: 'finished', running: false }
  return { key: 'pending', running: false }
}
function stepColor(key: string): string {
  return key === 'finished' ? STATUS_COLOR.finished : key === 'running' ? STATUS_COLOR.building : key === 'build_failed' ? STATUS_COLOR.build_failed : key === 'dependency_blocked' ? STATUS_COLOR.dependency_blocked : '#2a2740'
}
function gateFromNode(key: string): string {
  if (key === 'finished') return 'finished'
  if (['deploy_failed', 'build_failed', 'invalid', 'destroy_failed', 'failed'].includes(key)) return 'build_failed'
  if (['dependency_blocked', 'pending', 'planned'].includes(key)) return 'dependency_blocked'
  return 'building'
}
function gateFromGroup(key: string): string {
  return key === 'finished' ? 'finished' : key === 'build_failed' ? 'build_failed' : key === 'running' ? 'building' : 'dependency_blocked'
}

export function DependencyGraph({ repoId, buildId, onClose }: { repoId: string; buildId: string; onClose: () => void }) {
  const { data: build } = useBuild(buildId)
  const { data: topology } = useTopology(buildId)
  const { connected } = useLiveEvents(buildId)
  const [teamId, setTeamId] = useState<string | null>(null)
  const [selected, setSelected] = useState<{ object: DeployedObject; initialTab?: 'info' | 'steps'; focusGroup?: number; teamNumber: number } | null>(null)
  const [hoverId, setHoverId] = useState<string | null>(null)

  const teams = useMemo(() => [...(build?.teams ?? [])].sort((a, b) => a.team_number - b.team_number), [build])
  const team = useMemo(() => teams.find((t) => t.id === teamId) ?? teams[0], [teams, teamId])

  const memberDeps = useMemo(() => {
    const m = new Map<string, string[]>()
    for (const t of topology?.teams ?? []) for (const n of t.networks) for (const mem of n.members) m.set(mem.object_id, mem.depends_on ?? [])
    return m
  }, [topology])

  // Fetch steps once per distinct object name (the per-person duplicates share a
  // template), from the representative instance.
  const stepTargets = useMemo(() => {
    const seen = new Set<string>()
    const out: { objName: string; id: string }[] = []
    for (const o of team?.objects ?? []) {
      if (o.kind === 'network' || seen.has(o.object_name)) continue
      seen.add(o.object_name)
      out.push({ objName: o.object_name, id: o.id })
    }
    return out
  }, [team])
  const stepResults = useQueries({
    queries: stepTargets.map((t) => ({
      queryKey: ['object-steps', buildId, t.id],
      queryFn: () => api.get<StepStatus[]>(`/builds/${buildId}/objects/${t.id}/steps`),
      enabled: !!buildId,
      refetchInterval: 8000,
    })),
  })
  const stepsSig = stepResults.map((r) => r.dataUpdatedAt ?? 0).join('|')

  const model = useMemo(() => {
    const empty = { rows: [] as GNode[], edges: [] as GEdge[], byId: new Map<string, GNode>(), dependents: new Map<string, string[]>(), w: 0, h: 0 }
    if (!team) return empty
    const stepsByName = new Map<string, StepStatus[]>()
    stepTargets.forEach((t, i) => { const d = stepResults[i]?.data; if (d) stepsByName.set(t.objName, d) })

    // Combine identical object names into one node.
    const byObjName = new Map<string, DeployedObject[]>()
    for (const o of team.objects) { const a = byObjName.get(o.object_name) ?? []; a.push(o); byObjName.set(o.object_name, a) }
    const idByName = new Map<string, string>()
    const nodes: GNode[] = []
    const byId = new Map<string, GNode>()
    for (const [objName, objs] of byObjName) {
      const rep = objs[0]
      const key = worst(objs.map((o) => blockedStatusKey(o.status, o.blocked_on)))
      // authored-step groups from the representative's steps
      const raw = stepsByName.get(objName) ?? []
      const groups: SG[] = []
      const byGroup = new Map<number, StepStatus[]>()
      for (const s of raw) {
        const cur = byGroup.get(s.group_index) ?? []; cur.push(s); byGroup.set(s.group_index, cur)
        const last = groups[groups.length - 1]
        if (last && last.index === s.group_index) { if (s.milestone) last.milestone = s.milestone; if (s.wait_for) last.waitFor = Array.from(new Set([...last.waitFor, ...s.wait_for])) }
        else groups.push({ index: s.group_index, label: s.group_label, key: 'pending', running: false, milestone: s.milestone, waitFor: s.wait_for ?? [] })
      }
      for (const g of groups) { const agg = groupAgg(byGroup.get(g.index) ?? []); g.key = agg.key; g.running = agg.running }
      const node: GNode = {
        id: rep.id, name: rep.as_name ?? rep.object_name, objName, kind: rep.kind, net: rep.network_name ?? '', key,
        count: objs.length, obj: rep, groups, deps: [], depth: 0, y: 0, h: 0, cy: 0, x: 0, lineX: 0, src: false, i: 0,
      }
      idByName.set(objName, rep.id)
      nodes.push(node); byId.set(rep.id, node)
    }

    // Resolve dependencies (network + depends_on + wait_for milestones), de-duping
    // a whole-host dep when a wait_for to the same source supersedes it.
    for (const n of nodes) {
      const deps: Dep[] = []
      if (n.net && idByName.has(n.net)) deps.push({ on: idByName.get(n.net)! })
      for (const dn of memberDeps.get(n.obj.id) ?? []) { const id = idByName.get(dn); if (id) deps.push({ on: id }) }
      const waitSrc = new Set<string>()
      for (const g of n.groups) for (const wf of g.waitFor) {
        const [srcName, ms] = wf.split(' · ')
        const srcId = idByName.get((srcName ?? '').trim()); if (!srcId) continue
        const src = byId.get(srcId); const msIdx = src ? src.groups.findIndex((x) => x.milestone === (ms ?? '').trim()) : -1
        deps.push({ on: srcId, milestone: (ms ?? '').trim(), msIdx: msIdx >= 0 ? msIdx : undefined })
        waitSrc.add(srcId)
      }
      // keep milestone deps + whole-host deps not already covered by a wait_for; unique by (on,milestone)
      const seen = new Set<string>()
      n.deps = deps.filter((d) => {
        if (!d.milestone && waitSrc.has(d.on)) return false
        const k = d.on + '|' + (d.milestone ?? ''); if (seen.has(k)) return false; seen.add(k); return true
      })
    }

    // Depth (longest dep path), for ordering members within a network.
    const dmemo = new Map<string, number>()
    const depth = (id: string, seen: Set<string>): number => {
      if (dmemo.has(id)) return dmemo.get(id)!
      const n = byId.get(id); if (!n || !n.deps.length || seen.has(id)) { dmemo.set(id, 0); return 0 }
      seen.add(id); let d = 0; for (const dp of n.deps) d = Math.max(d, depth(dp.on, seen) + 1); seen.delete(id); dmemo.set(id, d); return d
    }
    for (const n of nodes) n.depth = depth(n.id, new Set())

    // Order: grouped by network (network heads its section, members by depth),
    // networks alphabetical so shared infra (corp/domain) comes before dependents.
    const rows: GNode[] = []
    const placed = new Set<string>()
    const nets = nodes.filter((n) => n.kind === 'network').sort((a, b) => a.objName.localeCompare(b.objName))
    for (const net of nets) {
      rows.push(net); placed.add(net.id)
      for (const m of nodes.filter((n) => n.kind !== 'network' && n.net === net.objName).sort((a, b) => a.depth - b.depth || a.name.localeCompare(b.name))) { rows.push(m); placed.add(m.id) }
    }
    for (const r of nodes.filter((n) => !placed.has(n.id)).sort((a, b) => a.depth - b.depth || a.name.localeCompare(b.name))) rows.push(r)
    rows.forEach((n, i) => { n.i = i })

    // Vertical positions (deterministic).
    let y = TOP_PAD
    for (const n of rows) { n.y = y; n.h = n.kind === 'network' || !n.groups.length ? NET_H : HOST_H; n.cy = y + BAR; y += n.h + ROW_GAP }
    const h = y + 6

    // Children + X pass: each node sits right of every line still passing its row.
    const children = new Map<string, { node: GNode; dep: Dep }[]>()
    for (const n of rows) for (const d of n.deps) { const a = children.get(d.on) ?? []; a.push({ node: n, dep: d }); children.set(d.on, a) }
    const active: { x: number; lastRow: number }[] = []
    for (const n of rows) {
      const kids = children.get(n.id)
      for (let j = active.length - 1; j >= 0; j--) if (active[j].lastRow < n.i) active.splice(j, 1)
      n.x = active.length ? Math.max(...active.map((a) => a.x)) + NODE_GAP : BASE_X
      if (kids && kids.length) { n.src = true; n.lineX = n.x + LINE_OFF; active.push({ x: n.lineX, lastRow: Math.max(...kids.map((c) => c.node.i)) }) }
    }
    const w = Math.max(...rows.map((n) => n.x + CARD_W), CARD_W) + 20

    // Edges.
    const edges: GEdge[] = []
    for (const n of rows) for (const d of n.deps) {
      const src = byId.get(d.on); if (!src) continue
      const gate = d.milestone && d.msIdx != null ? gateFromGroup(src.groups[d.msIdx]?.key ?? 'pending') : gateFromNode(src.key)
      edges.push({ from: src.id, to: n.id, gate, msIdx: d.msIdx })
    }
    const dependents = new Map<string, string[]>()
    for (const n of rows) dependents.set(n.id, [])
    for (const n of rows) for (const d of n.deps) dependents.get(d.on)?.push(n.id)

    return { rows, edges, byId, dependents, w, h }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [team, memberDeps, stepsSig, stepTargets])

  useEffect(() => {
    function onKey(e: KeyboardEvent) { if (e.key !== 'Escape') return; if (selected) setSelected(null); else onClose() }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [selected, onClose])

  const tipRef = useRef<HTMLDivElement>(null)
  function showTip(e: React.MouseEvent, html: string) {
    const t = tipRef.current; if (!t) return
    t.innerHTML = html; t.style.opacity = '1'
    t.style.left = Math.min(e.clientX + 14, window.innerWidth - t.offsetWidth - 14) + 'px'
    t.style.top = e.clientY + 16 + 'px'
  }
  const hideTip = () => { if (tipRef.current) tipRef.current.style.opacity = '0' }

  const related = useMemo(() => {
    if (!hoverId) return null
    const s = new Set<string>([hoverId])
    model.byId.get(hoverId)?.deps.forEach((d) => s.add(d.on))
    model.dependents.get(hoverId)?.forEach((d) => s.add(d))
    return s
  }, [hoverId, model])

  function nodeTip(n: GNode) {
    const waiting = n.obj.blocked_on ?? []
    const deps = n.deps.map((d) => (model.byId.get(d.on)?.name ?? '') + (d.milestone ? ' · ' + d.milestone : '')).filter(Boolean)
    return `<b>${n.name}${n.count > 1 ? ' ×' + n.count : ''}</b> · ${n.kind}`
      + `<div class="r">state <b>${statusLabel(n.key)}</b></div>`
      + `<div class="r">depends on <b>${deps.length ? deps.join(', ') : '—'}</b></div>`
      + (waiting.length ? `<div class="r">waiting on <b>${waiting.join(', ')}</b></div>` : '')
      + (n.count > 1 ? `<div class="r"><b>${n.count}</b> identical instances</div>` : '')
  }
  function stepTip(n: GNode, g: SG) {
    return `<b>${n.name}</b> · ${g.label ?? 'step ' + g.index}`
      + `<div class="r">state <b>${statusLabel(g.key === 'running' ? 'building' : g.key)}</b></div>`
      + (g.milestone ? `<div class="r">milestone <b>${g.milestone}</b> · others wait here</div>` : '')
      + (g.waitFor.length ? `<div class="r">waits for <b>${g.waitFor.join(', ')}</b></div>` : '')
  }

  const empty = !team || model.rows.length === 0

  return (
    <>
      <div className="fixed inset-0 z-50 flex flex-col" style={{ background: BOARD.bg, color: BOARD.fg }}>
        <header className="flex shrink-0 flex-wrap items-center gap-x-4 gap-y-2 px-4 py-2.5" style={{ borderBottom: `1px solid ${BOARD.line}` }}>
          <div className="flex items-center gap-2.5">
            <span className={cn('h-2.5 w-2.5 rounded-full', connected && 'animate-pulse')} style={{ background: connected ? STATUS_COLOR.finished : BOARD.faint }} title={connected ? 'Live' : 'Reconnecting…'} />
            <span className="text-base font-semibold tracking-tight">{build?.environment_name ?? 'Environment'}</span>
            <span className="text-sm" style={{ color: BOARD.muted }}>· dependency graph</span>
          </div>
          <select value={team?.id ?? ''} onChange={(e) => setTeamId(e.target.value)} className="rounded-token px-2.5 py-1.5 text-sm" style={{ background: BOARD.panel, border: `1px solid ${BOARD.line2}`, color: BOARD.fg }}>
            {teams.map((t) => (<option key={t.id} value={t.id}>Team {t.team_number}</option>))}
          </select>
          <button onClick={onClose} aria-label="Close dependency graph" className="ml-auto flex h-8 w-8 items-center justify-center rounded-token" style={{ color: BOARD.muted }}><X size={18} /></button>
        </header>

        {empty ? (
          <div className="flex flex-1 items-center justify-center" style={{ color: BOARD.muted }}>
            <EmptyState icon={Boxes} title="Nothing deployed yet" hint="Objects appear here as the build runs." />
          </div>
        ) : (
          <div className="min-h-0 flex-1 overflow-auto" style={{ background: '#0d0a17' }}>
            <div className="relative" style={{ width: model.w, height: model.h }}>
              <svg className="pointer-events-none absolute inset-0 z-[1] overflow-visible" width={model.w} height={model.h}>
                {model.edges.map((e, idx) => {
                  const s = model.byId.get(e.from)!, t = model.byId.get(e.to)!
                  const lx = s.x + LINE_OFF, sy = s.y + s.h, col = statusColor(e.gate), on = !hoverId || e.from === hoverId || e.to === hoverId
                  let top = sy
                  const parts: React.ReactNode[] = []
                  if (e.msIdx != null) {
                    const mx = s.x + stepCenterX(s.groups.length, e.msIdx)
                    parts.push(<path key="m" d={`M${mx} ${sy} C${mx} ${sy + 13} ${lx} ${sy + 1} ${lx} ${sy + 14}`} fill="none" stroke={col} strokeWidth={2} opacity={on ? 0.9 : 0.08} />)
                    top = sy + 14
                  }
                  parts.push(
                    <path key="e" d={`M${lx} ${top} C${lx} ${t.cy} ${lx} ${t.cy} ${t.x} ${t.cy}`} fill="none" stroke={col} strokeWidth={on && hoverId ? 2.6 : 2} strokeDasharray={e.gate === 'build_failed' ? '5 3' : undefined} opacity={on ? 0.95 : 0.08} />,
                    <circle key="d" cx={t.x} cy={t.cy} r={2.7} fill={col} opacity={on ? 1 : 0.08} />,
                  )
                  return <g key={idx}>{parts}</g>
                })}
              </svg>
              {model.rows.map((n) => (
                <GraphNode
                  key={n.id}
                  node={n}
                  dim={!!hoverId && !related?.has(n.id)}
                  onEnter={() => setHoverId(n.id)}
                  onLeave={() => { setHoverId(null); hideTip() }}
                  onNodeMove={(e) => showTip(e, nodeTip(n))}
                  onStepMove={(e, g) => showTip(e, stepTip(n, g))}
                  onOpenNode={() => setSelected({ object: n.obj, teamNumber: team!.team_number })}
                  onOpenStep={(g) => setSelected({ object: n.obj, initialTab: 'steps', focusGroup: g.index, teamNumber: team!.team_number })}
                />
              ))}
            </div>
          </div>
        )}

        <div className="flex shrink-0 flex-wrap items-center gap-x-4 gap-y-1.5 px-4 py-2 text-xs" style={{ borderTop: `1px solid ${BOARD.line}`, color: BOARD.muted }}>
          {BOARD_STATUS_ORDER.map((k) => {
            const Icon = statusIcon(k); const ghost = isGhostStatus(k)
            return (
              <span key={k} className="flex items-center gap-1.5">
                <span className="flex h-4 w-4 items-center justify-center rounded" style={ghost ? { border: `1px dashed ${BOARD.line2}` } : { background: statusColor(k) }}><Icon size={10} color="#fff" /></span>
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
          <ObjectLogPanel repoId={repoId} buildId={buildId} objectId={selected.object.id} label={selected.object.as_name ?? selected.object.object_name} kind={selected.object.kind} object={selected.object} teamNumber={selected.teamNumber} initialTab={selected.initialTab} focusGroup={selected.focusGroup} onClose={() => setSelected(null)} />
        </div>
      )}
    </>
  )
}

function GraphNode({ node: n, dim, onEnter, onLeave, onNodeMove, onStepMove, onOpenNode, onOpenStep }: {
  node: GNode; dim: boolean; onEnter: () => void; onLeave: () => void
  onNodeMove: (e: React.MouseEvent) => void; onStepMove: (e: React.MouseEvent, g: SG) => void
  onOpenNode: () => void; onOpenStep: (g: SG) => void
}) {
  const ghost = isGhostStatus(n.key)
  const Icon = statusIcon(n.key)
  const stacked = n.count > 1
  return (
    <div
      onMouseEnter={onEnter}
      onMouseLeave={onLeave}
      className="absolute z-[2] transition-opacity"
      style={{ left: n.x, top: n.y, width: CARD_W, background: BOARD.panel, border: `1px solid ${BOARD.line}`, borderRadius: 9, opacity: dim ? 0.26 : 1, boxShadow: stacked ? `4px 4px 0 0 ${BOARD.panel2}, 4px 4px 0 1px ${BOARD.line}, 8px 8px 0 0 ${BOARD.panel2}, 8px 8px 0 1px ${BOARD.line}` : undefined }}
    >
      <button
        onClick={onOpenNode}
        onMouseMove={onNodeMove}
        className="flex w-full items-center gap-2 px-2.5 py-1.5 text-left text-[13px] font-medium"
        style={ghost ? { background: 'transparent', color: BOARD.muted, border: '1px dashed #352f4d', borderRadius: 9 } : { background: statusColor(n.key), color: '#fff', borderRadius: n.groups.length ? '9px 9px 0 0' : 9 }}
      >
        <Icon size={15} className={statusAnimClass(n.key)} />
        <span className="truncate font-mono">{n.name}</span>
        {stacked && <span className="ml-auto rounded-full px-1.5 text-[10px] font-semibold" style={{ background: 'rgba(0,0,0,.28)' }}>×{n.count}</span>}
        <span className={cn('text-[10px] opacity-80', !stacked && 'ml-auto')}>{n.kind}</span>
      </button>
      {n.groups.length > 0 && (
        <div className="flex gap-0.5 px-2 pb-2 pt-2" style={{ background: BOARD.panel2, borderRadius: '0 0 9px 9px' }}>
          {n.groups.map((g) => (
            <button
              key={g.index}
              onClick={(e) => { e.stopPropagation(); onOpenStep(g) }}
              onMouseMove={(e) => onStepMove(e, g)}
              className={cn('relative h-1.5 min-w-[3px] flex-1 rounded-sm', g.running && 'animate-pulse')}
              style={{ background: stepColor(g.key), boxShadow: g.milestone ? `0 0 0 1.5px ${BOARD.panel2}, 0 0 0 2.5px ${BOARD.fg}` : undefined }}
            >
              {g.milestone && <span className="absolute left-1/2 top-[-8px] h-[7px] w-[7px] -translate-x-1/2 rounded-full" style={{ background: stepColor(g.key), border: `1.5px solid ${BOARD.fg}` }} />}
            </button>
          ))}
        </div>
      )}
    </div>
  )
}
