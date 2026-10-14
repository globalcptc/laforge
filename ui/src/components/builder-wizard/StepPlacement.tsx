import { useState } from 'react'
import { PublicNICSettings } from './PublicNICSettings'
import type { MicrocloudPublicAccess } from '../../api/types'
import { AlertTriangle, Boxes, FolderTree, Gauge, Globe, HardDrive, Network, Plus, Server, X } from 'lucide-react'
import { api, ApiError } from '../../api/client'
import { builderConnectionPath } from '../../api/hooks'
import type { BuilderConnection, ProjectInfo } from '../../api/types'
import { Button, Field, Input, Select, Slider, Spinner } from '../../ui'
import { TIMEOUT_OPTIONS, uplinkCandidates, type HostDraft, type Kind } from './model'

export function StepPlacement({
  kind,
  oneTeamPerHost,
  onPlacementPolicyChange,
  hosts,
  onChange,
  externalAccessIp,
  externalPortMin,
  externalPortMax,
  onExternalChange,
  microcloudPublicAccess,
  onPublicAccessChange,
  clusterGroup,
  onClusterGroupChange,
  maxConcurrency,
  onMaxConcurrencyChange,
}: {
  kind: Kind
  oneTeamPerHost: boolean
  onPlacementPolicyChange: (value: boolean) => void
  microcloudPublicAccess?: MicrocloudPublicAccess
  onPublicAccessChange: (value: MicrocloudPublicAccess) => void
  clusterGroup: string
  onClusterGroupChange: (value: string) => void
  maxConcurrency: number
  onMaxConcurrencyChange: (value: number) => void
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
      {kind === 'incus' && <div className="card p-4">
        <label className="flex items-center gap-2 text-sm font-medium">
          <input type="checkbox" checked={oneTeamPerHost} onChange={(e) => onPlacementPolicyChange(e.target.checked)} />
          One team per configured host
        </label>
        <p className="mt-2 text-xs text-fg-muted">Each team in a build gets a distinct host, in the order listed below. Builds with more teams than hosts are rejected before deployment. Native bridges also require distinct hosts for repeated team subnets.</p>
      </div>}
      {hosts.map((h, i) => (
        <div key={h.key} className="card p-4">
          <div className="mb-3 flex items-center gap-2 text-sm font-semibold text-fg">
            <Server size={14} className="text-fg-muted" />
            {kind === 'incus' ? `Host ${i + 1}` : 'Cluster'}
            {h.connection && <span className="font-normal text-fg-muted">· {h.connection.credential.server_name || h.connection.credential.api_url}</span>}
          </div>

          {(h.connection?.discovery.warnings ?? []).map((w) => (
            <div key={w} className="mb-3 flex items-start gap-2 rounded-token border border-warning/30 bg-warning-soft px-3 py-2 text-xs text-warning">
              <AlertTriangle size={13} className="mt-0.5 shrink-0" />
              {w}
            </div>
          ))}

          {kind === 'microcloud' && h.connection && h.credentialId && (
            <ProjectPicker host={h} onChange={(patch) => update(i, patch)} />
          )}
          {kind === 'microcloud' && (!h.connection || !h.credentialId) && (
            <Field label="Project" hint="The existing LXD project used for this builder's instances and networks. Empty uses default." className="mb-4 max-w-sm">
              <Input value={h.project ?? ''} onChange={(e) => update(i, { project: e.target.value.trim() })} placeholder="default" />
            </Field>
          )}

          {kind === 'incus' && <Field label="Deployment networking" hint="Choose how this host connects the VMs in each deployment subnet.">
            <Select value={h.networkType ?? 'bridge'} onChange={(e) => update(i, { networkType: e.target.value as 'bridge' | 'ovn' })}>
              <option value="bridge">Native bridges — host routing and NAT</option>
              <option value="ovn">OVN — isolated logical networks</option>
            </Select>
            {(h.networkType ?? 'bridge') === 'bridge' && <BridgeNotes host={h} />}
          </Field>}
          {h.connection || kind === 'microcloud' ? (
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
<PoolInput host={h} onChange={(storagePool) => update(i, { storagePool })} />
              </Field>
              {(kind === 'microcloud' || (h.networkType ?? 'bridge') !== 'bridge') && (<Field
                label={
                  <span className="flex items-center gap-1.5">
                    <Network size={12} /> OVN uplink network
                  </span>
                }
                hint="A managed physical or bridge network prepared as an OVN uplink. This does not change the deployment network type."
                className="mb-0"
              >
<UplinkInput host={h} onChange={(uplink) => update(i, { uplink })} />
              </Field>)}
            </div>
          ) : (
            <div className="text-sm text-fg-muted">
              This host uses its saved settings -- storage pool <span className="font-mono text-fg">{h.storagePool || '(default)'}</span>, uplink{' '}
              <span className="font-mono text-fg">{(h.networkType ?? 'bridge') === 'bridge' ? 'native bridges (no uplink needed)' : h.uplink || '(none)'}</span>. Connect it with a token to choose from the server's own list.
            </div>
          )}

          <div className="mt-4 max-w-sm">
            <Field label="API and operation timeout" hint={kind === 'microcloud' ? 'Per request, including image and snapshot checks when saving. Defaults to 10 minutes for slow clusters and large copies.' : 'Per request, including image and snapshot checks when saving and provisioning operations.'} className="mb-0">
              <Select value={String(h.timeoutSeconds)} onChange={(e) => update(i, { timeoutSeconds: Number(e.target.value) })}>
                {[...new Set([...TIMEOUT_OPTIONS, h.timeoutSeconds])].sort((a, b) => a - b).map((s) => (
                  <option key={s} value={s}>
                    {s < 60 ? `${s} seconds` : `${s / 60} minute${s === 60 ? '' : 's'}`}
                  </option>
                ))}
              </Select>
            </Field>
          </div>
        </div>
      ))}

      {/* MicroCloud cluster group: LaForge targets this group and lets LXD
          schedule across its members -- it never picks a member itself. */}
      {kind === 'microcloud' && (
        <div className="card p-4">
          <div className="mb-1 flex items-center gap-2 text-sm font-semibold text-fg">
            <Boxes size={14} className="text-fg-muted" /> Cluster Group
          </div>
          <p className="mb-3 text-xs text-fg-muted">
            The LXD cluster group LaForge deploys into. MicroCloud/LXD schedules each instance across the group's members — LaForge does
            <span className="font-medium text-fg"> not</span> pick a member or balance placement itself. <span className="font-mono">default</span> is
            the whole cluster; use a named group to confine deploys to a subset.
          </p>
          <Input
            value={clusterGroup}
            onChange={(e) => onClusterGroupChange(e.target.value)}
            placeholder="default"
            className="font-mono"
          />
        </div>
      )}

      {/* MicroCloud deploy concurrency: cap how many of this builder's hosts
          deploy at once, independent of the global runner pool. */}
      {kind === 'microcloud' && (
        <div className="card p-4">
          <div className="mb-1 flex items-center gap-2 text-sm font-semibold text-fg">
            <Gauge size={14} className="text-fg-muted" /> Deploy Concurrency
          </div>
          <p className="mb-3 text-xs text-fg-muted">
            The most hosts LaForge deploys on this cluster at once. Turn it down if parallel deploys overwhelm the cluster; the global runner
            pool still caps everything above this. <span className="font-medium text-fg">Unlimited</span> uses the full pool (the previous behavior).
          </p>
          <Slider
            min={0}
            max={50}
            step={1}
            value={maxConcurrency}
            onChange={onMaxConcurrencyChange}
            valueLabel={(v) => (v === 0 ? 'Unlimited' : `${v} thread${v === 1 ? '' : 's'}`)}
          />
        </div>
      )}

      {/* Builder-level external access -- one or more IPs, assigned one per team
          round-robin, with a port window; a host's public: ports get NAT'd in. */}
      <div className="card p-4">
        <div className="mb-1 flex items-center gap-2 text-sm font-semibold text-fg">
          <Globe size={14} className="text-fg-muted" /> External access
        </div>
        {kind === 'microcloud' && <Field label="Public access type" hint="Applies to this builder's hosts that declare public ports.">
          <Select value={microcloudPublicAccess?.type ?? 'proxy'} onChange={(e) => onPublicAccessChange(e.target.value === 'nic' ? { type: 'nic' } : { type: 'proxy' })}>
            <option value="proxy">Shared IP port forwarding</option>
            <option value="nic">Separate network NIC</option>
          </Select>
        </Field>}
        {kind === 'microcloud' && microcloudPublicAccess?.type === 'nic' ? (
          <PublicNICSettings value={microcloudPublicAccess} onChange={onPublicAccessChange} />
        ) : <>
        <p className="mb-3 text-xs text-fg-muted">
          The external IPs that hosts' <span className="font-mono">public:</span> ports (e.g. RDP) are NAT'd in on. Each team is assigned one IP,
          round-robin, so with enough IPs every team gets its own address; teams that share one get a distinct external port each. Add no IPs to disable
          external access for this builder.
          {kind === 'incus' && " For one team per host, list one external IP per host in the same order as the hosts above. Each address must reach its assigned host."}
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
        </>}
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
      {info?.detailed !== false && info && <ProjectNotes project={info} />}
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

// UplinkInput picks the network team networks route out through. On a shared
// cluster with hundreds of networks only some are read in full, so this is a
// text field that suggests every name (types shown where known) rather than a
// list that would have to describe them all.
function UplinkInput({ host, onChange }: { host: HostDraft; onChange: (uplink: string) => void }) {
  const disc = host.connection?.discovery
  const known = new Map((disc?.networks ?? []).map((n) => [n.name, n]))
  const names = disc?.network_names?.length ? disc.network_names : (disc?.networks ?? []).map((n) => n.name)
  const candidates = uplinkCandidates(disc?.networks ?? [])
  const listId = `uplinks-${host.key}`
  const chosen = known.get(host.uplink)
  return (
    <div>
      <Input value={host.uplink} onChange={(e) => onChange(e.target.value.trim())} list={listId} placeholder="UPLINK" className="font-mono" />
      <datalist id={listId}>
        {[...candidates.map((n) => n.name), ...names.filter((n) => !candidates.some((c) => c.name === n))].map((n) => (
          <option key={n} value={n}>
            {networkLabel(known.get(n))}
          </option>
        ))}
      </datalist>
      {candidates.length > 0 && (
        <div className="mt-1 flex flex-wrap items-center gap-1 text-xs text-fg-subtle">
          Likely uplinks:
          {candidates.map((n) => (
            <button key={n.name} type="button" onClick={() => onChange(n.name)} className="rounded bg-surface-sunken px-1.5 py-0.5 font-mono text-fg hover:bg-surface-hover">
              {n.name} · {networkLabel(n)}
            </button>
          ))}
        </div>
      )}
      {host.uplink && chosen && chosen.type !== 'physical' && chosen.type !== 'bridge' && (
        <div className="mt-1 text-xs text-warning">
          {host.uplink} is a {chosen.type} network; choose a managed physical or bridge network.
        </div>
      )}
      {host.uplink && names.length > 0 && !names.includes(host.uplink) && (
        <div className="mt-1 text-xs text-warning">No network named {host.uplink} was found on the server.</div>
      )}
    </div>
  )
}

// PoolInput picks the storage pool. Pool names are always listed (cheaply);
// drivers only where the pool's details could be read in time, so it's a
// text field with suggestions rather than a list that needs them all.
function PoolInput({ host, onChange }: { host: HostDraft; onChange: (pool: string) => void }) {
  const pools = host.connection?.discovery.storage_pools ?? []
  const listId = `pools-${host.key}`
  return (
    <div>
      <Input value={host.storagePool} onChange={(e) => onChange(e.target.value.trim())} list={listId} placeholder="default" className="font-mono" />
      <datalist id={listId}>
        {pools.map((p) => (
          <option key={p.name} value={p.name}>
            {p.driver}
          </option>
        ))}
      </datalist>
      {pools.length > 0 && (
        <div className="mt-1 flex flex-wrap items-center gap-1 text-xs text-fg-subtle">
          On the server:
          {pools.map((p) => (
            <button key={p.name} type="button" onClick={() => onChange(p.name)} className="rounded bg-surface-sunken px-1.5 py-0.5 font-mono text-fg hover:bg-surface-hover">
              {p.name}
              {p.driver && <span className="text-fg-subtle"> ({p.driver})</span>}
            </button>
          ))}
        </div>
      )}
      {host.storagePool && pools.length > 0 && !pools.some((p) => p.name === host.storagePool) && (
        <div className="mt-1 text-xs text-warning">No storage pool named {host.storagePool} was found on the server.</div>
      )}
    </div>
  )
}

function networkLabel(network: import('../../api/types').NetworkInfo | undefined): string {
  if (!network) return 'type unknown'
  return [network.type, network.managed ? 'managed' : 'host interface', network.config?.['ipv4.address']].filter(Boolean).join(' · ')
}

function BridgeNotes({ host }: { host: HostDraft }) {
  const discovery = host.connection?.discovery
  const bridges = (discovery?.networks ?? []).filter((n) => n.managed && n.type === 'bridge' && (!n.config?.['bridge.driver'] || n.config['bridge.driver'] === 'native'))
  return <div className="mt-2 text-xs text-fg-muted">
    <p>Incus creates a private bridge for each deployment subnet. The host provides routing and NAT through its existing default route; no uplink NIC is required.</p>
    {discovery?.native_bridge_supported === false
      ? <p className="mt-1 text-danger">This host lacks bridge NIC ACL support with nftables. Upgrade Incus or choose OVN.</p>
      : discovery?.native_bridge_supported === true
        ? <p className="mt-1">Detected native bridge and NIC firewall support.</p>
        : <p className="mt-1">Bridge NIC firewall support will be checked before network creation.</p>}
    {bridges.length > 0 && <p className="mt-1">Existing bridges: {bridges.map((n) => `${n.name} (${networkLabel(n)})`).join(', ')}. Deployment subnets must not overlap these networks.</p>}
  </div>
}
