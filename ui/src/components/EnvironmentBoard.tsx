import { Fragment, useEffect, useMemo, useState } from 'react'
import { X, Boxes, Network as NetworkIcon } from 'lucide-react'
import { useBuild, useLiveEvents, useTopology } from '../api/hooks'
import type { DeployedObject, ObjectKind } from '../api/types'
import { statusIcon, statusLabel, statusAnimClass, blockedStatusKey } from './StatusBadge'
import { ObjectLogPanel } from './ObjectLogPanel'
import { EmptyState } from './EmptyState'
import { STATUS_COLOR, isGhostStatus as isGhost, BOARD_STATUS_ORDER as LEGEND_ORDER } from './statusPalette'
import { cn } from '../ui'

// The Status Matrix: the whole build as one wall-display board -- a row per team,
// a column per network and host/container, each cell a single status colour +
// icon. Every team gets the same topology, so columns line up: a bad column is
// one host failing across teams, a bad row is one team behind. Full-screen
// takeover meant for a screen at the front of the room, so it commits to a dark
// look regardless of the app theme. Icons + animations come from StatusBadge (one
// source of truth); the colours are the board's own fixed palette. Updates live
// off useBuild's poll. Clicking a tile opens the same host panel the Hosts page
// uses.

type Col = { kind: 'network' | 'host'; net: string; name: string }
type Group = { net: string; cidr?: string; cols: Col[] }

function bucketOf(key: string): 'done' | 'prog' | 'fail' | 'wait' | 'gone' {
  if (key === 'finished' || key === 'open' || key === 'deployed') return 'done'
  if (['deploy_failed', 'build_failed', 'invalid', 'destroy_failed', 'failed'].includes(key)) return 'fail'
  if (key === 'pending' || key === 'planned') return 'wait'
  if (['torn_down', 'destroyed', 'purged', 'discarded'].includes(key)) return 'gone'
  return 'prog'
}

