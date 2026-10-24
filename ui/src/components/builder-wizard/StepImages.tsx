import { useState } from 'react'
import { AlertTriangle, Camera, Plus, RefreshCw, Trash2 } from 'lucide-react'
import { Badge, Button, Input, Spinner, cn } from '../../ui'
import {
  availableSnapshots,
  defaultImageName,
  isSnapshotDraft,
  shortFingerprint,
  snapshotKey,
  type AvailableImage,
  type AvailableSnapshot,
  type HostDraft,
  type ImageDraft,
  type Kind,
} from './model'

// The images this builder offers are the images already on its hosts, read
// from each host's API. With more than one host, an image is only offerable
// once every host has it (same fingerprint) -- a team can land on any host.
export function StepImages({
  kind,
  images,
  available,
  hosts,
  refreshing,
  onRefresh,
  onChange,
}: {
  kind: Kind
  images: ImageDraft[]
  available: AvailableImage[]
  hosts: HostDraft[]
  refreshing: boolean
  onRefresh: () => void
  onChange: (images: ImageDraft[]) => void
}) {
  const connected = hosts.filter((h) => h.connection)
  const unreadable = hosts.length - connected.length
  const multi = hosts.length > 1

  const selectedByFp = new Map(images.filter((i) => !isSnapshotDraft(i) && i.fingerprint).map((i) => [i.fingerprint, i]))
  const offline = images.filter((i) => !isSnapshotDraft(i) && (!i.fingerprint || !available.some((a) => a.image.fingerprint === i.fingerprint)))
  const snaps = availableSnapshots(hosts)
  const selectedBySnap = new Map(images.filter(isSnapshotDraft).map((i) => [snapshotKey(i.instance ?? '', i.snapshot ?? '', i.sourceProject ?? ''), i]))
  // Snapshots offered but not in the list read from the hosts: in another
  // project (never listed here), or gone. Either way they're checked on save.
  const unlistedSnaps = images.filter(
    (i) => isSnapshotDraft(i) && !snaps.some((s) => !i.sourceProject && s.snapshot.instance === i.instance && s.snapshot.snapshot === i.snapshot),
  )

  function uniqueName(base: string): string {
    const taken = new Set(images.map((i) => i.name))
    if (!taken.has(base)) return base
    for (let n = 2; ; n++) if (!taken.has(`${base}-${n}`)) return `${base}-${n}`
  }

  function toggle(a: AvailableImage, on: boolean) {
    if (on) {
      onChange([
        ...images,
        {
          name: uniqueName(defaultImageName(a.image)),
          fingerprint: a.image.fingerprint,
          alias: a.alias,
          vm: a.image.type === 'virtual-machine',
          server: '',
        },
      ])
    } else {
      onChange(images.filter((i) => i.fingerprint !== a.image.fingerprint))
    }
  }

  function toggleSnapshot(s: AvailableSnapshot, on: boolean) {
    const key = snapshotKey(s.snapshot.instance, s.snapshot.snapshot)
    if (on) {
      onChange([
        ...images,
        {
          name: uniqueName(s.snapshot.instance),
          fingerprint: '',
          alias: '',
          server: '',
          vm: s.snapshot.type === 'virtual-machine',
          source: 'snapshot',
          instance: s.snapshot.instance,
          snapshot: s.snapshot.snapshot,
          sourceProject: '',
        },
      ])
    } else {
      onChange(images.filter((i) => !(isSnapshotDraft(i) && snapshotKey(i.instance ?? '', i.snapshot ?? '', i.sourceProject ?? '') === key)))
    }
  }

  const rename = (target: ImageDraft, name: string) => onChange(images.map((i) => (i === target ? { ...i, name } : i)))
  const dupNames = new Set(images.map((i) => i.name).filter((n, idx, all) => all.indexOf(n) !== idx))

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-start justify-between gap-4">
        <p className="text-sm text-fg-muted">
          Choose which images on {multi ? 'the hosts' : 'the server'} this builder offers, and the name content uses to ask for each one (a
          host's <span className="font-mono">os</span> or a container's <span className="font-mono">image</span>).
          {multi && ' An image has to be on every host before it can be offered.'}
        </p>
        <Button variant="secondary" size="sm" onClick={onRefresh} disabled={refreshing || connected.length === 0} className="shrink-0">
          {refreshing ? <Spinner /> : <RefreshCw size={12} />} Refresh
        </Button>
      </div>

      {unreadable > 0 && (
        <div className="flex items-start gap-2 rounded-token border border-warning/30 bg-warning-soft px-3 py-2 text-sm text-warning">
          <AlertTriangle size={16} className="mt-0.5 shrink-0" />
          <span>
            {unreadable === hosts.length ? 'No host is' : `${unreadable} of ${hosts.length} hosts aren't`} connected with a trust token, so
            {unreadable === hosts.length ? ' its' : ' their'} images can't be read. Connect {unreadable === 1 ? 'it' : 'them'} on the Connect
            step.
          </span>
        </div>
      )}

      {connected.length > 0 && available.length === 0 && snaps.length === 0 && (
        <div className="rounded-token border border-dashed border-border px-4 py-4 text-sm text-fg-muted">
          <div className="font-medium text-fg">No images on {multi ? 'the hosts' : 'the server'} yet</div>
          <div className="mt-1">
            Import or copy the images you need onto {multi ? 'every host' : kind === 'microcloud' ? 'the cluster' : 'the server'} (for example{' '}
            <span className="font-mono">
              {kind === 'microcloud' ? 'lxc image copy ubuntu:24.04 local: --alias ubuntu-24.04' : 'incus image copy images:debian/12 local: --alias debian-12'}
            </span>
            ), then refresh.
          </div>
        </div>
      )}

      {available.length > 0 && (
        <div className="card divide-y divide-border">
          {available.map((a) => {
            const sel = selectedByFp.get(a.image.fingerprint)
            const missing = hosts.map((h, i) => ({ h, i })).filter(({ h }) => h.connection && !a.hostKeys.includes(h.key))
            const offerable = missing.length === 0
            const desc =
              a.image.properties.description || [a.image.properties.os, a.image.properties.release].filter(Boolean).join(' ') || 'No description'
            return (
              <div key={a.image.fingerprint} className={cn('flex flex-col gap-2 px-3 py-2.5', !offerable && !sel && 'opacity-70')}>
                <div className="flex items-start gap-3">
                  <input
                    type="checkbox"
                    className="mt-1"
                    checked={!!sel}
                    disabled={!offerable && !sel}
                    onChange={(e) => toggle(a, e.target.checked)}
                    aria-label={`Offer ${a.alias || a.image.fingerprint}`}
                  />
                  <div className="min-w-0 flex-1">
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="font-mono text-xs text-fg">{a.alias || '(no alias)'}</span>
                      <Badge tone={a.image.type === 'virtual-machine' ? 'info' : 'neutral'}>
                        {a.image.type === 'virtual-machine' ? 'VM' : 'Container'}
                      </Badge>
                      {a.image.aliases.length > 1 && <span className="text-xs text-fg-subtle">also {a.image.aliases.slice(1).join(', ')}</span>}
                    </div>
                    <div className="mt-0.5 text-xs text-fg-muted">
                      {desc} · <span className="font-mono">{shortFingerprint(a.image.fingerprint)}</span>
                    </div>
                    {multi && (
                      <div className="mt-1.5 flex flex-wrap gap-1">
                        {hosts.map((h, i) => {
                          const has = a.hostKeys.includes(h.key)
                          return (
                            <Badge key={h.key} tone={!h.connection ? 'neutral' : has ? 'success' : 'danger'}>
                              Host {i + 1} {!h.connection ? 'unknown' : has ? 'has it' : 'missing'}
                            </Badge>
                          )
                        })}
                      </div>
                    )}
                    {!offerable && (
                      <div className="mt-1 flex items-center gap-1 text-xs text-warning">
                        <AlertTriangle size={11} /> Not on {missing.map(({ i }) => `host ${i + 1}`).join(', ')}. Copy it there, then refresh
                        {sel ? ' -- saving is refused until then.' : '.'}
                      </div>
                    )}
                  </div>
                  {sel && (
                    <div className="w-60 shrink-0">
                      <Input
                        value={sel.name}
                        onChange={(e) => rename(sel, e.target.value.trim())}
                        className={cn('h-8 font-mono text-xs', (!sel.name || dupNames.has(sel.name)) && 'border-danger')}
                        aria-label="Name content uses"
                        placeholder="name content uses"
                      />
                      {dupNames.has(sel.name) && <div className="mt-0.5 text-xs text-danger">Another image uses this name.</div>}
                    </div>
                  )}
                </div>
              </div>
            )
          })}
        </div>
      )}

      <div>
        <div className="mb-1 flex items-center gap-1.5 text-xs font-semibold uppercase tracking-wide text-fg-muted">
          <Camera size={12} /> Snapshots
        </div>
        <div className="mb-2 text-xs text-fg-subtle">
          Instead of an image, an instance can be a copy of a stopped template's snapshot -- on storage like Ceph a copy is a fast clone. Each copy
          gets only LaForge's own network, disk and config drive, never the template's other devices. The template needs cloud-init (or
          cloudbase-init on Windows) so the copy runs LaForge's setup on first boot.
        </div>
        {snaps.length > 0 ? (
          <div className="card divide-y divide-border">
            {snaps.map((s) => {
              const key = snapshotKey(s.snapshot.instance, s.snapshot.snapshot)
              const sel = selectedBySnap.get(key)
              const missing = hosts.map((h, i) => ({ h, i })).filter(({ h }) => h.connection && !s.hostKeys.includes(h.key))
              const offerable = missing.length === 0
              const desc = s.snapshot.description || [s.snapshot.os, s.snapshot.release].filter(Boolean).join(' ')
              return (
                <div key={key} className={cn('flex items-start gap-3 px-3 py-2.5', !offerable && !sel && 'opacity-70')}>
                  <input
                    type="checkbox"
                    className="mt-1"
                    checked={!!sel}
                    disabled={!offerable && !sel}
                    onChange={(e) => toggleSnapshot(s, e.target.checked)}
                    aria-label={`Offer copies of ${s.snapshot.instance}/${s.snapshot.snapshot}`}
                  />
                  <div className="min-w-0 flex-1">
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="font-mono text-xs text-fg">
                        {s.snapshot.instance}/{s.snapshot.snapshot}
                      </span>
                      <Badge tone={s.snapshot.type === 'virtual-machine' ? 'info' : 'neutral'}>{s.snapshot.type === 'virtual-machine' ? 'VM' : 'Container'}</Badge>
                      {s.snapshot.status !== 'Stopped' && <Badge tone="warning">template is {s.snapshot.status.toLowerCase()}</Badge>}
                    </div>
                    <div className="mt-0.5 text-xs text-fg-muted">
                      {desc || 'No description'}
                      {s.snapshot.created_at && ` · taken ${new Date(s.snapshot.created_at).toLocaleString()}`}
                    </div>
                    {!offerable && (
                      <div className="mt-1 flex items-center gap-1 text-xs text-warning">
                        <AlertTriangle size={11} /> Not on {missing.map(({ i }) => `host ${i + 1}`).join(', ')}.
                      </div>
                    )}
                  </div>
                  {sel && (
                    <div className="w-60 shrink-0">
                      <Input
                        value={sel.name}
                        onChange={(e) => rename(sel, e.target.value.trim())}
                        className={cn('h-8 font-mono text-xs', (!sel.name || dupNames.has(sel.name)) && 'border-danger')}
                        aria-label="Name content uses"
                        placeholder="name content uses"
                      />
                      {dupNames.has(sel.name) && <div className="mt-0.5 text-xs text-danger">Another image uses this name.</div>}
                    </div>
                  )}
                </div>
              )
            })}
          </div>
        ) : (
          connected.length > 0 && (
            <div className="rounded-token border border-dashed border-border px-4 py-3 text-xs text-fg-muted">
              No instance snapshots in {kind === 'microcloud' ? "this builder's project" : 'the server'} yet. Take one of a stopped template (for example{' '}
              <span className="font-mono">{kind === 'microcloud' ? 'lxc snapshot win-tmpl golden' : 'incus snapshot create win-tmpl golden'}</span>), then refresh -- or add
              one from another project below.
            </div>
          )
        )}
        {unlistedSnaps.length > 0 && (
          <div className="card mt-2 divide-y divide-border">
            {unlistedSnaps.map((img) => (
              <div key={img.name + img.instance + img.snapshot} className="flex items-center gap-3 px-3 py-2">
                <span className="w-48 shrink-0 font-mono text-xs text-fg">{img.name}</span>
                <span className="min-w-0 flex-1 truncate text-xs text-fg-muted">
                  copy of <span className="font-mono">{img.instance}/{img.snapshot}</span>
                  {img.sourceProject ? ` in project ${img.sourceProject}` : ' -- not found now'} · {img.vm ? 'VM' : 'Container'} · checked when you save
                </span>
                <Button
                  variant="ghost"
                  size="icon"
                  onClick={() => onChange(images.filter((i) => i !== img))}
                  title="Remove"
                  className="text-fg-muted hover:bg-danger-soft hover:text-danger"
                >
                  <Trash2 size={12} />
                </Button>
              </div>
            ))}
          </div>
        )}
        <OtherProjectSnapshot
          taken={new Set(images.map((i) => i.name))}
          onAdd={(img) => onChange([...images, img])}
        />
      </div>

      {offline.length > 0 && connected.length > 0 && (
        <div>
          <div className="mb-1 text-xs font-semibold uppercase tracking-wide text-fg-muted">Not found on the hosts</div>
          <div className="mb-2 text-xs text-fg-subtle">Offered by this builder's saved settings, but no connected host has them now.</div>
          <div className="card divide-y divide-border">
            {offline.map((img) => (
              <div key={img.name} className="flex items-center gap-3 px-3 py-2">
                <span className="w-48 shrink-0 font-mono text-xs text-fg">{img.name}</span>
                <span className="min-w-0 flex-1 truncate text-xs text-fg-muted">
                  {img.server ? `${img.alias} from ${img.server}` : img.alias || shortFingerprint(img.fingerprint)}
                </span>
                <Button
                  variant="ghost"
                  size="icon"
                  onClick={() => onChange(images.filter((i) => i !== img))}
                  title="Remove"
                  className="text-fg-muted hover:bg-danger-soft hover:text-danger"
                >
                  <Trash2 size={12} />
                </Button>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}

// OtherProjectSnapshot adds a template that lives in a different project from
// the builder's -- not listed above, so it's typed in, and checked on save.
function OtherProjectSnapshot({ taken, onAdd }: { taken: Set<string>; onAdd: (img: ImageDraft) => void }) {
  const [open, setOpen] = useState(false)
  const [project, setProject] = useState('')
  const [instance, setInstance] = useState('')
  const [snapshot, setSnapshot] = useState('')
  const [name, setName] = useState('')
  const [vm, setVm] = useState(false)
  const valid = project.trim() && instance.trim() && snapshot.trim() && name.trim() && !taken.has(name.trim())

  if (!open)
    return (
      <Button variant="ghost" size="sm" className="mt-2" onClick={() => setOpen(true)}>
        <Plus size={12} /> Add a snapshot from another project
      </Button>
    )
  return (
    <div className="mt-2 rounded-token border border-border p-3">
      <div className="grid gap-2 sm:grid-cols-4">
        <Input value={project} onChange={(e) => setProject(e.target.value)} placeholder="project" className="h-8 font-mono text-xs" />
        <Input value={instance} onChange={(e) => setInstance(e.target.value)} placeholder="instance" className="h-8 font-mono text-xs" />
        <Input value={snapshot} onChange={(e) => setSnapshot(e.target.value)} placeholder="snapshot" className="h-8 font-mono text-xs" />
        <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="name content uses" className="h-8 font-mono text-xs" />
      </div>
      <div className="mt-2 flex flex-wrap items-center gap-3">
        <label className="flex items-center gap-1.5 text-xs text-fg">
          <input type="checkbox" checked={vm} onChange={(e) => setVm(e.target.checked)} /> The template is a VM
        </label>
        {name.trim() && taken.has(name.trim()) && <span className="text-xs text-danger">Another image uses this name.</span>}
        <div className="ml-auto flex gap-2">
          <Button variant="ghost" size="sm" onClick={() => setOpen(false)}>
            Cancel
          </Button>
          <Button
            variant="secondary"
            size="sm"
            disabled={!valid}
            onClick={() => {
              onAdd({ name: name.trim(), fingerprint: '', alias: '', server: '', vm, source: 'snapshot', instance: instance.trim(), snapshot: snapshot.trim(), sourceProject: project.trim() })
              setProject('')
              setInstance('')
              setSnapshot('')
              setName('')
              setVm(false)
              setOpen(false)
            }}
          >
            <Plus size={12} /> Add
          </Button>
        </div>
      </div>
      <div className="mt-1 text-xs text-fg-subtle">LaForge's identity needs read access to that project to copy from it.</div>
    </div>
  )
}
