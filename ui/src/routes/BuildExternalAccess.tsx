import { useParams } from '@tanstack/react-router'
import { useState } from 'react'
import { Globe, Check, Copy } from 'lucide-react'
import { useExternalAccess } from '../api/hooks'
import { EmptyState } from '../components/EmptyState'
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
      <p className="text-sm text-fg-muted">
        The address and port to connect to each host from outside the environment. Access follows each team's schedule — a closed team is
        unreachable even while these endpoints are listed.
      </p>
      {teams.map((team) => (
        <Card key={team} className="p-0">
          <div className="border-b border-border px-4 py-2 text-sm font-semibold text-fg">Team {team}</div>
          <TableScroller>
            <Table>
              <thead>
                <tr>
                  <Th>Host</Th>
                  <Th>Protocol</Th>
                  <Th>Port</Th>
                  <Th>Connect to</Th>
                </tr>
              </thead>
              <tbody>
                {byTeam.get(team)!.map((e) => (
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
