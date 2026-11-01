import { useMemo } from 'react'
import { Ban } from 'lucide-react'
import { useRepositories } from '../api/hooks'
import { ApiError } from '../api/client'
import type { ApiKey } from '../api/types'
import { Badge, Button, Table, TableScroller, Td, Th, useToast } from '../ui'
import { useTimeFormat } from '../lib/time'

const STATUS_TONE = { active: 'success', expired: 'neutral', revoked: 'danger' } as const
const STATUS_LABEL = { active: 'Active', expired: 'Expired', revoked: 'Revoked' } as const

// The API keys table, shared by a person's own API Keys page and the admin
// page (which adds an Owner column).
export function ApiKeysTable({
  keys,
  showOwner = false,
  revoke,
  revoking,
}: {
  keys: ApiKey[]
  showOwner?: boolean
  revoke: (id: string) => Promise<unknown>
  revoking: boolean
}) {
  const { data: repos } = useRepositories()
  const fmt = useTimeFormat()
  const toast = useToast()
  const repoNames = useMemo(() => new Map((repos ?? []).map((r) => [r.id, `${r.github_owner}/${r.github_repo}`])), [repos])

  async function onRevoke(k: ApiKey) {
    const who = showOwner && k.owner_login ? ` belonging to @${k.owner_login}` : ''
    if (!confirm(`Revoke "${k.name}"${who}? Anything using it stops working immediately. This can't be undone.`)) return
    try {
      await revoke(k.id)
      toast({ title: 'Key Revoked', description: k.name, tone: 'success' })
    } catch (e) {
      toast({ title: 'Revoke Failed', description: e instanceof ApiError ? e.message : 'error', tone: 'danger' })
    }
  }

  return (
    <TableScroller contain={false}>
      <Table>
        <thead>
          <tr>
            <Th>Name</Th>
            {showOwner && <Th>Owner</Th>}
            <Th>Key</Th>
            <Th>Repositories</Th>
            <Th>Created</Th>
            <Th>Expires</Th>
            <Th>Last Used</Th>
            <Th>Status</Th>
            <Th></Th>
          </tr>
        </thead>
        <tbody>
          {keys.map((k) => (
            <tr key={k.id}>
              <Td className="whitespace-nowrap text-fg">
                {k.name}
                {k.kind === 'app' && <div className="text-xs text-fg-muted">Connected app{k.client_host ? ` · ${k.client_host}` : ''}</div>}
              </Td>
              {showOwner && (
                <Td className="whitespace-nowrap text-fg">
                  <span className="inline-flex items-center gap-2">
                    {k.owner_avatar_url ? <img src={k.owner_avatar_url} alt="" className="size-5 rounded-full" /> : <span className="size-5 rounded-full bg-surface-sunken" />}
                    <span className="font-mono">@{k.owner_login}</span>
                  </span>
                </Td>
              )}
              <Td className="whitespace-nowrap font-mono text-xs text-fg-muted">{k.prefix}…</Td>
              <Td className="text-fg-muted">
                {k.repository_ids.length === 0 ? (
                  'All readable'
                ) : (
                  <span className="flex flex-col">
                    {k.repository_ids.map((id) => (
                      <span key={id} className="font-mono text-xs">
                        {repoNames.get(id) ?? id.slice(0, 8)}
                      </span>
                    ))}
                  </span>
                )}
              </Td>
              <Td className="whitespace-nowrap text-fg-muted">{fmt.date(k.created_at)}</Td>
              <Td className="whitespace-nowrap text-fg-muted">{fmt.date(k.expires_at)}</Td>
              <Td className="whitespace-nowrap text-fg-muted">{k.last_used_at ? fmt.dateTime(k.last_used_at) : 'Never'}</Td>
              <Td>
                <Badge tone={STATUS_TONE[k.status]}>{STATUS_LABEL[k.status]}</Badge>
              </Td>
              <Td className="text-right whitespace-nowrap">
                {k.status === 'active' && (
                  <Button variant="danger" size="sm" onClick={() => onRevoke(k)} disabled={revoking}>
                    <Ban size={12} /> Revoke
                  </Button>
                )}
              </Td>
            </tr>
          ))}
        </tbody>
      </Table>
    </TableScroller>
  )
}
