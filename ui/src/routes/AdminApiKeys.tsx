import { useState } from 'react'
import { KeyRound } from 'lucide-react'
import { useAdminApiKeyActivity, useAdminApiKeys, useAdminRevokeApiKey } from '../api/hooks'
import { ApiError } from '../api/client'
import { EmptyState } from '../components/EmptyState'
import { ApiKeyActivityLog } from '../components/ApiKeyActivityLog'
import { ApiKeysTable } from '../components/ApiKeysTable'
import { Card, CardHeader, PageHeader, Spinner } from '../ui'

// Every person's API keys, for instance admins: who has keys out, when each
// was last used, and a Revoke for any of them (the lever when a key leaks).
export function AdminApiKeys() {
  const { data: keys, isLoading, error } = useAdminApiKeys()
  const revoke = useAdminRevokeApiKey()
  const [showInactive, setShowInactive] = useState(false)

  if (error) {
    const forbidden = error instanceof ApiError && error.status === 403
    return (
      <div className="mx-auto max-w-2xl p-6">
        <Card className="p-4 text-sm text-fg">
          {forbidden ? "You're not an instance admin, so you can't see other people's API keys." : error instanceof ApiError ? error.message : 'Failed to load keys.'}
        </Card>
      </div>
    )
  }

  const active = (keys ?? []).filter((k) => k.status === 'active')
  const inactiveCount = (keys ?? []).length - active.length
  const shown = showInactive ? (keys ?? []) : active

  return (
    <>
      <PageHeader title="API Keys" description="Every person's API keys for LaForge's MCP server." />
      <div className="page-body">
        <Card>
          <CardHeader title="All Keys" description="Revoking a key stops it working immediately." />
          <div className="p-4 pt-2">
            {isLoading && (
              <div className="flex items-center gap-2 text-sm text-fg-muted">
                <Spinner /> Loading…
              </div>
            )}
            {!isLoading && shown.length === 0 && <EmptyState icon={KeyRound} title="No Active Keys" hint="Nobody has an active API key." />}
            {shown.length > 0 && <ApiKeysTable keys={shown} showOwner revoke={revoke.mutateAsync} revoking={revoke.isPending} />}
            {inactiveCount > 0 && (
              <label className="mt-3 flex items-center gap-2 text-xs text-fg-muted">
                <input type="checkbox" checked={showInactive} onChange={(e) => setShowInactive(e.target.checked)} />
                Show Inactive ({inactiveCount} expired or revoked)
              </label>
            )}
          </div>
        </Card>
        <RecentActivity />
      </div>
    </>
  )
}

function RecentActivity() {
  // Cursors of the pages before this one, so Newer can step back.
  const [cursors, setCursors] = useState<string[]>([])
  const before = cursors[cursors.length - 1]
  const { data, isLoading, error } = useAdminApiKeyActivity(before)
  return (
    <Card className="mt-4">
      <CardHeader title="Activity Log" description="Every MCP call made with anyone's keys, newest first. Open a call to see everything it did." />
      <div className="p-4 pt-2">
        <ApiKeyActivityLog
          page={data}
          isLoading={isLoading}
          error={error} admin
          pageIndex={cursors.length}
          onNewer={() => setCursors((c) => c.slice(0, -1))}
          onOlder={() => data?.next && setCursors((c) => [...c, data.next!])}
        />
      </div>
    </Card>
  )
}
