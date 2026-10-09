import { useEffect, useState } from 'react'
import { AlertTriangle, CheckCircle2, Copy, Plug, Plus, Radar, Server, Trash2, XCircle } from 'lucide-react'
import { useBuilderConnection, useCheckBuilderToken, useConnectBuilder } from '../../api/hooks'
import { ApiError } from '../../api/client'
import type { BuilderConnection, TokenCheck } from '../../api/types'
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
  const existing = useBuilderConnection(host.credentialId && !host.connection ? host.credentialId : undefined, kind, host.project)
  useEffect(() => {
    if (existing.data && !host.connection) onChange({ ...host, connection: existing.data })
  }, [existing.data, host, onChange])

  const reset = () => onChange({ key: host.key, storagePool: '', uplink: '', project: '', timeoutSeconds: host.timeoutSeconds })

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
  const check = useCheckBuilderToken()
  const [token, setToken] = useState('')
  const [address, setAddress] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [checked, setChecked] = useState<TokenCheck | null>(null)
  const [checkError, setCheckError] = useState<string | null>(null)
  const parsed = parseTrustToken(token)

  // A check describes one token and one address choice; editing either makes it stale.
  function edit(set: (v: string) => void, v: string) {
    set(v)
    setChecked(null)
    setCheckError(null)
  }

  async function onCheck() {
    setCheckError(null)
    try {
      setChecked(await check.mutateAsync({ kind, token: token.trim(), address: address.trim() || undefined }))
    } catch (e) {
      setCheckError(e instanceof ApiError ? e.message : 'Checking failed')
    }
  }
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
            onChange={(e) => edit(setToken, e.target.value)}
            rows={3}
            placeholder="eyJjbGllbnRfbmFtZSI6…"
            className="font-mono text-xs"
            spellCheck={false}
          />
          {parsed && 'error' in parsed && token.trim() !== '' && <div className="mt-1 text-xs text-danger">{parsed.error}</div>}
          {parsed && !('error' in parsed) && <TokenDetails token={parsed} kind={kind} override={address.trim()} check={checked} />}
          <details className="mt-2 text-xs text-fg-muted" open={address !== '' || undefined}>
            <summary className="cursor-pointer select-none">The server isn't reachable at its own addresses?</summary>
            <div className="mt-2">
              <Input value={address} onChange={(e) => edit(setAddress, e.target.value)} placeholder="cluster.example.org or 203.0.113.10:8443" />
              <div className="mt-1 text-fg-subtle">
                Replaces the token's addresses: only this one is tried. Only needed behind NAT or a proxy. The certificate is still checked against the token.
              </div>
            </div>
          </details>
        </div>
      </li>
      <li className="flex gap-3">
        <StepNumber n={3} />
        <div className="flex-1">
          <div className="flex flex-wrap items-center gap-2">
            <Button variant="brand" onClick={onConnect} disabled={!token.trim() || connect.isPending}>
              {connect.isPending ? <Spinner /> : <Plug size={12} />} {connect.isPending ? 'Connecting…' : 'Connect'}
            </Button>
            <Button
              variant="secondary"
              onClick={onCheck}
              disabled={!parsed || 'error' in parsed || check.isPending}
              title="Try each address from the LaForge server without using up the token"
            >
              {check.isPending ? <Spinner /> : <Radar size={12} />} {check.isPending ? 'Checking…' : 'Check addresses'}
            </Button>
          </div>
          {checkError && <div className="mt-2 text-xs text-danger">{checkError}</div>}
          {error && <div className="mt-2 text-xs text-danger">{error}</div>}
        </div>
      </li>
    </ol>
  )
}

// ParsedToken is the public part of a trust token -- everything except its
// secret, which is never shown.
interface ParsedToken {
  clientName: string
  fingerprint: string
  addresses: string[]
  expiresAt?: Date
  type?: string
}

// parseTrustToken decodes a pasted Incus/LXD trust token the way the server
// does (base64 or base64url JSON, ignoring line wrapping), for display only.
function parseTrustToken(raw: string): ParsedToken | { error: string } | null {
  const cleaned = raw.replace(/\s+/g, '')
  if (!cleaned) return null
  let json: unknown
  try {
    const b64 = cleaned.replace(/-/g, '+').replace(/_/g, '/')
    json = JSON.parse(atob(b64 + '='.repeat((4 - (b64.length % 4)) % 4)))
  } catch {
    return { error: "That doesn't look like a trust token -- copy the whole token the command printed." }
  }
  const t = json as { client_name?: string; fingerprint?: string; addresses?: string[]; secret?: string; expires_at?: string; type?: string }
  if (!t.fingerprint || !t.secret || !Array.isArray(t.addresses) || t.addresses.length === 0) {
    return { error: 'The token is missing its fingerprint, secret, or addresses.' }
  }
  const expires = t.expires_at ? new Date(t.expires_at) : undefined
  return {
    clientName: t.client_name ?? '',
    fingerprint: t.fingerprint,
    addresses: t.addresses,
    expiresAt: expires && !isNaN(expires.getTime()) && expires.getFullYear() > 1 ? expires : undefined,
    type: t.type || undefined,
  }
}

