import { useEffect, useState } from 'react'
import { Check, Info, Plus, Trash2 } from 'lucide-react'
import { Button, Input, Spinner } from '../../ui'
import { api, ApiError } from '../../api/client'
import { useConnectBuilder, builderConnectionPath } from '../../api/hooks'
import { SearchSelect, type SearchOption } from '../SearchSelect'
import type { AwsConfig, BuilderConnection, Discovery } from '../../api/types'
import type { ImageDraft, SizeDraft } from './model'

// StepCloud configures an OpenStack builder (the typed-id form): credentials and
// region come from the runner environment, and the two maps are entered by hand
// -- OpenStack has no stored credential to discover from yet. AWS uses the
// discovery-driven StepCloud* steps below instead.
export function StepCloud({
  images,
  sizes,
  onImages,
  onSizes,
}: {
  images: ImageDraft[]
  sizes: SizeDraft[]
  onImages: (images: ImageDraft[]) => void
  onSizes: (sizes: SizeDraft[]) => void
}) {
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
        <span>
          Credentials and region are read from the runner’s OpenStack environment (OS_AUTH_URL / OS_USERNAME /
          OS_PASSWORD / OS_PROJECT_NAME / OS_REGION_NAME). Nothing secret is stored here.
        </span>
      </div>
      <section>
        <div className="mb-1 text-sm font-medium text-fg">Images</div>
        <div className="mb-2 text-xs text-fg-muted">
          Map each LaForge <code className="font-mono">os</code> name to the Glance image id to launch it from.
        </div>
        <div className="flex flex-col gap-2">
          {images.map((img, i) => (
            <div key={i} className="flex items-center gap-2">
              <Input value={img.name} onChange={(e) => setImage(i, { name: e.target.value })} placeholder="ubuntu22" className="flex-1" />
              <span className="text-fg-subtle">→</span>
              <Input value={img.fingerprint} onChange={(e) => setImage(i, { fingerprint: e.target.value })} placeholder="Glance image id" className="flex-1 font-mono" aria-label="Image id" />
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
          Map each LaForge <code className="font-mono">size</code> name to a flavor.
        </div>
        <div className="flex flex-col gap-2">
          {sizes.map((s, i) => (
            <div key={i} className="flex items-center gap-2">
              <Input value={s.name} onChange={(e) => setSize(i, { name: e.target.value })} placeholder="small" className="flex-1" />
              <span className="text-fg-subtle">→</span>
              <Input value={s.type ?? ''} onChange={(e) => setSize(i, { type: e.target.value })} placeholder="m1.medium" className="flex-1 font-mono" aria-label="Flavor" />
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

// ---- AWS discovery-driven steps -----------------------------------------

function infoBox(children: React.ReactNode) {
  return (
    <div className="flex items-start gap-2 rounded-token border border-border bg-surface-sunken px-3 py-2 text-xs text-fg-muted">
      <Info size={14} className="mt-0.5 shrink-0 text-fg-subtle" />
      <span>{children}</span>
    </div>
  )
}

// Field is a labelled input with a one-line explanation under it, so every AWS
// setting says what it is and where to get it inline (not only in the doc).
function Field({ label, hint, children }: { label: string; hint: string; children: React.ReactNode }) {
  return (
    <label className="flex flex-col gap-1 text-xs text-fg-muted">
      {label}
      {children}
      <span className="text-fg-subtle">{hint}</span>
    </label>
  )
}

// StepCloudConnect enrolls the AWS access key (validated + stored server-side;
// the wizard keeps only the credential id). It must happen first -- every later
// step discovers against this credential.
export function StepCloudConnect({ credentialId, account, onConnected }: { credentialId?: string; account?: string; onConnected: (id: string | undefined, account?: string) => void }) {
  const connect = useConnectBuilder()
  const [keyId, setKeyId] = useState('')
  const [secret, setSecret] = useState('')
  const [error, setError] = useState<string | null>(null)

  async function onConnect() {
    setError(null)
    try {
      const conn = await connect.mutateAsync({ kind: 'aws', aws_access_key_id: keyId.trim(), aws_secret_access_key: secret.trim() })
      // credential.server_name carries the AWS account id (captured by STS at connect).
      onConnected(conn.credential.id, conn.credential.server_name || undefined)
      setKeyId('')
      setSecret('')
    } catch (e) {
      setError(e instanceof ApiError ? e.message : 'Connecting failed')
    }
  }

  return (
    <div className="flex flex-col gap-4">
      {infoBox(
        <>
          Enter an IAM access key LaForge authenticates as. It is validated and stored server-side; only its id is kept
          in this form afterward. The exact IAM permissions (and a ready-to-deploy CloudFormation template) are in{' '}
          <code className="font-mono">docs/builders/aws.md</code>.
        </>,
      )}
      {credentialId ? (
        <div className="flex items-center justify-between gap-2 rounded-token border border-success/40 bg-success-soft px-3 py-2 text-xs">
          <span className="flex items-center gap-2 text-fg">
            <Check size={14} className="text-success" /> Access key connected and stored{account ? ` (account ${account})` : ''}.
          </span>
          <Button variant="ghost" size="sm" onClick={() => onConnected(undefined, undefined)}>
            Change
          </Button>
        </div>
      ) : (
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
      )}
    </div>
  )
}

// fetchDiscovery is the shared imperative rediscovery call (mirrors
// StepPlacement's ProjectPicker.choose): one request, result stored by the caller.
async function fetchDiscovery(credentialId: string, region?: string, imageQuery?: string): Promise<Discovery> {
  const conn = await api.get<BuilderConnection>(builderConnectionPath(credentialId, 'aws', undefined, { region, imageQuery }))
  return conn.discovery
}

// StepCloudRegion picks the region (AMIs and instance types are region-scoped, so
// it comes before them) and collects the AWS account infrastructure.
export function StepCloudRegion({
  credentialId,
  awsConfig,
  discovery,
  onAwsConfig,
  onDiscovery,
  onRegionChange,
}: {
  credentialId?: string
  awsConfig?: AwsConfig
  discovery?: Discovery
  onAwsConfig: (cfg: AwsConfig) => void
  onDiscovery: (d: Discovery) => void
  onRegionChange: () => void
}) {
  const aws = awsConfig ?? {}
  const setAws = (patch: Partial<AwsConfig>) => onAwsConfig({ ...aws, ...patch })
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // Load discovery once on entry: scoped to the already-chosen region (editing an
  // existing builder) so its instance types / AMIs are ready, else region-less to
  // populate the region list. Skips if we already have regions in hand.
  useEffect(() => {
    if (!credentialId || (discovery?.regions && discovery.regions.length > 0)) return
    let live = true
    setLoading(true)
    fetchDiscovery(credentialId, aws.region)
      .then((d) => live && onDiscovery(d))
      .catch((e) => live && setError(e instanceof ApiError ? e.message : 'Could not list regions'))
      .finally(() => live && setLoading(false))
    return () => {
      live = false
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [credentialId])

  async function chooseRegion(region: string) {
    setAws({ region })
    onRegionChange() // region-scoped picks (AMIs/types) no longer apply
    if (!credentialId) return
    setLoading(true)
    setError(null)
    try {
      onDiscovery(await fetchDiscovery(credentialId, region))
    } catch (e) {
      setError(e instanceof ApiError ? e.message : 'Could not read that region')
    } finally {
      setLoading(false)
    }
  }

  const regionOptions: SearchOption[] = (discovery?.regions ?? []).map((r) => ({ value: r, label: r }))

  return (
    <div className="flex flex-col gap-4">
      {infoBox('Pick the region to deploy in. Instance types and AMIs are discovered for the region you choose, so set it before the next steps.')}
      <section>
        <div className="mb-1 text-sm font-medium text-fg">Region</div>
        <div className="flex items-center gap-2">
          <div className="max-w-sm flex-1">
            <SearchSelect value={aws.region ?? ''} options={regionOptions} onChange={chooseRegion} placeholder="Choose a region" searchPlaceholder="Search regions…" />
          </div>
          {loading && <Spinner />}
        </div>
        <div className="mt-1 text-xs text-fg-subtle">Where this builder deploys. The AMIs and instance types in the next steps are discovered for the region you pick.</div>
        {error && <div className="mt-2 text-xs text-danger">{error}</div>}
      </section>

      <section>
        <div className="mb-1 text-sm font-medium text-fg">AWS Account</div>
        <div className="mb-2 text-xs text-fg-muted">
          The execution role is required to run containers; the rest have sane defaults. Full field-by-field guidance
          (and the CloudFormation template that outputs the role ARN) is in <code className="font-mono">docs/builders/aws.md</code>.
        </div>
        <div className="flex flex-col gap-3">
          <Field label="ECS Cluster" hint="The ECS cluster Fargate containers run in. Every account has one named “default” — leave it unless you made a dedicated cluster. Only used for container: objects.">
            <Input value={aws.ecs_cluster ?? ''} onChange={(e) => setAws({ ecs_cluster: e.target.value })} placeholder="default" className="font-mono" />
          </Field>
          <Field label="Task Execution Role ARN" hint="IAM role Fargate assumes to pull images and ship logs. Required for containers. It's the ExecutionRoleArn output of the aws-iam.yaml CloudFormation stack — paste it verbatim.">
            <Input value={aws.execution_role_arn ?? ''} onChange={(e) => setAws({ execution_role_arn: e.target.value })} placeholder="arn:aws:iam::123456789012:role/laforge-runner-ecs-exec" className="font-mono" />
          </Field>
          <Field label="CloudWatch Log Group" hint="Where Fargate task logs go when an environment sets no container_logs of its own. Optional — leave blank for none.">
            <Input value={aws.log_group ?? ''} onChange={(e) => setAws({ log_group: e.target.value })} placeholder="/laforge/containers (optional)" className="font-mono" />
          </Field>
          <Field label="Agent Init Image" hint="Small public image a Fargate container runs once to download the agent. Default alpine:3.20 — change only if you can't pull from Docker Hub.">
            <Input value={aws.agent_init_image ?? ''} onChange={(e) => setAws({ agent_init_image: e.target.value })} placeholder="alpine:3.20" className="font-mono" />
          </Field>
        </div>
      </section>
    </div>
  )
}

// StepCloudImages maps each `os` name to an AMI: owned AMIs are listed, and
// typing searches the public catalogs for the region.
export function StepCloudImages({
  credentialId,
  region,
  images,
  discovery,
  onImages,
  onDiscovery,
}: {
  credentialId?: string
  region?: string
  images: ImageDraft[]
  discovery?: Discovery
  onImages: (images: ImageDraft[]) => void
  onDiscovery: (d: Discovery) => void
}) {
  const [searching, setSearching] = useState(false)
  const amiOptions: SearchOption[] = (discovery?.images ?? []).map((img) => ({
    value: img.fingerprint,
    label: img.aliases?.[0] || img.fingerprint,
    hint: img.fingerprint,
  }))

  async function search(q: string) {
    if (!credentialId) return
    setSearching(true)
    try {
      onDiscovery(await fetchDiscovery(credentialId, region, q))
    } catch {
      // Leave the current options; the picker just shows what it has.
    } finally {
      setSearching(false)
    }
  }

  function setImage(i: number, patch: Partial<ImageDraft>) {
    onImages(images.map((img, idx) => (idx === i ? { ...img, ...patch } : img)))
  }

  return (
    <div className="flex flex-col gap-4">
      {infoBox('Map each os name content uses to an AMI. Your own AMIs are listed; type a name (e.g. “ubuntu”, “Windows”) to search Amazon, Canonical and Microsoft images for this region.')}
      <div className="flex flex-col gap-2">
        {images.map((img, i) => (
          <div key={i} className="flex items-center gap-2">
            <Input value={img.name} onChange={(e) => setImage(i, { name: e.target.value })} placeholder="ubuntu22" className="w-40" aria-label="os name" />
            <span className="text-fg-subtle">→</span>
            <div className="min-w-0 flex-1">
              <SearchSelect
                value={img.fingerprint}
                options={amiOptions}
                onChange={(v) => setImage(i, { fingerprint: v })}
                onSearch={search}
                loading={searching}
                placeholder="Choose or search an AMI"
                searchPlaceholder="Search AMIs by name…"
                emptyLabel="No AMIs — type to search"
              />
            </div>
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
    </div>
  )
}

// StepCloudSizes maps each `size` name to an EC2 instance type from the region's
// discovered list.
export function StepCloudSizes({ sizes, discovery, onSizes }: { sizes: SizeDraft[]; discovery?: Discovery; onSizes: (sizes: SizeDraft[]) => void }) {
  const typeOptions: SearchOption[] = (discovery?.instance_types ?? []).map((t) => ({ value: t, label: t }))
  function setSize(i: number, patch: Partial<SizeDraft>) {
    onSizes(sizes.map((s, idx) => (idx === i ? { ...s, ...patch } : s)))
  }
  return (
    <div className="flex flex-col gap-4">
      {infoBox('Map each size name content uses to an EC2 instance type. The list is the instance types available in the region you chose.')}
      <div className="flex flex-col gap-2">
        {sizes.map((s, i) => (
          <div key={i} className="flex items-center gap-2">
            <Input value={s.name} onChange={(e) => setSize(i, { name: e.target.value })} placeholder="small" className="w-40" aria-label="size name" />
            <span className="text-fg-subtle">→</span>
            <div className="min-w-0 flex-1">
              <SearchSelect value={s.type ?? ''} options={typeOptions} onChange={(v) => setSize(i, { type: v })} placeholder="Choose an instance type" searchPlaceholder="Search instance types…" />
            </div>
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
    </div>
  )
}
