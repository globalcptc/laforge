import { useState } from 'react'
import { Activity, Check, Copy, KeyRound, Plus, TriangleAlert } from 'lucide-react'
import { useApiKeyActivity, useApiKeys, useCreateApiKey, useRepositories, useRevokeApiKey } from '../api/hooks'
import { API_BASE, ApiError } from '../api/client'
import { EmptyState } from '../components/EmptyState'
import { ApiKeyActivityTable } from '../components/ApiKeyActivityTable'
import { ApiKeysTable } from '../components/ApiKeysTable'
import type { CreatedApiKey } from '../api/types'
import { Button, Card, CardHeader, Input, Label, Modal, PageHeader, Select, Spinner } from '../ui'

const LIFETIMES = [7, 30, 90]

// A person's own API keys: what their AI agent (Claude Code, Cursor, ...) uses
// to read their builds through LaForge's MCP server. Reached from the user
// menu. A key sees what its owner can see, and its only action is powering
// hosts, for an owner with manage access.
export function ApiKeys() {
  const { data: keys, isLoading, error } = useApiKeys()
  const revoke = useRevokeApiKey()
  const [createOpen, setCreateOpen] = useState(false)
  const [created, setCreated] = useState<CreatedApiKey | null>(null)
  const [showInactive, setShowInactive] = useState(false)

  const active = (keys ?? []).filter((k) => k.status === 'active')
  const inactiveCount = (keys ?? []).length - active.length
  const shown = showInactive ? (keys ?? []) : active

  return (
    <>
      <PageHeader
        title="API Keys"
        description="Keys let an AI agent read your builds through LaForge's MCP server. A key sees everything you can, passwords included. Its only action is starting, stopping or rebooting hosts, where you have manage access; it can't rebuild, deploy or tear down."
      />
      <div className="page-body">
        <Card>
          <CardHeader
            title="Your Keys"
            description="Every key expires. Revoke a key as soon as you stop using it."
            actions={
              <Button variant="primary" size="sm" onClick={() => setCreateOpen(true)}>
                <Plus size={14} /> Create API Key
              </Button>
            }
          />
          <div className="p-4 pt-2">
            {isLoading && (
              <div className="flex items-center gap-2 text-sm text-fg-muted">
                <Spinner /> Loading…
              </div>
            )}
            {error && <div className="text-sm text-danger">{error instanceof ApiError ? error.message : 'Failed to load keys.'}</div>}
            {!isLoading && !error && shown.length === 0 && (
              <EmptyState icon={KeyRound} title="No Active Keys" hint="Create a key to connect an AI agent to LaForge." />
            )}
            {shown.length > 0 && <ApiKeysTable keys={shown} revoke={revoke.mutateAsync} revoking={revoke.isPending} />}
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
      <CreateKeyModal
        open={createOpen}
        onClose={() => setCreateOpen(false)}
        onCreated={(c) => {
          setCreateOpen(false)
          setCreated(c)
        }}
      />
      {created && <NewKeyModal created={created} onClose={() => setCreated(null)} />}
    </>
  )
}