// normalizeAddress mirrors the server's: no scheme or trailing slash, and port
// 8443 (the Incus/LXD default) when none is given.
function normalizeAddress(a: string): string {
  const s = a.trim().replace(/^https?:\/\//, '').replace(/\/$/, '')
  const hasPort = s.startsWith('[') ? /\]:\d+$/.test(s) : /^[^:]+:\d+$/.test(s)
  if (hasPort) return s
  const host = s.replace(/^\[|\]$/g, '')
  return host.includes(':') ? `[${host}]:8443` : `${host}:8443`
}

function relativeTime(d: Date, now: number): string {
  const mins = Math.round((d.getTime() - now) / 60000)
  const abs = Math.abs(mins)
  const span = abs < 60 ? `${abs} minute${abs === 1 ? '' : 's'}` : abs < 48 * 60 ? `${Math.round(abs / 60)} hours` : `${Math.round(abs / 1440)} days`
  return mins >= 0 ? `in ${span}` : `${span} ago`
}

function TokenDetails({ token, kind, override, check }: { token: ParsedToken; kind: Kind; override: string; check: TokenCheck | null }) {
  const [now] = useState(() => Date.now())
  const expired = token.expiresAt && token.expiresAt.getTime() < now
  const cli = kind === 'microcloud' ? 'lxc' : 'incus'
  const addresses = override ? [normalizeAddress(override)] : token.addresses
  const result = (addr: string, i: number) => check?.addresses[i] ?? check?.addresses.find((a) => a.address === addr)
  return (
    <div className="mt-2 rounded-token border border-border bg-surface-sunken p-3 text-xs">
      <dl className="grid grid-cols-[auto_1fr] gap-x-3 gap-y-1">
        <dt className="text-fg-muted">Added as</dt>
        <dd className="font-mono text-fg">{token.clientName || '(unnamed)'}</dd>
        <dt className="text-fg-muted">Server certificate</dt>
        <dd className="text-fg">
          <span className="font-mono" title={token.fingerprint}>
            {shortFingerprint(token.fingerprint)}…
          </span>{' '}
          <span className="text-fg-subtle">
            compare with <span className="font-mono">certificate_fingerprint</span> in <span className="font-mono">{cli} info</span>
          </span>
        </dd>
        <dt className="text-fg-muted">Expires</dt>
        <dd className={expired ? 'text-danger' : 'text-fg'}>
          {token.expiresAt ? `${token.expiresAt.toLocaleString()} (${relativeTime(token.expiresAt, now)})` : 'No expiry'}
          {expired && ' -- generate a new token'}
        </dd>
        <dt className="text-fg-muted">Kind</dt>
        <dd className="text-fg">{token.type ? `LXD identity token (${token.type})` : 'Trust token'}</dd>
      </dl>
      {token.type && kind === 'incus' && (
        <div className="mt-2 text-warning">This is an LXD identity token, which Incus doesn't issue. Did you mean a MicroCloud builder?</div>
      )}

      <div className="mt-3 mb-1 text-fg-muted">
        {override ? 'Will try only the address you entered (the token lists ' + token.addresses.length + '):' : `Will try these ${addresses.length} addresses, in order:`}
      </div>
      <ol className="flex flex-col gap-1">
        {addresses.map((addr, i) => {
          const r = result(addr, i)
          return (
            <li key={addr + i} className="flex flex-wrap items-baseline gap-x-2">
              <span className="w-4 text-right text-fg-subtle">{i + 1}.</span>
              {r ? (
                r.fingerprint_matches ? (
                  <CheckCircle2 size={12} className="self-center text-success" />
                ) : r.reachable ? (
                  <AlertTriangle size={12} className="self-center text-warning" />
                ) : (
                  <XCircle size={12} className="self-center text-danger" />
                )
              ) : null}
              <span className={cn('font-mono', check && check.will_use === (r?.address ?? addr) ? 'font-semibold text-fg' : 'text-fg')}>{addr}</span>
              {r && <span className="text-fg-subtle">{r.fingerprint_matches ? `reachable, ${r.millis} ms` : r.error}</span>}
            </li>
          )
        })}
      </ol>
      {check && (
        <div className={cn('mt-2', check.will_use ? 'text-success' : 'text-danger')}>
          {check.will_use
            ? `Connect will use ${check.will_use}.`
            : 'None of these can be reached from the LaForge server with the right certificate. Enter an address that can, or fix the route between them.'}
        </div>
      )}
      {!override && !check && token.addresses.length > 1 && (
        <div className="mt-2 text-fg-subtle">All are tried at once, for 4 seconds each; Connect uses the first in this order that answers with the right certificate.</div>
      )}
    </div>
  )
}

function StepNumber({ n }: { n: number }) {
  return (
    <span className={cn('flex size-5 shrink-0 items-center justify-center rounded-full bg-accent-soft text-[11px] font-semibold text-accent-fg')}>{n}</span>
  )
}
