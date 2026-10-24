import { useParams } from '@tanstack/react-router'
import { useMemo, useState } from 'react'
import { ArrowDown, ArrowUp, Download, Search, ShieldAlert } from 'lucide-react'
import { useFindings } from '../api/hooks'
import { API_BASE } from '../api/client'
import { EmptyState } from '../components/EmptyState'
import type { FindingInstance } from '../api/types'
import { Badge, buttonClass, cn, Spinner, Table, TableScroller, Td, Th, type BadgeProps } from '../ui'

// A 1–5 scale to the design system's status tones -- higher number, hotter
// colour. Used for both severity and difficulty so the two columns read the
// same way (the badge is coloured by its value).
const SCALE_TONE: Record<number, BadgeProps['tone']> = {
  5: 'danger',
  4: 'orange',
  3: 'warning',
  2: 'info',
  1: 'neutral',
}

type SortKey = 'severity' | 'difficulty' | 'team' | 'object' | 'source'

function objectName(f: FindingInstance): string {
  return f.as_name ?? f.object_name
}
function sourceLabel(f: FindingInstance): string {
  return f.source === 'direct' ? 'Config files' : `Inherited: ${f.source.replace('script:', '')}`
}

// "Build → Findings: findings instantiated in this build ... the object each
// came from and whether it was direct or inherited from a script." Severity
// and difficulty are equal-weight columns here (not a grouping header), so the
// table sorts, filters, and searches on either.
export function BuildFindings() {
  const { buildId } = useParams({ from: '/repos/$repoId/builds/$buildId/findings' })
  const { data: findings, isLoading } = useFindings(buildId)
  const [search, setSearch] = useState('')
  const [sevFilter, setSevFilter] = useState<Set<number>>(new Set())
  const [diffFilter, setDiffFilter] = useState<Set<number>>(new Set())
  const [sortKey, setSortKey] = useState<SortKey>('severity')
  const [sortDir, setSortDir] = useState<'asc' | 'desc'>('desc')

  function toggleSet(setter: typeof setSevFilter, v: number) {
    setter((prev) => {
      const next = new Set(prev)
      next.has(v) ? next.delete(v) : next.add(v)
      return next
    })
  }
  function sortBy(key: SortKey) {
    if (key === sortKey) setSortDir((d) => (d === 'asc' ? 'desc' : 'asc'))
    else {
      setSortKey(key)
      // Severity/difficulty default worst-first; text columns default A→Z.
      setSortDir(key === 'severity' || key === 'difficulty' ? 'desc' : 'asc')
    }
  }

  const rows = useMemo(() => {
    const q = search.trim().toLowerCase()
    let out = (findings ?? []).filter((f) => {
      if (sevFilter.size > 0 && !sevFilter.has(f.severity)) return false
      if (diffFilter.size > 0 && !diffFilter.has(f.difficulty)) return false
      if (q) {
        const hay = [f.description, objectName(f), f.object_kind, sourceLabel(f), f.team_number ? `team ${f.team_number}` : 'build'].join(' ').toLowerCase()
        if (!hay.includes(q)) return false
      }
      return true
    })
    const dir = sortDir === 'asc' ? 1 : -1
    out = [...out].sort((a, b) => {
      let cmp = 0
      switch (sortKey) {
        case 'severity':
          cmp = a.severity - b.severity || a.difficulty - b.difficulty
          break
        case 'difficulty':
          cmp = a.difficulty - b.difficulty || a.severity - b.severity
          break
        case 'team':
          cmp = (a.team_number ?? 0) - (b.team_number ?? 0)
          break
        case 'object':
          cmp = objectName(a).localeCompare(objectName(b))
          break
        case 'source':
          cmp = sourceLabel(a).localeCompare(sourceLabel(b))
          break
      }
      return cmp * dir || a.description.localeCompare(b.description)
    })
    return out
  }, [findings, search, sevFilter, diffFilter, sortKey, sortDir])

  if (isLoading)
    return (
      <div className="flex items-center gap-2 text-sm text-fg-muted">
        <Spinner /> Loading…
      </div>
    )

  if (!findings || findings.length === 0) {
    return <EmptyState icon={ShieldAlert} title="No findings in this build" hint="Findings are authored on any object in content and appear here once deployed." />
  }

  return (
    <div>
      <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
        <div className="flex flex-wrap items-center gap-3">
          <div className="relative">
            <Search size={13} className="pointer-events-none absolute left-2 top-1/2 -translate-y-1/2 text-fg-subtle" />
            <input
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Search findings…"
              className="h-8 w-56 rounded-token border border-border bg-surface-raised pl-7 pr-2 text-sm text-fg placeholder:text-fg-subtle outline-none focus-visible:border-accent focus-visible:outline-2 focus-visible:outline-offset-0 focus-visible:outline-accent-soft"
            />
          </div>
          <FilterChips label="Severity" values={[5, 4, 3, 2, 1]} active={sevFilter} onToggle={(v) => toggleSet(setSevFilter, v)} />
          <FilterChips label="Difficulty" values={[1, 2, 3, 4, 5]} active={diffFilter} onToggle={(v) => toggleSet(setDiffFilter, v)} />
        </div>
        <a href={`${API_BASE}/builds/${buildId}/findings?format=csv`} className={cn(buttonClass({ variant: 'ghost', size: 'sm' }), 'text-fg-muted')}>
          <Download size={14} /> Export CSV
        </a>
      </div>

      <div className="mb-2 text-xs text-fg-muted">
        {rows.length} of {findings.length} finding{findings.length === 1 ? '' : 's'}
      </div>

      <TableScroller contain={false}>
        <Table>
          <thead>
            <tr>
              <SortHeader label="Severity" col="severity" sortKey={sortKey} sortDir={sortDir} onSort={sortBy} className="w-24" />
              <SortHeader label="Difficulty" col="difficulty" sortKey={sortKey} sortDir={sortDir} onSort={sortBy} className="w-24" />
              <SortHeader label="Object" col="object" sortKey={sortKey} sortDir={sortDir} onSort={sortBy} className="w-44" />
              <SortHeader label="Team" col="team" sortKey={sortKey} sortDir={sortDir} onSort={sortBy} className="w-20" />
              <SortHeader label="Source" col="source" sortKey={sortKey} sortDir={sortDir} onSort={sortBy} className="w-32" />
              <Th>Description</Th>
            </tr>
          </thead>
          <tbody>
            {rows.map((f, i) => (
              <tr key={i}>
                <Td>
                  <Badge tone={SCALE_TONE[f.severity]}>{f.severity}</Badge>
                </Td>
                <Td>
                  <Badge tone={SCALE_TONE[f.difficulty]}>{f.difficulty}</Badge>
                </Td>
                <Td className="font-medium text-fg">
                  {objectName(f)}
                  <span className="ml-1 text-xs text-fg-muted">({f.object_kind})</span>
                </Td>
                <Td className="text-fg-muted">{f.team_number ? `Team ${f.team_number}` : 'Build'}</Td>
                <Td className="text-xs text-fg-muted">{sourceLabel(f)}</Td>
                <Td className="text-fg">{f.description}</Td>
              </tr>
            ))}
          </tbody>
        </Table>
      </TableScroller>
      {rows.length === 0 && <div className="mt-3 text-sm text-fg-muted">No findings match the current search or filters.</div>}
    </div>
  )
}

