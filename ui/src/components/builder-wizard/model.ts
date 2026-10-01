import type {
  BuilderConfig,
  BuilderConfigRequest,
  BuilderConnection,
  ImageInfo,
  IncusHostConfig,
  IncusImageRef,
  IncusSizeSpec,
  NetworkInfo,
} from '../../api/types'

// The builder workflow's own state. A draft is built step by step and only
// turned into a BuilderConfigRequest at the end -- nothing is saved until
// the admin reviews it.

export type Kind = 'microcloud' | 'incus' | 'fake' | 'aws' | 'openstack'

// The cloud kinds share one wizard path: no host connection or placement,
// just an image map (os -> AMI/Glance id) and a size map (size -> instance
// type/flavor). Credentials and region come from the runner's environment.
export const isCloud = (k: Kind): boolean => k === 'aws' || k === 'openstack'

export interface HostDraft {
  key: string
  credentialId?: string
  connection?: BuilderConnection
  // A host set up with cert files before token enrollment existed. Kept
  // working as-is; the admin can switch it to a token at any time.
  legacy?: IncusHostConfig
  storagePool: string
  uplink: string
  timeoutSeconds: number
}

// One image this builder offers: the name content uses for it (a host's os
// or a container's image) and the exact image on the hosts, by fingerprint.
export interface ImageDraft {
  name: string
  fingerprint: string
  alias: string
  vm: boolean
  // Set only for older configs that pull by alias from a remote server.
  server: string
}

export interface SizeDraft {
  name: string
  cpu: string
  memory: string
  // Cloud instance type / flavor (e.g. "t3.medium"); used only by the cloud
  // kinds, where cpu/memory don't apply.
  type?: string
}

export interface Draft {
  name: string
  kind: Kind
  hosts: HostDraft[]
  images: ImageDraft[]
  sizes: SizeDraft[]
  // External access (Incus/MicroCloud): the single external IP content `public:`
  // ports are NAT'd in on, and the external-port window allocated on it. Empty
  // IP = external access off; 0 ports = builder defaults.
  externalAccessIp: string
  externalPortMin?: number
  externalPortMax?: number
}

let seq = 0
export const newKey = () => `host-${++seq}`

export const emptyHost = (): HostDraft => ({ key: newKey(), storagePool: '', uplink: '', timeoutSeconds: 120 })

export const CPU_OPTIONS = ['1', '2', '4', '8', '16']
export const MEMORY_OPTIONS = ['512MiB', '1GiB', '2GiB', '4GiB', '8GiB', '16GiB', '32GiB']
export const TIMEOUT_OPTIONS = [60, 120, 300, 600, 1200]

export const KIND_LABEL: Record<Kind, string> = {
  microcloud: 'MicroCloud cluster',
  incus: 'Incus hosts',
  fake: 'Simulated',
  aws: 'AWS (EC2)',
  openstack: 'OpenStack',
}

// The standard sizes a new builder starts with.
export const DEFAULT_SIZES: SizeDraft[] = [
  { name: 'tiny', cpu: '1', memory: '1GiB' },
  { name: 'small', cpu: '1', memory: '2GiB' },
  { name: 'medium', cpu: '2', memory: '4GiB' },
  { name: 'large', cpu: '4', memory: '8GiB' },
  { name: 'x-large', cpu: '8', memory: '16GiB' },
]

export function defaultSize(name: string): { cpu: string; memory: string } {
  return DEFAULT_SIZES.find((d) => d.name === name) ?? { cpu: '2', memory: '4GiB' }
}

export function imageLabel(img: ImageInfo): string {
  const alias = img.aliases[0] ?? img.fingerprint.slice(0, 12)
  const desc = [img.properties.os, img.properties.release].filter(Boolean).join(' ') || img.properties.description
  return desc ? `${alias} — ${desc}` : alias
}

