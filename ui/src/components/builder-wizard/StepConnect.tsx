import { useEffect, useState } from 'react'
import { AlertTriangle, CheckCircle2, Copy, Plug, Plus, Server, Trash2 } from 'lucide-react'
import { useBuilderConnection, useConnectBuilder } from '../../api/hooks'
import { ApiError } from '../../api/client'
import type { BuilderConnection } from '../../api/types'
import { Button, Input, Spinner, Textarea, cn } from '../../ui'
import { emptyHost, placementDefaults, shortFingerprint, type HostDraft, type Kind } from './model'

export function StepConnect({
  kind,
  hosts,
  onChange,
}: {
  kind: Kind
  hosts: HostDraft[]
  onChange: (hosts: HostDraft[]) => void
}) {
  const update = (i: number, h: HostDraft) => onChange(hosts.map((x, j) => (j === i ? h : x)))
  const remove = (i: number) => onChange(hosts.filter((_, j) => j !== i))

  return (
    <div className="flex flex-col gap-4">
      <p className="text-sm text-fg-muted">
        {kind === 'microcloud'
          ? 'Connect LaForge to your cluster. Any member works -- it answers for the whole cluster.'
          : 'Connect LaForge to each Incus server in the pool. Teams are spread evenly across them.'}
      </p>
      {hosts.map((h, i) => (
        <HostConnectCard
          key={h.key}
          host={h}
          title={kind === 'incus' ? `Host ${i + 1}` : 'Cluster'}
          kind={kind}
          onChange={(next) => update(i, next)}
          onRemove={kind === 'incus' && hosts.length > 1 ? () => remove(i) : undefined}
        />
      ))}
      {kind === 'incus' && (
        <div>
          <Button variant="secondary" size="sm" onClick={() => onChange([...hosts, emptyHost()])}>
            <Plus size={12} /> Add Host
          </Button>
        </div>
      )}
    </div>
  )
}

function HostConnectCard({
  host,
  title,
  kind,
  onChange,
  onRemove,
}: {
  host: HostDraft
  title: string
  kind: Kind
  onChange: (h: HostDraft) => void
  onRemove?: () => void
}) {
  // An existing host (editing a saved builder) re-reads what its server has
  // from the stored credential.
  const existing = useBuilderConnection(host.credentialId && !host.connection ? host.credentialId : undefined, kind)
  useEffect(() => {
    if (existing.data && !host.connection) onChange({ ...host, connection: existing.data })
  }, [existing.data, host, onChange])

  const reset = () => onChange({ key: host.key, storagePool: '', uplink: '', timeoutSeconds: host.timeoutSeconds })

  let body
  if (host.connection) {
    const c = host.connection.credential
    body = (
      <div className="flex items-start justify-between gap-3 rounded-token border border-success/30 bg-success-soft px-3 py-2.5">
        <div className="flex items-start gap-2">
          <CheckCircle2 size={16} className="mt-0.5 shrink-0 text-success" />
          <div className="text-sm">
            <div className="font-medium text-fg">Connected to {c.server_name || c.api_url}</div>
            <div className="mt-0.5 text-xs text-fg-muted">
              {c.api_url} · certificate <span className="font-mono">{shortFingerprint(c.server_fingerprint)}…</span>
            </div>
          </div>
        </div>
        <Button variant="ghost" size="sm" onClick={reset}>
          Use Another Server
        </Button>
      </div>
    )
  } else if (host.credentialId && existing.isLoading) {
    body = (
      <div className="flex items-center gap-2 text-sm text-fg-muted">
        <Spinner /> Checking the saved connection…
      </div>
    )
  } else if (host.credentialId && existing.error) {
    body = (
      <div className="flex flex-col gap-3">
        <div className="flex items-start gap-2 rounded-token border border-warning/30 bg-warning-soft px-3 py-2 text-sm text-warning">
          <AlertTriangle size={16} className="mt-0.5 shrink-0" />
          <span>
            Couldn't reach the saved server ({existing.error instanceof ApiError ? existing.error.message : 'unknown error'}). You can keep
            it as-is, or connect again with a new token.
          </span>
        </div>
        <TokenForm kind={kind} onConnected={(conn) => onChange({ ...host, credentialId: conn.credential.id, connection: conn, ...placementDefaults(conn) })} />
      </div>
    )
  } else if (host.legacy) {
    body = (
      <div className="flex items-start justify-between gap-3 rounded-token border border-border bg-surface-sunken px-3 py-2.5">
        <div className="text-sm">
          <div className="font-medium text-fg">Connected with certificate files</div>
          <div className="mt-0.5 text-xs text-fg-muted">
            {host.legacy.api_url} -- set up before token connections existed. It keeps working; switch to a token to pick pools, networks,
            and images from the server.
          </div>
        </div>
        <Button variant="ghost" size="sm" onClick={reset}>
          Switch to a Token
        </Button>
      </div>
    )
  } else {
    body = <TokenForm kind={kind} onConnected={(conn) => onChange({ ...host, credentialId: conn.credential.id, connection: conn, ...placementDefaults(conn) })} />
  }

  return (
    <div className="card p-4">
      <div className="mb-3 flex items-center justify-between">
        <div className="flex items-center gap-2 text-sm font-semibold text-fg">
          <Server size={14} className="text-fg-muted" /> {title}
        </div>
        {onRemove && (
          <Button variant="ghost" size="icon" onClick={onRemove} title="Remove this host" className="text-fg-muted hover:bg-danger-soft hover:text-danger">
            <Trash2 size={13} />
          </Button>
        )}
      </div>
      {body}
    </div>
  )
}

