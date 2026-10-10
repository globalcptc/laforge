import { useState } from 'react'
import { Check, Info, Plus, Trash2 } from 'lucide-react'
import { Button, Input } from '../../ui'
import { ApiError } from '../../api/client'
import { useConnectBuilder } from '../../api/hooks'
import type { AwsConfig } from '../../api/types'
import type { ImageDraft, Kind, SizeDraft } from './model'

// StepCloud configures a cloud builder (AWS/OpenStack): the credential, the
// account infrastructure (AWS), and two maps -- a LaForge `os` name to a cloud
// image id (AMI / Glance) and a `size` name to an instance type / flavor. For
// AWS the credential is entered here and enrolled server-side (stored like an
// Incus client credential; the wizard keeps only the resulting id, never the
// secret). OpenStack still reads its credential from the runner environment.
export function StepCloud({
  kind,
  images,
  sizes,
  awsConfig,
  awsCredentialId,
  onImages,
  onSizes,
  onAwsConfig,
  onAwsCredential,
}: {
  kind: Kind
  images: ImageDraft[]
  sizes: SizeDraft[]
  awsConfig?: AwsConfig
  awsCredentialId?: string
  onImages: (images: ImageDraft[]) => void
  onSizes: (sizes: SizeDraft[]) => void
  onAwsConfig: (cfg: AwsConfig) => void
  onAwsCredential: (id: string | undefined) => void
}) {
  const isAws = kind === 'aws'
  const aws = awsConfig ?? {}
  const setAws = (patch: Partial<AwsConfig>) => onAwsConfig({ ...aws, ...patch })
  const imageIdLabel = isAws ? 'AMI id' : 'Image id'
  const imageIdPlaceholder = isAws ? 'ami-0abc123…' : 'Glance image id'
  const sizeLabel = isAws ? 'Instance type' : 'Flavor'
  const sizePlaceholder = isAws ? 't3.medium' : 'm1.medium'

  function setImage(i: number, patch: Partial<ImageDraft>) {
    onImages(images.map((img, idx) => (idx === i ? { ...img, ...patch } : img)))
  }
  function setSize(i: number, patch: Partial<SizeDraft>) {
    onSizes(sizes.map((s, idx) => (idx === i ? { ...s, ...patch } : s)))
  }

  return (
    <div className="flex flex-col gap-5">
      {!isAws && (
        <div className="flex items-start gap-2 rounded-token border border-border bg-surface-sunken px-3 py-2 text-xs text-fg-muted">
          <Info size={14} className="mt-0.5 shrink-0 text-fg-subtle" />
          <span>
            Credentials and region are read from the runner’s OpenStack environment (OS_AUTH_URL / OS_USERNAME /
            OS_PASSWORD / OS_PROJECT_NAME / OS_REGION_NAME). Nothing secret is stored here.
          </span>
        </div>
      )}

      {isAws && <AwsCredentials credentialId={awsCredentialId} onCredential={onAwsCredential} />}

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

      {isAws && (
        <section>
          <div className="mb-1 text-sm font-medium text-fg">AWS Account</div>
          <div className="mb-2 text-xs text-fg-muted">
            Account infrastructure. The execution role is required to run containers; the rest have sane defaults.
          </div>
          <div className="flex flex-col gap-3">
            <label className="flex flex-col gap-1 text-xs text-fg-muted">
              Region
              <Input value={aws.region ?? ''} onChange={(e) => setAws({ region: e.target.value })} placeholder="us-east-1" className="font-mono" />
            </label>
            <label className="flex flex-col gap-1 text-xs text-fg-muted">
              ECS Cluster
              <Input value={aws.ecs_cluster ?? ''} onChange={(e) => setAws({ ecs_cluster: e.target.value })} placeholder="default" className="font-mono" />
            </label>
            <label className="flex flex-col gap-1 text-xs text-fg-muted">
              Task Execution Role ARN
              <Input value={aws.execution_role_arn ?? ''} onChange={(e) => setAws({ execution_role_arn: e.target.value })} placeholder="arn:aws:iam::123456789012:role/ecsTaskExecutionRole" className="font-mono" />
            </label>
            <label className="flex flex-col gap-1 text-xs text-fg-muted">
              CloudWatch Log Group
              <Input value={aws.log_group ?? ''} onChange={(e) => setAws({ log_group: e.target.value })} placeholder="/laforge/containers (optional)" className="font-mono" />
            </label>
            <label className="flex flex-col gap-1 text-xs text-fg-muted">
              Agent Init Image
              <Input value={aws.agent_init_image ?? ''} onChange={(e) => setAws({ agent_init_image: e.target.value })} placeholder="alpine:3.20" className="font-mono" />
            </label>
            <label className="flex items-center gap-2 text-xs text-fg">
              <input
                type="checkbox"
                checked={aws.assign_public_ip !== false}
                onChange={(e) => setAws({ assign_public_ip: e.target.checked })}
                className="h-4 w-4 rounded-token-sm border-border accent-accent"
              />
              Assign a public IP to hosts (needed for egress to the LaForge control plane without a NAT)
            </label>
          </div>
        </section>
      )}
    </div>
  )
}