export function uplinkCandidates(networks: NetworkInfo[]): NetworkInfo[] {
  return networks.filter((n) => n.type === 'physical' || n.type === 'bridge')
}

// Sensible first choices once a server is connected: its "default" pool and
// a network named UPLINK (the MicroCloud convention), else the only
// physical network.
export function placementDefaults(conn: BuilderConnection): { storagePool: string; uplink: string } {
  const pools = conn.discovery.storage_pools
  const storagePool = pools.find((p) => p.name === 'default')?.name ?? pools[0]?.name ?? ''
  const candidates = uplinkCandidates(conn.discovery.networks)
  const physical = candidates.filter((n) => n.type === 'physical')
  const uplink =
    candidates.find((n) => n.name.toLowerCase() === 'uplink')?.name ?? (physical.length === 1 ? physical[0].name : '')
  return { storagePool, uplink }
}

// Every image on every connected host, deduplicated, with which hosts have
// it -- for a pool, an image missing from some hosts is worth a warning.
export interface AvailableImage {
  image: ImageInfo
  alias: string
  hostKeys: string[]
}

export function availableImages(hosts: HostDraft[]): AvailableImage[] {
  const byFingerprint = new Map<string, AvailableImage>()
  for (const h of hosts) {
    for (const img of h.connection?.discovery.images ?? []) {
      const existing = byFingerprint.get(img.fingerprint)
      if (existing) existing.hostKeys.push(h.key)
      else byFingerprint.set(img.fingerprint, { image: img, alias: img.aliases[0] ?? '', hostKeys: [h.key] })
    }
  }
  return [...byFingerprint.values()].sort((a, b) => (a.alias || '~').localeCompare(b.alias || '~'))
}

// The name an image is offered under by default: its alias, else its OS
// and release, else a short fingerprint.
export function defaultImageName(img: ImageInfo): string {
  if (img.aliases[0]) return img.aliases[0]
  const slug = [img.properties.os, img.properties.release]
    .filter(Boolean)
    .join('-')
    .toLowerCase()
    .replace(/[^a-z0-9.-]+/g, '-')
    .replace(/^-+|-+$/g, '')
  return slug || img.fingerprint.slice(0, 12)
}

export function isHostConnected(h: HostDraft): boolean {
  return !!h.connection || !!h.legacy
}

export function isHostPlaced(h: HostDraft): boolean {
  return h.storagePool !== '' && h.uplink !== ''
}

export function imagesFromConfig(existing: Record<string, IncusImageRef>): ImageDraft[] {
  return Object.entries(existing)
    .map(([name, ref]) => ({ name, fingerprint: ref.fingerprint ?? '', alias: ref.alias, vm: ref.vm, server: ref.server ?? '' }))
    .sort((a, b) => a.name.localeCompare(b.name))
}

export function sizesFromConfig(existing: Record<string, IncusSizeSpec>): SizeDraft[] {
  const rows = Object.entries(existing).map(([name, s]) => ({ name, cpu: s.cpu, memory: s.memory, type: s.type }))
  return rows.length ? rows : DEFAULT_SIZES.map((d) => ({ ...d }))
}

export function draftFromConfig(bc: BuilderConfig): Draft {
  const hosts: HostDraft[] = []
  if (bc.kind === 'microcloud') {
    const h: HostDraft = {
      key: newKey(),
      storagePool: bc.incus_storage_pool ?? '',
      uplink: bc.incus_ovn_uplink_network ?? '',
      timeoutSeconds: bc.incus_operation_timeout_seconds ?? 120,
    }
    if (bc.incus_credential_id) h.credentialId = bc.incus_credential_id
    else if (bc.incus_api_url)
      h.legacy = {
        api_url: bc.incus_api_url,
        client_cert_path: bc.incus_client_cert_path ?? '',
        client_key_path: bc.incus_client_key_path ?? '',
        server_cert_pem: bc.incus_server_cert_pem ?? '',
      }
    hosts.push(h)
  } else if (bc.kind === 'incus') {
    for (const hc of bc.incus_hosts) {
      hosts.push({
        key: newKey(),
        credentialId: hc.credential_id || undefined,
        legacy: hc.credential_id ? undefined : hc,
        storagePool: hc.storage_pool ?? '',
        uplink: hc.ovn_uplink_network ?? '',
        timeoutSeconds: hc.operation_timeout_seconds ?? 120,
      })
    }
  }
  return {
    name: bc.name,
    kind: bc.kind,
    hosts,
    images: [],
    sizes: [],
    externalAccessIp: bc.external_access_ip ?? '',
    externalPortMin: bc.external_port_min ?? undefined,
    externalPortMax: bc.external_port_max ?? undefined,
  }
}