function TokenForm({ kind, onConnected }: { kind: Kind; onConnected: (conn: BuilderConnection) => void }) {
  const connect = useConnectBuilder()
  const [token, setToken] = useState('')
  const [address, setAddress] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [copied, setCopied] = useState(false)
  // LXD's positional argument is a certificate file (a token needs --name);
  // Incus's is the client name.
  const command = kind === 'microcloud' ? 'lxc config trust add --name laforge' : 'incus config trust add laforge'

  async function copy() {
    try {
      await navigator.clipboard.writeText(command)
      setCopied(true)
      setTimeout(() => setCopied(false), 1500)
    } catch {
      // Clipboard blocked -- the command is visible to copy by hand.
    }
  }

  async function onConnect() {
    setError(null)
    try {
      const conn = await connect.mutateAsync({ kind, token: token.trim(), address: address.trim() || undefined })
      onConnected(conn)
    } catch (e) {
      setError(e instanceof ApiError ? e.message : 'Connecting failed')
    }
  }

  return (
    <ol className="flex flex-col gap-4 text-sm">
      <li className="flex gap-3">
        <StepNumber n={1} />
        <div className="min-w-0 flex-1">
          <div className="mb-1.5 text-fg">{kind === 'microcloud' ? 'On any cluster member, run:' : 'On the server, run:'}</div>
          <div className="flex items-center gap-2">
            <code className="flex-1 rounded-token border border-border bg-surface-sunken px-3 py-1.5 font-mono text-xs text-fg">{command}</code>
            <Button variant="secondary" size="sm" onClick={copy}>
              <Copy size={12} /> {copied ? 'Copied' : 'Copy'}
            </Button>
          </div>
          {kind === 'microcloud' && (
            <div className="mt-1 text-xs text-fg-subtle">MicroCloud manages its instances with LXD, so this is LXD's client. Use sudo if your user isn't in the lxd group.</div>
          )}
        </div>
      </li>
      <li className="flex gap-3">
        <StepNumber n={2} />
        <div className="min-w-0 flex-1">
          <div className="mb-1.5 text-fg">Paste the token it prints:</div>
          <Textarea
            value={token}
            onChange={(e) => setToken(e.target.value)}
            rows={3}
            placeholder="eyJjbGllbnRfbmFtZSI6…"
            className="font-mono text-xs"
            spellCheck={false}
          />
          <details className="mt-2 text-xs text-fg-muted">
            <summary className="cursor-pointer select-none">The server isn't reachable at its own addresses?</summary>
            <div className="mt-2">
              <Input value={address} onChange={(e) => setAddress(e.target.value)} placeholder="cluster.example.org or 203.0.113.10:8443" />
              <div className="mt-1 text-fg-subtle">Only needed behind NAT or a proxy. The certificate is still checked against the token.</div>
            </div>
          </details>
        </div>
      </li>
      <li className="flex gap-3">
        <StepNumber n={3} />
        <div className="flex-1">
          <Button onClick={onConnect} disabled={!token.trim() || connect.isPending}>
            {connect.isPending ? <Spinner /> : <Plug size={12} />} {connect.isPending ? 'Connecting…' : 'Connect'}
          </Button>
          {error && <div className="mt-2 text-xs text-danger">{error}</div>}
        </div>
      </li>
    </ol>
  )
}

function StepNumber({ n }: { n: number }) {
  return (
    <span className={cn('flex size-5 shrink-0 items-center justify-center rounded-full bg-accent-soft text-[11px] font-semibold text-accent-fg')}>{n}</span>
  )
}
