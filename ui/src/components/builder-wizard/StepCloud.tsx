import { Info, Plus, Trash2 } from 'lucide-react'
import { Button, Input } from '../../ui'
import type { ImageDraft, Kind, SizeDraft } from './model'

// StepCloud configures a cloud builder (AWS/OpenStack). There is no host to
// connect or place: credentials and region come from the runner's environment.
// All the config here is two maps -- a LaForge `os` name to a cloud image id
// (AMI / Glance), and a `size` name to a cloud instance type / flavor. They are
// stored in the same shared image/size columns every builder kind uses.
export function StepCloud({
  kind,
  images,
  sizes,
  onImages,
  onSizes,
}: {
  kind: Kind
  images: ImageDraft[]
  sizes: SizeDraft[]
  onImages: (images: ImageDraft[]) => void
  onSizes: (sizes: SizeDraft[]) => void
}) {
  const isAws = kind === 'aws'
  const imageIdLabel = isAws ? 'AMI id' : 'Image id'
  const imageIdPlaceholder = isAws ? 'ami-0abc123…' : 'Glance image id'
  const sizeLabel = isAws ? 'Instance type' : 'Flavor'
  const sizePlaceholder = isAws ? 't3.medium' : 'm1.medium'
  const envHint = isAws
    ? 'Credentials and region are read from the runner’s AWS environment (AWS_ACCESS_KEY_ID / AWS_SECRET_ACCESS_KEY / AWS_REGION, or an instance profile). Nothing secret is stored here.'
    : 'Credentials and region are read from the runner’s OpenStack environment (OS_AUTH_URL / OS_USERNAME / OS_PASSWORD / OS_PROJECT_NAME / OS_REGION_NAME). Nothing secret is stored here.'

  function setImage(i: number, patch: Partial<ImageDraft>) {
    onImages(images.map((img, idx) => (idx === i ? { ...img, ...patch } : img)))
  }
  function setSize(i: number, patch: Partial<SizeDraft>) {
    onSizes(sizes.map((s, idx) => (idx === i ? { ...s, ...patch } : s)))
  }

  return (
    <div className="flex flex-col gap-5">
      <div className="flex items-start gap-2 rounded-token border border-border bg-surface-sunken px-3 py-2 text-xs text-fg-muted">
        <Info size={14} className="mt-0.5 shrink-0 text-fg-subtle" />
        <span>{envHint}</span>
      </div>

      <section>
        <div className="mb-1 text-sm font-medium text-fg">Images</div>
        <div className="mb-2 text-xs text-fg-muted">
          Map each LaForge <code className="font-mono">os</code> name (as content uses it) to the cloud image to launch it from.
        </div>
        <div className="flex flex-col gap-2">
          {images.map((img, i) => (
            <div key={i} className="flex items-center gap-2">
              <Input value={img.name} onChange={(e) => setImage(i, { name: e.target.value })} placeholder="ubuntu22" className="flex-1" />
              <span className="text-fg-subtle">→</span>
              <Input value={img.fingerprint} onChange={(e) => setImage(i, { fingerprint: e.target.value })} placeholder={imageIdPlaceholder} className="flex-1 font-mono" aria-label={imageIdLabel} />
              <Button variant="ghost" size="icon" onClick={() => onImages(images.filter((_, idx) => idx !== i))} aria-label="Remove image">
                <Trash2 size={14} />
              </Button>
            </div>
          ))}
          <div>
            <Button variant="secondary" size="sm" onClick={() => onImages([...images, { name: '', fingerprint: '', alias: '', vm: false, server: '' }])}>
              <Plus size={14} /> Add image
            </Button>
          </div>
        </div>
      </section>

      <section>
        <div className="mb-1 text-sm font-medium text-fg">Sizes</div>
        <div className="mb-2 text-xs text-fg-muted">
          Map each LaForge <code className="font-mono">size</code> name to a {sizeLabel.toLowerCase()}.
        </div>
        <div className="flex flex-col gap-2">
          {sizes.map((s, i) => (
            <div key={i} className="flex items-center gap-2">
              <Input value={s.name} onChange={(e) => setSize(i, { name: e.target.value })} placeholder="small" className="flex-1" />
              <span className="text-fg-subtle">→</span>
              <Input value={s.type ?? ''} onChange={(e) => setSize(i, { type: e.target.value })} placeholder={sizePlaceholder} className="flex-1 font-mono" aria-label={sizeLabel} />
              <Button variant="ghost" size="icon" onClick={() => onSizes(sizes.filter((_, idx) => idx !== i))} aria-label="Remove size">
                <Trash2 size={14} />
              </Button>
            </div>
          ))}
          <div>
            <Button variant="secondary" size="sm" onClick={() => onSizes([...sizes, { name: '', cpu: '', memory: '', type: '' }])}>
              <Plus size={14} /> Add size
            </Button>
          </div>
        </div>
      </section>
    </div>
  )
}
