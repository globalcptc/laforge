import { useState } from 'react'
import { FolderTree, Globe, HardDrive, Network, Plus, Server, X } from 'lucide-react'
import { api, ApiError } from '../../api/client'
import { builderConnectionPath } from '../../api/hooks'
import type { BuilderConnection, ProjectInfo } from '../../api/types'
import { Button, Field, Input, Select, Spinner } from '../../ui'
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

  // External IPs are stored as one comma-joined string (the builder parses it),
  // but edited as rows -- each row one IP or range. Local state keeps otherwise
  // blank rows visible (they are dropped from the stored value until filled).
  const [ipRows, setIpRows] = useState<string[]>(() => {
    const parts = externalAccessIp.split(',').map((s) => s.trim()).filter(Boolean)
    return parts.length > 0 ? parts : ['']
  })
  const emitIps = (rows: string[]) => {
    setIpRows(rows)
    onExternalChange({ externalAccessIp: rows.map((r) => r.trim()).filter(Boolean).join(', ') })
  }
  const setIpRow = (i: number, v: string) => emitIps(ipRows.map((r, j) => (j === i ? v : r)))
  const addIpRow = () => emitIps([...ipRows, ''])
  const removeIpRow = (i: number) => {
    const next = ipRows.filter((_, j) => j !== i)
    emitIps(next.length > 0 ? next : [''])
  }

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

          {kind === 'microcloud' && h.connection && h.credentialId && (
            <ProjectPicker host={h} onChange={(patch) => update(i, patch)} />
          )}

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

      {/* Builder-level external access -- one or more IPs, assigned one per team
          round-robin, with a port window; a host's public: ports get NAT'd in. */}
      <div className="card p-4">
        <div className="mb-1 flex items-center gap-2 text-sm font-semibold text-fg">
          <Globe size={14} className="text-fg-muted" /> External access
        </div>
        <p className="mb-3 text-xs text-fg-muted">
          The external IPs that hosts' <span className="font-mono">public:</span> ports (e.g. RDP) are NAT'd in on. Each team is assigned one IP,
          round-robin, so with enough IPs every team gets its own address; teams that share one get a distinct external port each. Add no IPs to disable
          external access for this builder.
        </p>
        <div className="flex flex-col gap-4">
          <div className="flex flex-col gap-2">
            <span className="text-xs font-medium text-fg-muted">External IPs</span>
            {ipRows.map((row, i) => (
              <div key={i} className="flex items-center gap-2">
                <div className="flex-1">
                  <Input value={row} onChange={(e) => setIpRow(i, e.target.value)} placeholder="203.0.113.10  or  203.0.113.20-25" />
                </div>
                <Button variant="ghost" size="icon" onClick={() => removeIpRow(i)} aria-label="Remove IP" title="Remove">
                  <X size={14} />
                </Button>
              </div>
            ))}
            <div>
              <Button variant="secondary" size="sm" onClick={addIpRow}>
                <Plus size={14} /> Add IP or Range
              </Button>
            </div>
            <p className="text-xs text-fg-subtle">Each row is one IP or a range — full (203.0.113.20-203.0.113.25) or short (203.0.113.20-25).</p>
          </div>
          <div className="grid gap-4 sm:grid-cols-2">
            <Field label="Port range start" hint="Default 40000." className="mb-0">
              <Input type="number" value={externalPortMin ?? ''} onChange={(e) => onExternalChange({ externalPortMin: num(e.target.value) })} placeholder="40000" />
            </Field>
            <Field label="Port range end" hint="Default 50000." className="mb-0">
              <Input type="number" value={externalPortMax ?? ''} onChange={(e) => onExternalChange({ externalPortMax: num(e.target.value) })} placeholder="50000" />
            </Field>
          </div>
        </div>
      </div>
    </div>
  )
}

// ProjectPicker chooses the LXD project a MicroCloud builder creates everything
// in. Images can differ per project, so choosing one re-reads the cluster's
// images in it. When LaForge can't list projects (an identity allowed only some),
// the name is typed instead.
function ProjectPicker({ host, onChange }: { host: HostDraft; onChange: (patch: Partial<HostDraft>) => void }) {
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const projects = host.connection?.discovery.projects ?? []
  const current = host.project || 'default'
  const info = projects.find((p) => p.name === current)

  async function choose(name: string) {
    const project = name === 'default' ? '' : name
    onChange({ project })
    if (!host.credentialId) return
    setLoading(true)
    setError(null)
    try {
      const connection = await api.get<BuilderConnection>(builderConnectionPath(host.credentialId, 'microcloud', project))
      onChange({ project, connection })
    } catch (e) {
      setError(e instanceof ApiError ? e.message : 'Could not read that project')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="mb-4">
      <Field
        label={
          <span className="flex items-center gap-1.5">
            <FolderTree size={12} /> Project {loading && <Spinner />}
          </span>
        }
        hint="The LXD project every instance, team network, network ACL, image and volume is created in."
        className="mb-0 max-w-sm"
      >
        {projects.length > 0 ? (
          <Select value={current} onChange={(e) => choose(e.target.value)} disabled={loading}>
            {projects.map((p) => (
              <option key={p.name} value={p.name}>
                {p.name}
                {p.description ? ` — ${p.description}` : ''}
              </option>
            ))}
          </Select>
        ) : (
          <Input value={host.project ?? ''} placeholder="default" onChange={(e) => onChange({ project: e.target.value.trim() })} onBlur={(e) => choose(e.target.value.trim() || 'default')} />
        )}
      </Field>
      {error && <div className="mt-1 text-xs text-danger">{error}</div>}
      {info && <ProjectNotes project={info} />}
    </div>
  )
}

function ProjectNotes({ project }: { project: ProjectInfo }) {
  const notes: { tone: 'warn' | 'info'; text: string }[] = []
  if (project.name !== 'default' && !project.features_networks)
    notes.push({ tone: 'warn', text: "This project shares the default project's networks (features.networks is off), so team networks and network ACLs would still be created in default." })
  if (project.name !== 'default' && project.features_profiles)
    notes.push({ tone: 'info', text: "This project has its own profiles. LaForge gives every instance a root disk on the storage pool below, but the docker base image build also needs this project's default profile to have a network device with internet access." })
  if (project.restricted)
    notes.push({ tone: 'info', text: 'This project is restricted. It must allow nested containers, the uplink below, proxy devices, and low-level VM settings -- see the MicroCloud builder guide.' })
  if (notes.length === 0) return null
  return (
    <ul className="mt-2 flex flex-col gap-1 text-xs">
      {notes.map((n) => (
        <li key={n.text} className={n.tone === 'warn' ? 'text-warning' : 'text-fg-muted'}>
          {n.text}
        </li>
      ))}
    </ul>
  )
}