// AwsCredentials enrolls an AWS access key: the secret is sent once to the
// server, validated (STS), and stored; the component holds only the resulting
// credential id. On edit of an existing builder the id arrives already set and
// the form shows "connected" without re-entry.
function AwsCredentials({ credentialId, onCredential }: { credentialId?: string; onCredential: (id: string | undefined) => void }) {
  const connect = useConnectBuilder()
  const [keyId, setKeyId] = useState('')
  const [secret, setSecret] = useState('')
  const [error, setError] = useState<string | null>(null)

  async function onConnect() {
    setError(null)
    try {
      const conn = await connect.mutateAsync({ kind: 'aws', aws_access_key_id: keyId.trim(), aws_secret_access_key: secret.trim() })
      onCredential(conn.credential.id)
      setKeyId('')
      setSecret('')
    } catch (e) {
      setError(e instanceof ApiError ? e.message : 'Connecting failed')
    }
  }

  if (credentialId) {
    return (
      <section>
        <div className="mb-1 text-sm font-medium text-fg">AWS Credentials</div>
        <div className="flex items-center justify-between gap-2 rounded-token border border-success/40 bg-success-soft px-3 py-2 text-xs">
          <span className="flex items-center gap-2 text-fg">
            <Check size={14} className="text-success" /> Access key connected and stored.
          </span>
          <Button variant="ghost" size="sm" onClick={() => onCredential(undefined)}>
            Change
          </Button>
        </div>
      </section>
    )
  }

  return (
    <section>
      <div className="mb-1 text-sm font-medium text-fg">AWS Credentials</div>
      <div className="mb-2 text-xs text-fg-muted">
        An IAM access key LaForge authenticates as. It is validated and stored server-side; only its id is kept in this
        form afterward. See the required IAM permissions in <code className="font-mono">docs/builders/aws.md</code>.
      </div>
      <div className="flex flex-col gap-3">
        <label className="flex flex-col gap-1 text-xs text-fg-muted">
          Access Key ID
          <Input value={keyId} onChange={(e) => setKeyId(e.target.value)} placeholder="AKIA…" className="font-mono" />
        </label>
        <label className="flex flex-col gap-1 text-xs text-fg-muted">
          Secret Access Key
          <Input type="password" value={secret} onChange={(e) => setSecret(e.target.value)} placeholder="••••••••••••••••" className="font-mono" />
        </label>
        {error && <div className="rounded-token border border-danger/40 bg-danger-soft px-3 py-2 text-xs text-fg">{error}</div>}
        <div>
          <Button variant="secondary" size="sm" onClick={onConnect} disabled={!keyId.trim() || !secret.trim() || connect.isPending}>
            {connect.isPending ? 'Connecting…' : 'Connect'}
          </Button>
        </div>
      </div>
    </section>
  )
}
