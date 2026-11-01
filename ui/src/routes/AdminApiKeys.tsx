import { useState } from 'react'
import { Activity, KeyRound } from 'lucide-react'
import { useAdminApiKeyActivity, useAdminApiKeys, useAdminRevokeApiKey } from '../api/hooks'
import { ApiError } from '../api/client'
import { EmptyState } from '../components/EmptyState'
import { ApiKeyActivityTable } from '../components/ApiKeyActivityTable'
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
  const { data: rows, isLoading, error } = useAdminApiKeyActivity()
  return (
    <Card className="mt-4">
      <CardHeader title="Recent Activity" description="The last 50 MCP calls made with anyone's keys. Power actions are marked." />
      <div className="p-4 pt-2">
        {isLoading && (
          <div className="flex items-center gap-2 text-sm text-fg-muted">
            <Spinner /> Loading…
          </div>
        )}
        {error && <div className="text-sm text-danger">{error instanceof ApiError ? error.message : 'Failed to load activity.'}</div>}
        {!isLoading && !error && (rows ?? []).length === 0 && <EmptyState icon={Activity} title="No Activity Yet" hint="Calls made with API keys show up here." />}
        {rows && rows.length > 0 && <ApiKeyActivityTable rows={rows} showOwner />}
      </div>
    </Card>
  )
}