function FilterChips({ label, values, active, onToggle }: { label: string; values: number[]; active: Set<number>; onToggle: (v: number) => void }) {
  return (
    <div className="flex items-center gap-1">
      <span className="mr-0.5 text-xs font-medium text-fg-muted">{label}</span>
      {values.map((v) => (
        <button
          key={v}
          onClick={() => onToggle(v)}
          className={cn(
            'rounded-token-sm px-1.5 py-1 text-xs font-medium',
            active.has(v) ? 'bg-accent text-fg-inverted' : 'bg-surface-sunken text-fg-muted hover:text-fg',
          )}
        >
          {v}
        </button>
      ))}
    </div>
  )
}

function SortHeader({
  label,
  col,
  sortKey,
  sortDir,
  onSort,
  className,
}: {
  label: string
  col: SortKey
  sortKey: SortKey
  sortDir: 'asc' | 'desc'
  onSort: (k: SortKey) => void
  className?: string
}) {
  const active = sortKey === col
  return (
    <Th className={className}>
      <button onClick={() => onSort(col)} className={cn('inline-flex items-center gap-1 hover:text-fg', active ? 'text-fg' : '')}>
        {label}
        {active && (sortDir === 'asc' ? <ArrowUp size={12} /> : <ArrowDown size={12} />)}
      </button>
    </Th>
  )
}
