import { useParams } from '@tanstack/react-router'
import { useState } from 'react'
import { Globe, Check, Copy, FileJson, ArrowUp, ArrowDown, ArrowUpDown } from 'lucide-react'
import { useExternalAccess } from '../api/hooks'
import { EmptyState } from '../components/EmptyState'
import { ScoringExportWizard } from '../components/ScoringExportWizard'
import type { ExternalAccessEntry } from '../api/types'
import { Badge, Button, Card, Spinner, Table, TableScroller, Td, Th } from '../ui'

// "Build → External Access": the realized endpoints a build exposes to the
// outside world -- for each team's hosts, the public address:port to connect to
// for its `public:` ports (RDP and the like). On Incus/MicroCloud that's one
// shared IP with a distinct port per team/host; on AWS a public IP per host.
// Reads GET /builds/{id}/external-access, which the orchestrator fills in as it
// realizes each team's public ports during a deploy.
export function BuildExternalAccess() {
  const { buildId } = useParams({ from: '/repos/$repoId/builds/$buildId/external-access' })
  const { data: entries, isLoading } = useExternalAccess(buildId)
  const [exporting, setExporting] = useState(false)
  // One sort shared across every team's table, so they all order the same way.
  // Defaults to host name ascending (more useful than the API's arrival order).
  const [sortKey, setSortKey] = useState<SortKey>('host')
  const [sortDir, setSortDir] = useState<'asc' | 'desc'>('asc')
  function toggleSort(k: SortKey) {
    if (k === sortKey) setSortDir((d) => (d === 'asc' ? 'desc' : 'asc'))
    else {
      setSortKey(k)
      setSortDir('asc')
    }
  }

  if (isLoading)
    return (
      <div className="flex items-center gap-2 text-sm text-fg-muted">
        <Spinner /> Loading…
      </div>
    )

  if (!entries || entries.length === 0)
    return (
      <EmptyState
        icon={Globe}
        title="No external access"
        hint="Hosts with a `public:` block (e.g. RDP) get an external endpoint here once the build deploys. None are public yet, or none are deployed."
      />
    )

  // Group by team, teams in order.
  const byTeam = new Map<number, ExternalAccessEntry[]>()
  for (const e of entries) {
    const list = byTeam.get(e.team_number) ?? []
    list.push(e)
    byTeam.set(e.team_number, list)
  }
  const teams = [...byTeam.keys()].sort((a, b) => a - b)

  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-start gap-4">
        <p className="flex-1 text-sm text-fg-muted">
          The address and port to connect to each host from outside the environment. Access follows each team's schedule — a closed team
          is unreachable even while these endpoints are listed.
        </p>
        <Button variant="brand" size="sm" onClick={() => setExporting(true)}>
          <FileJson size={14} /> Export for Scoring
        </Button>
      </div>
      <ScoringExportWizard buildId={buildId} open={exporting} onClose={() => setExporting(false)} />
      {teams.map((team) => (
        <Card key={team} className="overflow-hidden p-0">
          <div className="bg-cptc-violet px-4 py-2 text-sm font-semibold text-cptc-violet-fg">Team {team}</div>
          {/* contain={false}: this page scrolls as a whole, and a containing
              scroller would swallow the wheel while the cursor is over a team. */}
          <TableScroller contain={false}>
            <Table>
              <thead>
                <tr>
                  <SortTh label="Host" k="host" sortKey={sortKey} sortDir={sortDir} onSort={toggleSort} />
                  <SortTh label="Protocol" k="protocol" sortKey={sortKey} sortDir={sortDir} onSort={toggleSort} />
                  <SortTh label="Port" k="port" sortKey={sortKey} sortDir={sortDir} onSort={toggleSort} />
                  <SortTh label="Connect to" k="address" sortKey={sortKey} sortDir={sortDir} onSort={toggleSort} />
                </tr>
              </thead>
              <tbody>
                {[...byTeam.get(team)!].sort((a, b) => compareEntries(a, b, sortKey, sortDir)).map((e) => (
                  <tr key={`${e.deployed_object_id}:${e.protocol}:${e.internal_port}`}>
                    <Td className="font-medium text-fg">{e.as_name || e.object_name}</Td>
                    <Td>
                      <Badge tone="neutral" className="rounded-full py-0 uppercase">
                        {e.protocol}
                      </Badge>
                    </Td>
                    <Td className="font-mono text-xs text-fg-muted">{e.internal_port}</Td>
                    <Td>
                      <ConnectTo address={e.public_address} />
                    </Td>
                  </tr>
                ))}
              </tbody>
            </Table>
          </TableScroller>
        </Card>
      ))}
    </div>
  )
}

type SortKey = 'host' | 'protocol' | 'port' | 'address'

// Compare two endpoints by the active column. Host/address use a numeric-aware
// collator so bff-ws-2 sorts before bff-ws-10 (and 10.0.0.9 before 10.0.0.10);
// port is a plain numeric compare.
function compareEntries(a: ExternalAccessEntry, b: ExternalAccessEntry, key: SortKey, dir: 'asc' | 'desc'): number {
  let r = 0
  switch (key) {
    case 'host':
      r = (a.as_name || a.object_name).localeCompare(b.as_name || b.object_name, undefined, { numeric: true })
      break
    case 'protocol':
      r = a.protocol.localeCompare(b.protocol)
      break
    case 'port':
      r = a.internal_port - b.internal_port
      break
    case 'address':
      r = a.public_address.localeCompare(b.public_address, undefined, { numeric: true })
      break
  }
  return dir === 'asc' ? r : -r
}

// SortTh is a clickable column header: click to sort by it, click again to flip
// direction. The arrow shows the current sort (a faint up/down toggle when the
// column isn't the active one).
function SortTh({ label, k, sortKey, sortDir, onSort }: { label: string; k: SortKey; sortKey: SortKey; sortDir: 'asc' | 'desc'; onSort: (k: SortKey) => void }) {
  const active = sortKey === k
  const Icon = !active ? ArrowUpDown : sortDir === 'asc' ? ArrowUp : ArrowDown
  return (
    <Th aria-sort={active ? (sortDir === 'asc' ? 'ascending' : 'descending') : 'none'}>
      <button type="button" onClick={() => onSort(k)} className="inline-flex select-none items-center gap-1 hover:text-fg">
        {label}
        <Icon size={12} className={active ? 'text-fg-muted' : 'text-fg-subtle/50'} />
      </button>
    </Th>
  )
}

// ConnectTo shows the public address:port mono, with a one-click copy -- what an
// operator actually pastes into an RDP client.
function ConnectTo({ address }: { address: string }) {
  const [copied, setCopied] = useState(false)
  async function copy() {
    try {
      await navigator.clipboard.writeText(address)
      setCopied(true)
      setTimeout(() => setCopied(false), 1500)
    } catch {
      /* clipboard blocked */
    }
  }
  return (
    <span className="inline-flex items-center gap-1.5">
      <span className="font-mono text-xs text-fg">{address}</span>
      <Button variant="ghost" size="icon" onClick={copy} aria-label="Copy address" title="Copy">
        {copied ? <Check size={13} className="text-success" /> : <Copy size={13} />}
      </Button>
    </span>
  )
}