function CreateKeyModal({ open, onClose, onCreated }: { open: boolean; onClose: () => void; onCreated: (c: CreatedApiKey) => void }) {
  const create = useCreateApiKey()
  const { data: repos } = useRepositories()
  const [name, setName] = useState('')
  const [days, setDays] = useState(30)
  const [limit, setLimit] = useState(false)
  const [repoIds, setRepoIds] = useState<string[]>([])
  const [error, setError] = useState<string | null>(null)

  const trimmed = name.trim()
  const valid = trimmed !== '' && (!limit || repoIds.length > 0)

  function close() {
    setName('')
    setDays(30)
    setLimit(false)
    setRepoIds([])
    setError(null)
    onClose()
  }

  async function onCreate() {
    setError(null)
    try {
      const c = await create.mutateAsync({ name: trimmed, expires_in_days: days, repository_ids: limit ? repoIds : [] })
      setName('')
      setLimit(false)
      setRepoIds([])
      onCreated(c)
    } catch (e) {
      setError(e instanceof ApiError ? e.message : 'Create failed')
    }
  }

  function toggleRepo(id: string, on: boolean) {
    setRepoIds((ids) => (on ? [...ids, id] : ids.filter((x) => x !== id)))
  }

  return (
    <Modal
      open={open}
      onClose={close}
      title="Create API Key"
      description="The key sees the builds you can see, passwords included, and can power hosts wherever you have manage access. It can't rebuild, deploy, tear down or run commands."
      footer={
        <>
          <Button variant="ghost" onClick={close}>
            Cancel
          </Button>
          <Button variant="primary" onClick={onCreate} disabled={!valid || create.isPending}>
            {create.isPending ? <Spinner /> : <Plus size={14} />} Create API Key
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-4">
        <div>
          <Label>Name</Label>
          <Input className="mt-1" placeholder="Claude on my laptop" value={name} maxLength={100} onChange={(e) => setName(e.target.value)} autoFocus />
        </div>
        <div>
          <Label>Expires In</Label>
          <Select className="mt-1 w-full" value={String(days)} onChange={(e) => setDays(Number(e.target.value))}>
            {LIFETIMES.map((d) => (
              <option key={d} value={d}>
                {d} days
              </option>
            ))}
          </Select>
        </div>
        <div>
          <label className="flex items-center gap-2 text-sm text-fg">
            <input type="checkbox" checked={limit} onChange={(e) => setLimit(e.target.checked)} />
            Limit To Specific Repositories
          </label>
          {limit && (
            <div className="mt-2 flex max-h-48 flex-col gap-1.5 overflow-y-auto rounded-token border border-border p-2">
              {(repos ?? []).length === 0 && <div className="text-xs text-fg-muted">You can't see any repositories.</div>}
              {(repos ?? []).map((r) => (
                <label key={r.id} className="flex items-center gap-2 font-mono text-xs text-fg">
                  <input type="checkbox" checked={repoIds.includes(r.id)} onChange={(e) => toggleRepo(r.id, e.target.checked)} />
                  {r.github_owner}/{r.github_repo}
                </label>
              ))}
            </div>
          )}
        </div>
        {error && <div className="rounded-token border border-danger/30 bg-danger-soft p-2 text-xs text-danger">{error}</div>}
      </div>
    </Modal>
  )
}

function CopyBlock({ label, text }: { label: string; text: string }) {
  const [copied, setCopied] = useState(false)
  async function copy() {
    try {
      await navigator.clipboard.writeText(text)
      setCopied(true)
      setTimeout(() => setCopied(false), 1500)
    } catch {
      // Clipboard blocked -- the text is visible to copy by hand.
    }
  }
  return (
    <div>
      <div className="mb-1 flex items-center justify-between">
        <Label>{label}</Label>
        <Button variant="brand" size="sm" onClick={copy}>
          {copied ? <Check size={12} /> : <Copy size={12} />} {copied ? 'Copied' : 'Copy'}
        </Button>
      </div>
      <pre className="overflow-x-auto whitespace-pre-wrap break-all rounded-token border border-border bg-surface-sunken px-3 py-2 font-mono text-xs text-fg">{text}</pre>
    </div>
  )
}

// Shown once, right after creation: the only time the key itself is visible.
function NewKeyModal({ created, onClose }: { created: CreatedApiKey; onClose: () => void }) {
  const mcpURL = `${API_BASE}/mcp`
  const claudeCode = `claude mcp add --transport http laforge ${mcpURL} --header "Authorization: Bearer ${created.token}"`
  const mcpJSON = JSON.stringify(
    { mcpServers: { laforge: { type: 'http', url: mcpURL, headers: { Authorization: `Bearer ${created.token}` } } } },
    null,
    2,
  )
  return (
    <Modal
      open
      onClose={onClose}
      title="Copy Your New Key"
      description={`"${created.key.name}" is ready. Copy it now: LaForge only stores a hash, so it can't show you this key again.`}
      footer={
        <Button variant="primary" onClick={onClose}>
          Done
        </Button>
      }
    >
      <div className="flex flex-col gap-4">
        <div className="flex items-start gap-2 rounded-token border border-warning/30 bg-warning-soft p-2 text-xs text-warning">
          <TriangleAlert size={14} className="mt-0.5 shrink-0" />
          Treat this like a password. Anyone with it can read your builds until it expires or you revoke it.
        </div>
        <CopyBlock label="API Key" text={created.token} />
        <CopyBlock label="Claude Code" text={claudeCode} />
        <CopyBlock label="MCP Config (.mcp.json, Cursor)" text={mcpJSON} />
      </div>
    </Modal>
  )
}

function RecentActivity() {
  const { data: rows, isLoading, error } = useApiKeyActivity()
  return (
    <Card className="mt-4">
      <CardHeader title="Recent Activity" description="The last 50 MCP calls made with your keys. Power actions are marked." />
      <div className="p-4 pt-2">
        {isLoading && (
          <div className="flex items-center gap-2 text-sm text-fg-muted">
            <Spinner /> Loading…
          </div>
        )}
        {error && <div className="text-sm text-danger">{error instanceof ApiError ? error.message : 'Failed to load activity.'}</div>}
        {!isLoading && !error && (rows ?? []).length === 0 && <EmptyState icon={Activity} title="No Activity Yet" hint="Calls made with API keys show up here." />}
        {rows && rows.length > 0 && <ApiKeyActivityTable rows={rows} />}
      </div>
    </Card>
  )
}