export function toRequest(d: Draft): BuilderConfigRequest {
  if (d.kind === 'fake') return { kind: 'fake' }

  if (isCloud(d.kind)) {
    // Cloud kinds reuse the shared image/size columns: the image id (AMI /
    // Glance id) rides in the ImageRef fingerprint, the instance type/flavor
    // in the SizeSpec type. Credentials and region come from the runner env.
    const images: Record<string, IncusImageRef> = {}
    for (const img of d.images) if (img.name && img.fingerprint) images[img.name] = { fingerprint: img.fingerprint, alias: '', server: '', protocol: '', vm: false }
    const sizes: Record<string, IncusSizeSpec> = {}
    for (const s of d.sizes) if (s.name && s.type) sizes[s.name] = { cpu: '', memory: '', type: s.type }
    return { kind: d.kind, incus_images: images, incus_sizes: sizes }
  }

  const incus_images: Record<string, IncusImageRef> = {}
  for (const img of d.images) {
    incus_images[img.name] = {
      fingerprint: img.fingerprint || undefined,
      alias: img.alias,
      server: img.server,
      protocol: img.server ? 'simplestreams' : '',
      vm: img.vm,
    }
  }
  const incus_sizes: Record<string, IncusSizeSpec> = {}
  for (const s of d.sizes) incus_sizes[s.name] = { cpu: s.cpu, memory: s.memory }

  // Builder-level external access -- shared across the whole builder, not per host.
  const externalAccess = {
    external_access_ip: d.externalAccessIp || undefined,
    external_port_min: d.externalPortMin || undefined,
    external_port_max: d.externalPortMax || undefined,
  }

  if (d.kind === 'microcloud') {
    const h = d.hosts[0]
    const placement = {
      incus_storage_pool: h.storagePool,
      incus_ovn_uplink_network: h.uplink,
      incus_operation_timeout_seconds: h.timeoutSeconds,
    }
    if (h.credentialId) return { kind: 'microcloud', incus_credential_id: h.credentialId, ...placement, ...externalAccess, incus_images, incus_sizes }
    return {
      kind: 'microcloud',
      incus_api_url: h.legacy?.api_url,
      incus_client_cert_path: h.legacy?.client_cert_path,
      incus_client_key_path: h.legacy?.client_key_path,
      incus_server_cert_pem: h.legacy?.server_cert_pem,
      ...placement,
      ...externalAccess,
      incus_images,
      incus_sizes,
    }
  }

  const incus_hosts: IncusHostConfig[] = d.hosts.map((h) => {
    const placement = { storage_pool: h.storagePool, ovn_uplink_network: h.uplink, operation_timeout_seconds: h.timeoutSeconds }
    if (h.credentialId)
      return { credential_id: h.credentialId, api_url: '', client_cert_path: '', client_key_path: '', server_cert_pem: '', ...placement }
    return { ...(h.legacy as IncusHostConfig), ...placement }
  })
  return { kind: 'incus', incus_hosts, ...externalAccess, incus_images, incus_sizes }
}

export function shortFingerprint(fp: string): string {
  return (fp.match(/.{1,4}/g) ?? []).slice(0, 4).join(' ')
}