export function EnvironmentBoard({ repoId, buildId, onClose }: { repoId: string; buildId: string; onClose: () => void }) {
  const { data: build } = useBuild(buildId)
  const { data: topology } = useTopology(buildId)
  const { connected } = useLiveEvents(buildId)
  const [selected, setSelected] = useState<{ id: string; label: string; kind: ObjectKind; object: DeployedObject; teamNumber: number } | null>(null)
  const [now, setNow] = useState(() => new Date())

  useEffect(() => {
    const t = setInterval(() => setNow(new Date()), 1000)
    return () => clearInterval(t)
  }, [])
  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if (e.key !== 'Escape') return
      if (selected) setSelected(null)
      else onClose()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [selected, onClose])

  const teams = useMemo(() => [...(build?.teams ?? [])].sort((a, b) => a.team_number - b.team_number), [build])

  const cidrByNet = useMemo(() => {
    const m = new Map<string, string>()
    for (const t of topology?.teams ?? []) for (const n of t.networks) if (n.name && n.cidr && !m.has(n.name)) m.set(n.name, n.cidr)
    return m
  }, [topology])

  const groups = useMemo<Group[]>(() => {
    const hostsByNet = new Map<string, Set<string>>()
    const netNames = new Set<string>()
    for (const team of teams) {
      for (const o of team.objects) {
        if (o.kind === 'network') {
          netNames.add(o.object_name)
          if (!hostsByNet.has(o.object_name)) hostsByNet.set(o.object_name, new Set())
        } else {
          const net = o.network_name ?? ''
          netNames.add(net)
          const set = hostsByNet.get(net) ?? new Set<string>()
          set.add(o.object_name)
          hostsByNet.set(net, set)
        }
      }
    }
    const order = Array.from(netNames).sort((a, b) => (a === '' ? 1 : b === '' ? -1 : a.localeCompare(b)))
    return order
      .map((net) => {
        const cols: Col[] = []
        if (net !== '') cols.push({ kind: 'network', net, name: net })
        for (const h of Array.from(hostsByNet.get(net) ?? []).sort()) cols.push({ kind: 'host', net, name: h })
        return { net, cidr: cidrByNet.get(net), cols }
      })
      .filter((g) => g.cols.length > 0)
  }, [teams, cidrByNet])

  const objByTeam = useMemo(() => {
    const m = new Map<string, Map<string, DeployedObject>>()
    for (const team of teams) {
      const inner = new Map<string, DeployedObject>()
      for (const o of team.objects) inner.set(o.object_name, o)
      m.set(team.id, inner)
    }
    return m
  }, [teams])

  const counts = useMemo(() => {
    const c = { done: 0, prog: 0, fail: 0, wait: 0, gone: 0 }
    for (const team of teams) for (const o of team.objects) c[bucketOf(blockedStatusKey(o.status, o.blocked_on))]++
    return c
  }, [teams])

  const NC = groups.reduce((n, g) => n + g.cols.length, 0)
  const NT = teams.length

  return (
    <>
      <div className="fixed inset-0 z-50 flex flex-col bg-[#0b0714] text-[#e9e6f3]">
        <header className="flex shrink-0 flex-wrap items-center gap-x-4 gap-y-2 border-b border-[#241f33] px-4 py-2.5">
          <div className="flex items-center gap-2.5">
            <span className={cn('h-2.5 w-2.5 rounded-full', connected ? 'animate-pulse bg-[#5f941c]' : 'bg-[#6c6683]')} title={connected ? 'Live' : 'Reconnecting…'} />
            <span className="text-base font-semibold tracking-tight">{build?.environment_name ?? 'Environment'}</span>
            <span className="text-sm text-[#9089a6]">· status matrix</span>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <Chip color="#5f941c" label={`${counts.done} complete`} />
            <Chip color="#7f77dd" label={`${counts.prog} in progress`} />
            <Chip color="#e04a49" label={`${counts.fail} failed`} />
            <Chip color="#6c6683" label={`${counts.wait} pending`} />
          </div>
          <div className="ml-auto flex items-center gap-3">
            <span className="font-mono text-sm tabular-nums text-[#9089a6]">{now.toLocaleTimeString('en-GB')}</span>
            <button
              onClick={onClose}
              aria-label="Close status matrix"
              className="flex h-8 w-8 items-center justify-center rounded-token text-[#9089a6] hover:bg-[#17122a] hover:text-[#e9e6f3]"
            >
              <X size={18} />
            </button>
          </div>
        </header>

        {NT === 0 || NC === 0 ? (
          <div className="flex flex-1 items-center justify-center text-[#9089a6]">
            <EmptyState icon={Boxes} title="Nothing deployed yet" hint="Hosts and networks appear here as the build runs." />
          </div>
        ) : (
          <div className="min-h-0 flex-1 overflow-auto p-3">
            <div
              className="grid h-full min-w-[56rem] gap-1"
              style={{
                gridTemplateColumns: `clamp(2.2rem,3vw,3.4rem) repeat(${NC}, minmax(0,1fr))`,
                gridTemplateRows: `auto auto repeat(${NT}, minmax(0,1fr))`,
              }}
            >
              {/* Row 1 — network group headers */}
              <div />
              {groups.map((g, gi) => (
                <div
                  key={`h-${g.net || '—'}`}
                  style={{ gridColumn: `span ${g.cols.length}` }}
                  className={cn('flex items-baseline gap-2 px-1 pb-1 text-sm font-medium', gi > 0 && 'border-l border-[#241f33] pl-2')}
                >
                  <span className="truncate">{g.net || 'no network'}</span>
                  {g.cidr && <span className="font-mono text-[10px] text-[#6c6683]">{g.cidr}</span>}
                </div>
              ))}

              {/* Row 2 — host sub-headers */}
              <div />
              {groups.map((g, gi) =>
                g.cols.map((c, ci) => (
                  <div
                    key={`s-${g.net}-${c.name}`}
                    title={c.name}
                    className={cn('self-end truncate pb-1 text-center font-mono text-[11px] text-[#9089a6]', gi > 0 && ci === 0 && 'border-l border-[#241f33]')}
                  >
                    {c.kind === 'network' ? <NetworkIcon size={12} className="mx-auto text-[#6c6683]" /> : c.name}
                  </div>
                )),
              )}

              {/* Rows 3+ — one per team */}
              {teams.map((team) => (
                <Fragment key={team.id}>
                  <div className="flex items-center justify-center text-sm font-semibold text-[#c3bdd9]">T{team.team_number}</div>
                  {groups.map((g, gi) =>
                    g.cols.map((c, ci) => {
                      const obj = objByTeam.get(team.id)?.get(c.name)
                      const sep = gi > 0 && ci === 0
                      if (!obj) {
                        return <div key={`${team.id}-${g.net}-${c.name}`} className={cn('rounded-md border border-dashed border-[#241f33]', sep && 'ml-0.5')} />
                      }
                      const key = blockedStatusKey(obj.status, obj.blocked_on)
                      const ghost = isGhost(key)
                      const Icon = statusIcon(key)
                      return (
                        <button
                          key={`${team.id}-${g.net}-${c.name}`}
                          onClick={() => setSelected({ id: obj.id, label: obj.as_name ?? obj.object_name, kind: obj.kind, object: obj, teamNumber: team.team_number })}
                          title={`T${team.team_number} · ${g.net || 'no network'} / ${c.name} — ${statusLabel(key)}`}
                          style={ghost ? undefined : { background: STATUS_COLOR[key] ?? '#5f5e5a' }}
                          className={cn(
                            'flex items-center justify-center rounded-md outline-2 outline-transparent transition hover:outline-white/70 focus-visible:outline-white',
                            '[&>svg]:h-[clamp(0.7rem,2vh,1.35rem)] [&>svg]:w-[clamp(0.7rem,2vh,1.35rem)]',
                            ghost ? 'border border-dashed border-[#352f4d] text-[#5b5570]' : 'text-white/90',
                            sep && 'ml-0.5',
                          )}
                        >
                          <Icon className={statusAnimClass(key)} />
                        </button>
                      )
                    }),
                  )}
                </Fragment>
              ))}
            </div>
          </div>
        )}

        {/* Legend */}
        <div className="flex shrink-0 flex-wrap gap-x-4 gap-y-1.5 border-t border-[#241f33] px-4 py-2 text-xs text-[#9089a6]">
          {LEGEND_ORDER.map((k) => {
            const Icon = statusIcon(k)
            const ghost = isGhost(k)
            return (
              <span key={k} className="flex items-center gap-1.5">
                <span
                  style={ghost ? undefined : { background: STATUS_COLOR[k] ?? '#5f5e5a' }}
                  className={cn('flex h-4 w-4 items-center justify-center rounded [&>svg]:h-2.5 [&>svg]:w-2.5', ghost ? 'border border-dashed border-[#352f4d] text-[#5b5570]' : 'text-white/90')}
                >
                  <Icon />
                </span>
                {statusLabel(k)}
              </span>
            )
          })}
        </div>
      </div>

      {/* Host detail drawer, above the board (its own z-60 stacking context so the
          InspectorPanel's z-30/z-40 overlay layers sit over the z-50 board). */}
      {selected && (
        <div className="relative z-[60]">
          <ObjectLogPanel
            repoId={repoId}
            buildId={buildId}
            objectId={selected.id}
            label={selected.label}
            kind={selected.kind}
            object={selected.object}
            teamNumber={selected.teamNumber}
            onClose={() => setSelected(null)}
          />
        </div>
      )}
    </>
  )
}

function Chip({ color, label }: { color: string; label: string }) {
  return (
    <span className="inline-flex items-center gap-1.5 rounded-full border border-[#2e2947] bg-[#17122a] px-2.5 py-0.5 text-xs tabular-nums text-[#e9e6f3]">
      <span className="h-2 w-2 rounded-full" style={{ background: color }} />
      {label}
    </span>
  )
}
