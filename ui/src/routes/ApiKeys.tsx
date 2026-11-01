import { useState } from 'react'
import { AppWindow, Check, Copy, KeyRound, Link2, Plus, TriangleAlert } from 'lucide-react'
import { useApiKeyActivity, useApiKeys, useCreateApiKey, useRepositories, useRevokeApiKey } from '../api/hooks'
import { API_BASE, ApiError } from '../api/client'
import { EmptyState } from '../components/EmptyState'
import { ApiKeyActivityLog } from '../components/ApiKeyActivityLog'
import { ApiKeysTable } from '../components/ApiKeysTable'
import type { ApiKey, CreatedApiKey } from '../api/types'
import { mcpClients, type McpClient } from './mcpClients'
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
  const [connectOpen, setConnectOpen] = useState(false)

  // Pasted keys, and connected apps (keys an app got by OAuth sign-in).
  const pasted = (keys ?? []).filter((k) => k.kind !== 'app')
  const apps = (keys ?? []).filter((k) => k.kind === 'app')
  const active = pasted.filter((k) => k.status === 'active')
  const inactiveCount = pasted.length - active.length
  const shown = showInactive ? pasted : active

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
        <ConnectedApps apps={apps} loading={isLoading} revoke={revoke.mutateAsync} revoking={revoke.isPending} onConnect={() => setConnectOpen(true)} />
        <RecentActivity />
      </div>
      {connectOpen && <ConnectAppModal onClose={() => setConnectOpen(false)} />}
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

const CLIENT_KEY = 'laforge.apiKeys.client'

function savedClient(): string {
  try {
    return localStorage.getItem(CLIENT_KEY) ?? 'claude-code'
  } catch {
    return 'claude-code'
  }
}

// Inline `code` in a step's text.
function StepText({ text }: { text: string }) {
  return (
    <>
      {text.split('`').map((part, i) =>
        i % 2 === 1 ? (
          <code key={i} className="rounded bg-surface-sunken px-1 font-mono text-[11px]">
            {part}
          </code>
        ) : (
          part
        ),
      )}
    </>
  )
}

// Shown once, right after creation: the only time the key itself is visible.
function NewKeyModal({ created, onClose }: { created: CreatedApiKey; onClose: () => void }) {
  const clients = mcpClients(`${API_BASE}/mcp`, created.token)
  const [clientId, setClientId] = useState(savedClient)
  const client = clients.find((c) => c.id === clientId) ?? clients[0]

  function pick(id: string) {
    setClientId(id)
    try {
      localStorage.setItem(CLIENT_KEY, id)
    } catch {
      // Not remembered; the default is fine.
    }
  }

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
        <div>
          <Label>Connect From</Label>
          <Select className="mt-1 w-full" value={client.id} onChange={(e) => pick(e.target.value)}>
            {clients.map((c) => (
              <option key={c.id} value={c.id}>
                {c.name}
              </option>
            ))}
          </Select>
        </div>
        {client.auth === 'oauth' ? (
          <div className="rounded-token border border-border bg-surface-sunken p-2 text-xs text-fg-muted">
            {client.name} doesn't use a key: it signs in with your LaForge account. Use Connect An App on the API Keys page instead.
          </div>
        ) : (
          <ClientInstructions client={client} />
        )}
      </div>
    </Modal>
  )
}

function RecentActivity() {
  // Cursors of the pages before this one, so Newer can step back.
  const [cursors, setCursors] = useState<string[]>([])
  const before = cursors[cursors.length - 1]
  const { data, isLoading, error } = useApiKeyActivity(before)
  return (
    <Card className="mt-4">
      <CardHeader title="Activity Log" description="Every MCP call made with your keys, newest first. Open a call to see everything it did." />
      <div className="p-4 pt-2">
        <ApiKeyActivityLog
          page={data}
          isLoading={isLoading}
          error={error}
          pageIndex={cursors.length}
          onNewer={() => setCursors((c) => c.slice(0, -1))}
          onOlder={() => data?.next && setCursors((c) => [...c, data.next!])}
        />
      </div>
    </Card>
  )
}

// A client's note, steps and copyable snippets.
function ClientInstructions({ client }: { client: McpClient }) {
  return (
    <>
      {client.note && <div className="rounded-token border border-border bg-surface-sunken p-2 text-xs text-fg-muted">{client.note}</div>}
      {client.steps.length > 0 && (
        <ol className="list-decimal space-y-1 pl-5 text-xs text-fg">
          {client.steps.map((step) => (
            <li key={step}>
              <StepText text={step} />
            </li>
          ))}
        </ol>
      )}
      {client.snippets.map((sn) => (
        <CopyBlock key={`${client.id}-${sn.label}`} label={sn.label} text={sn.text} />
      ))}
    </>
  )
}

// Apps that signed in with the person's account (ChatGPT, Claude.ai) instead
// of a pasted key. Each one is a key underneath, so Revoke works the same.
function ConnectedApps({
  apps,
  loading,
  revoke,
  revoking,
  onConnect,
}: {
  apps: ApiKey[]
  loading: boolean
  revoke: (id: string) => Promise<unknown>
  revoking: boolean
  onConnect: () => void
}) {
  const [showInactive, setShowInactive] = useState(false)
  const active = apps.filter((k) => k.status === 'active')
  const shown = showInactive ? apps : active
  return (
    <Card className="mt-4">
      <CardHeader
        title="Connected Apps"
        description="Apps that signed in with your LaForge account instead of a key. They see what a key sees, and each one's access lasts 30 days."
        actions={
          <Button variant="primary" size="sm" onClick={onConnect}>
            <Link2 size={14} /> Connect An App
          </Button>
        }
      />
      <div className="p-4 pt-2">
        {!loading && shown.length === 0 && (
          <EmptyState icon={AppWindow} title="No Connected Apps" hint="Connect ChatGPT or Claude.ai to use LaForge from them without a key." />
        )}
        {shown.length > 0 && <ApiKeysTable keys={shown} revoke={revoke} revoking={revoking} />}
        {apps.length > active.length && (
          <label className="mt-3 flex items-center gap-2 text-xs text-fg-muted">
            <input type="checkbox" checked={showInactive} onChange={(e) => setShowInactive(e.target.checked)} />
            Show Inactive ({apps.length - active.length} expired or revoked)
          </label>
        )}
      </div>
    </Card>
  )
}

// How to connect an app that signs in with OAuth. No key is made here: the
// app gets its own when the person approves it.
function ConnectAppModal({ onClose }: { onClose: () => void }) {
  const clients = mcpClients(`${API_BASE}/mcp`).filter((c) => c.auth === 'oauth')
  const [clientId, setClientId] = useState(clients[0].id)
  const client = clients.find((c) => c.id === clientId) ?? clients[0]
  return (
    <Modal
      open
      onClose={onClose}
      title="Connect An App"
      description="These apps sign in with your LaForge account. Add LaForge in the app, then approve it when LaForge asks."
      footer={
        <Button variant="primary" onClick={onClose}>
          Done
        </Button>
      }
    >
      <div className="flex flex-col gap-4">
        <div>
          <Label>App</Label>
          <Select className="mt-1 w-full" value={client.id} onChange={(e) => setClientId(e.target.value)}>
            {clients.map((c) => (
              <option key={c.id} value={c.id}>
                {c.name}
              </option>
            ))}
          </Select>
        </div>
        <ClientInstructions client={client} />
      </div>
    </Modal>
  )
}
