import { AlertTriangle, RefreshCw, Trash2 } from 'lucide-react'
import { Badge, Button, Input, Spinner, cn } from '../../ui'
import { defaultImageName, shortFingerprint, type AvailableImage, type HostDraft, type ImageDraft, type Kind } from './model'

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

  const selectedByFp = new Map(images.filter((i) => i.fingerprint).map((i) => [i.fingerprint, i]))
  const offline = images.filter((i) => !i.fingerprint || !available.some((a) => a.image.fingerprint === i.fingerprint))

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

      {connected.length > 0 && available.length === 0 && (
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
