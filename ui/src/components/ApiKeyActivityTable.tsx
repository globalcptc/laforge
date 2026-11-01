import type { McpActivity } from '../api/types'
import { Badge, Table, TableScroller, Td, Th } from '../ui'
import { useTimeFormat } from '../lib/time'

const RESULT_TONE = { ok: 'success', denied: 'warning', error: 'danger', rate_limited: 'orange' } as const
const RESULT_LABEL = { ok: 'OK', denied: 'Denied', error: 'Error', rate_limited: 'Rate Limited' } as const

// The tool's arguments in one short line, e.g. "action=reboot force=true team=2"
// (the build has its own column).
function summarizeArgs(args: McpActivity['args']): string {
  if (!args) return ''
  return Object.entries(args)
    .filter(([k, v]) => k !== 'build_id' && v !== '' && v !== null && v !== undefined && v !== false && !(Array.isArray(v) && v.length === 0))
    .map(([k, v]) => `${k}=${typeof v === 'string' ? v : JSON.stringify(v)}`)
    .join(' ')
}

// Recent MCP tool calls made with API keys: the API Keys page's Recent
// Activity (your own keys) and the admin page's (everyone's, with Owner).
export function ApiKeyActivityTable({ rows, showOwner = false }: { rows: McpActivity[]; showOwner?: boolean }) {
  const fmt = useTimeFormat()
  return (
    <TableScroller contain={false}>
      <Table>
        <thead>
          <tr>
            <Th>Time</Th>
            {showOwner && <Th>Owner</Th>}
            <Th>Key</Th>
            <Th>Tool</Th>
            <Th>Result</Th>
            <Th>Build</Th>
            <Th>Arguments</Th>
            <Th>Took</Th>
            <Th>From</Th>
          </tr>
        </thead>
        <tbody>
          {rows.map((a) => {
            const args = summarizeArgs(a.args)
            return (
              <tr key={a.id}>
                <Td className="whitespace-nowrap text-fg-muted">{fmt.dateTime(a.created_at)}</Td>
                {showOwner && <Td className="whitespace-nowrap font-mono text-fg">@{a.owner_login}</Td>}
                <Td className="whitespace-nowrap text-fg" title={`${a.key_prefix}…`}>
                  {a.key_name}
                </Td>
                <Td className="whitespace-nowrap">
                  {a.tool === 'power' ? <Badge tone="brand">power</Badge> : <span className="font-mono text-xs text-fg">{a.tool}</span>}
                </Td>
                <Td className="whitespace-nowrap">
                  <Badge tone={RESULT_TONE[a.status]}>{RESULT_LABEL[a.status]}</Badge>
                  {a.error && (
                    <div className="mt-0.5 max-w-[20rem] truncate text-xs text-fg-subtle" title={a.error}>
                      {a.error}
                    </div>
                  )}
                </Td>
                <Td className="whitespace-nowrap font-mono text-xs text-fg-muted" title={a.build_id ?? undefined}>
                  {a.build_id ? a.build_id.slice(0, 8) : '—'}
                </Td>
                <Td className="max-w-[22rem] text-fg-muted">
                  <div className="truncate font-mono text-xs" title={args}>
                    {args || '—'}
                  </div>
                </Td>
                <Td className="whitespace-nowrap text-fg-muted">{a.duration_ms} ms</Td>
                <Td className="whitespace-nowrap font-mono text-xs text-fg-muted" title={a.user_agent}>
                  {a.remote_addr || '—'}
                </Td>
              </tr>
            )
          })}
        </tbody>
      </Table>
    </TableScroller>
  )
}
