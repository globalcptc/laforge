import { Globe, HardDrive, Network, Server } from 'lucide-react'
import { Field, Input, Select } from '../../ui'
import { TIMEOUT_OPTIONS, uplinkCandidates, type HostDraft, type Kind } from './model'

export function StepPlacement({
  kind,
  hosts,
  onChange,
  externalAccessIp,
  externalPortMin,
  externalPortMax,
  onExternalChange,
}: {
  kind: Kind
  hosts: HostDraft[]
  onChange: (hosts: HostDraft[]) => void
  externalAccessIp: string
  externalPortMin?: number
  externalPortMax?: number
  onExternalChange: (patch: { externalAccessIp?: string; externalPortMin?: number; externalPortMax?: number }) => void
}) {
  const update = (i: number, patch: Partial<HostDraft>) => onChange(hosts.map((h, j) => (j === i ? { ...h, ...patch } : h)))
  const num = (v: string): number | undefined => (v.trim() === '' ? undefined : Number(v))

  return (
    <div className="flex flex-col gap-4">
      <p className="text-sm text-fg-muted">Choose where machines are stored and how team networks reach the outside, from what the server actually has.</p>
      {hosts.map((h, i) => (
        <div key={h.key} className="card p-4">
          <div className="mb-3 flex items-center gap-2 text-sm font-semibold text-fg">
            <Server size={14} className="text-fg-muted" />
            {kind === 'incus' ? `Host ${i + 1}` : 'Cluster'}
            {h.connection && <span className="font-normal text-fg-muted">· {h.connection.credential.server_name || h.connection.credential.api_url}</span>}
          </div>

          {h.connection ? (
            <div className="grid gap-4 sm:grid-cols-2">
              <Field
                label={
                  <span className="flex items-center gap-1.5">
                    <HardDrive size={12} /> Storage pool
                  </span>
                }
                hint="Where each machine's disk is created."
                className="mb-0"
              >
                <Select value={h.storagePool} onChange={(e) => update(i, { storagePool: e.target.value })}>
                  <option value="">Choose a pool…</option>
                  {h.connection.discovery.storage_pools.map((p) => (
                    <option key={p.name} value={p.name}>
                      {p.name} ({p.driver})
                    </option>
                  ))}
                </Select>
              </Field>
              <Field
                label={
                  <span className="flex items-center gap-1.5">
                    <Network size={12} /> Uplink network
                  </span>
                }
                hint="The existing network each team's network routes out through. MicroCloud names it UPLINK by convention."
                className="mb-0"
              >
                <Select value={h.uplink} onChange={(e) => update(i, { uplink: e.target.value })}>
                  <option value="">Choose a network…</option>
                  {uplinkCandidates(h.connection.discovery.networks).map((n) => (
                    <option key={n.name} value={n.name}>
                      {n.name} ({n.type})
                    </option>
                  ))}
                </Select>
              </Field>
            </div>
          ) : (
            <div className="text-sm text-fg-muted">
              This host uses its saved settings -- storage pool <span className="font-mono text-fg">{h.storagePool || '(default)'}</span>, uplink{' '}
              <span className="font-mono text-fg">{h.uplink || '(none)'}</span>. Connect it with a token to choose from the server's own list.
            </div>
          )}

          <details className="mt-3 text-xs text-fg-muted">
            <summary className="cursor-pointer select-none">Advanced</summary>
            <div className="mt-2 max-w-xs">
              <Field label="Wait for slow operations up to" hint="Large images on network storage can take minutes to copy." className="mb-0">
                <Select value={String(h.timeoutSeconds)} onChange={(e) => update(i, { timeoutSeconds: Number(e.target.value) })}>
                  {TIMEOUT_OPTIONS.map((s) => (
                    <option key={s} value={s}>
                      {s < 60 ? `${s} seconds` : `${s / 60} minute${s === 60 ? '' : 's'}`}
                    </option>
                  ))}
                </Select>
              </Field>
            </div>
          </details>
        </div>
      ))}

      {/* Builder-level external access -- one shared IP across teams, with a
          port window; a host's public: ports get NAT'd in on it. */}
      <div className="card p-4">
        <div className="mb-1 flex items-center gap-2 text-sm font-semibold text-fg">
          <Globe size={14} className="text-fg-muted" /> External access
        </div>
        <p className="mb-3 text-xs text-fg-muted">
          The single external IP that hosts' <span className="font-mono">public:</span> ports (e.g. RDP) are NAT'd in on, shared across every team with a
          distinct external port each. Leave the IP blank to disable external access for this builder.
        </p>
        <div className="grid gap-4 sm:grid-cols-3">
          <Field label="External IP" hint="Shared across all teams." className="mb-0">
            <Input value={externalAccessIp} onChange={(e) => onExternalChange({ externalAccessIp: e.target.value })} placeholder="203.0.113.10" />
          </Field>
          <Field label="Port range start" hint="Default 40000." className="mb-0">
            <Input type="number" value={externalPortMin ?? ''} onChange={(e) => onExternalChange({ externalPortMin: num(e.target.value) })} placeholder="40000" />
          </Field>
          <Field label="Port range end" hint="Default 50000." className="mb-0">
            <Input type="number" value={externalPortMax ?? ''} onChange={(e) => onExternalChange({ externalPortMax: num(e.target.value) })} placeholder="50000" />
          </Field>
        </div>
      </div>
    </div>
  )
}
