import { useEffect, useMemo, useState, type ReactNode } from 'react'
import { useNavigate, useParams, useSearch } from '@tanstack/react-router'
import { ArrowLeft, ArrowRight, Check } from 'lucide-react'
import { builderConnectionPath, useBuilderConfigs, useCreateBuilderConfig, useUpdateBuilderConfig } from '../api/hooks'
import { ApiError, api } from '../api/client'
import type { BuilderConnection } from '../api/types'
import { Button, PageHeader, Spinner, cn, useToast } from '../ui'
import {
  DEFAULT_SIZES,
  availableImages,
  draftFromConfig,
  emptyHost,
  imagesFromConfig,
  isCloud,
  isHostPlaced,
  sizesFromConfig,
  toRequest,
  type Draft,
  type Kind,
} from '../components/builder-wizard/model'
import { NAME_PATTERN, StepType } from '../components/builder-wizard/StepType'
import { StepConnect } from '../components/builder-wizard/StepConnect'
import { StepPlacement } from '../components/builder-wizard/StepPlacement'
import { StepImages } from '../components/builder-wizard/StepImages'
import { StepSizes } from '../components/builder-wizard/StepSizes'
import { StepCloud, StepCloudConnect, StepCloudRegion, StepCloudImages, StepCloudSizes } from '../components/builder-wizard/StepCloud'
import { StepReview } from '../components/builder-wizard/StepReview'
import { ColorPicker } from '../components/ColorPicker'

type StepId = 'type' | 'connect' | 'placement' | 'images' | 'sizes' | 'cloud' | 'cloud-connect' | 'cloud-region' | 'cloud-images' | 'cloud-sizes' | 'review'

// editorSections groups a kind's forms into the editable sections the info page
// links to (Config / Images / Sizes), so EDITING a builder is a focused editor
// with Save -- not a walk through the creation wizard's steps. 'type' (name/kind,
// fixed after create) and 'review' are deliberately excluded.
function editorSections(kind: Kind): { id: string; label: string; steps: StepId[] }[] {
  if (kind === 'aws') {
    return [
      { id: 'config', label: 'Config', steps: ['cloud-connect', 'cloud-region'] },
      { id: 'images', label: 'Images', steps: ['cloud-images'] },
      { id: 'sizes', label: 'Sizes', steps: ['cloud-sizes'] },
    ]
  }
  if (isCloud(kind)) {
    return [{ id: 'config', label: 'Config', steps: ['cloud'] }] // StepCloud holds images + sizes inline
  }
  if (kind === 'fake') {
    return [{ id: 'config', label: 'Config', steps: [] }]
  }
  return [
    { id: 'config', label: 'Config', steps: ['connect', 'placement'] },
    { id: 'images', label: 'Images', steps: ['images'] },
    { id: 'sizes', label: 'Sizes', steps: ['sizes'] },
  ]
}

const STEP_TITLE: Record<StepId, string> = {
  type: 'Type',
  connect: 'Connect',
  placement: 'Placement',
  images: 'Images',
  sizes: 'Sizes',
  cloud: 'Mappings',
  'cloud-connect': 'Connect',
  'cloud-region': 'Region',
  'cloud-images': 'Images',
  'cloud-sizes': 'Sizes',
  review: 'Review',
}

// EditorSection gives one group of the builder editor a clear heading and a
// divider so the page reads as distinct sections (Appearance, Connection, …)
// instead of one long flat form. It stays frameless on purpose -- the step
// forms carry their own inner cards, so an outer card here would nest cards.
function EditorSection({ title, description, children }: { title: string; description?: string; children: ReactNode }) {
  return (
    <section>
      <div className="mb-4 border-b border-border pb-2">
        <h3 className="text-sm font-semibold tracking-tight text-fg">{title}</h3>
        {description && <p className="mt-0.5 text-xs text-fg-muted">{description}</p>}
      </div>
      {children}
    </section>
  )
}

// cloudBlocker validates a cloud builder's image/size maps: at least one of
// each, every row complete, no duplicate names. Blank rows are ignored (and
// dropped on save).
function cloudBlocker(d: Draft): string | null {
  if (d.kind === 'aws' && !d.awsCredentialId) return 'Connect an AWS access key first.'
  const imgs = d.images.filter((i) => i.name || i.fingerprint)
  if (imgs.length === 0) return 'Add at least one image mapping.'
  if (imgs.some((i) => !i.name || !i.fingerprint)) return 'Every image needs an os name and an id.'
  if (new Set(imgs.map((i) => i.name)).size !== imgs.length) return 'Two images share a name.'
  const szs = d.sizes.filter((s) => s.name || s.type)
  if (szs.length === 0) return 'Add at least one size mapping.'
  if (szs.some((s) => !s.name || !s.type)) return 'Every size needs a name and a type.'
  if (new Set(szs.map((s) => s.name)).size !== szs.length) return 'Two sizes share a name.'
  return null
}

// The AWS wizard splits the cloud image/size validation across its own steps
// (the picks are the same shape: image.fingerprint = AMI id, size.type = instance type).
function awsImageBlocker(d: Draft): string | null {
  const imgs = d.images.filter((i) => i.name || i.fingerprint)
  if (imgs.length === 0) return 'Add at least one os → AMI mapping.'
  if (imgs.some((i) => !i.name || !i.fingerprint)) return 'Every image needs an os name and an AMI.'
  if (new Set(imgs.map((i) => i.name)).size !== imgs.length) return 'Two images share a name.'
  return null
}

function awsSizeBlocker(d: Draft): string | null {
  const szs = d.sizes.filter((s) => s.name || s.type)
  if (szs.length === 0) return 'Add at least one size → instance-type mapping.'
  if (szs.some((s) => !s.name || !s.type)) return 'Every size needs a name and an instance type.'
  if (new Set(szs.map((s) => s.name)).size !== szs.length) return 'Two sizes share a name.'
  return null
}

// Adding or editing a builder, one decision at a time: pick the type, paste
// one trust token per server, then choose storage, networks, images, and
// sizes from what the servers actually have. Nothing is saved until the
// review step.
export function BuilderWizard() {
  const params = useParams({ strict: false }) as { name?: string }
  const editingName = params.name
  const search = useSearch({ strict: false }) as { section?: string }
  const navigate = useNavigate()
  const toast = useToast()

  const { data: configs, isLoading: configsLoading } = useBuilderConfigs()
  const create = useCreateBuilderConfig()
  const update = useUpdateBuilderConfig()

  const [draft, setDraft] = useState<Draft>({ name: '', kind: 'microcloud', hosts: [emptyHost()], images: [], sizes: [], externalAccessIp: '' })
  const [initialized, setInitialized] = useState(false)
  // Edit mode skips the Connect step, so host connections are auto-read once after
  // the draft loads (see the effect below). This guards that one-shot.
  const [hostsAutoRead, setHostsAutoRead] = useState(false)
  // Wizard-flow (create) state only; edit mode is the editor below, not the steps.
  const [stepIndex, setStepIndex] = useState(0)
  const [furthest, setFurthest] = useState(0)
  const [saveError, setSaveError] = useState<string | null>(null)
  const [refreshing, setRefreshing] = useState(false)
  // Which section the editor (edit mode) is showing; seeded from the ?section
  // deep-link the info page's Config/Images/Sizes buttons use.
  const [activeSection, setActiveSection] = useState<string>(search.section ?? 'config')

  const existing = editingName ? configs?.find((c) => c.name === editingName) : undefined

  // Build the starting draft once, when editing, the saved builder is in hand.
  useEffect(() => {
    if (initialized) return
    if (editingName) {
      if (!existing) return
      setDraft({ ...draftFromConfig(existing), images: imagesFromConfig(existing.incus_images), sizes: sizesFromConfig(existing.incus_sizes) })
    } else {
      setDraft((d) => ({ ...d, sizes: DEFAULT_SIZES.map((s) => ({ ...s })) }))
    }
    setInitialized(true)
  }, [initialized, editingName, existing])

  const available = useMemo(() => availableImages(draft.hosts), [draft.hosts])
  const availableKey = available.map((a) => a.image.fingerprint).join(',')

  // Older configs name images by alias only; once the hosts are read, pin
  // each to the fingerprint that alias points at.
  useEffect(() => {
    if (!availableKey) return
    setDraft((d) => ({
      ...d,
      images: d.images.map((img) => {
        if (img.fingerprint || img.server || img.source === 'snapshot') return img
        const found = available.find((a) => a.image.aliases.includes(img.alias))
        return found ? { ...img, fingerprint: found.image.fingerprint, vm: found.image.type === 'virtual-machine' } : img
      }),
    }))
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [availableKey])

  // Re-read every connected host, e.g. after copying an image onto one.
  async function refreshHosts() {
    setRefreshing(true)
    try {
      const hosts = await Promise.all(
        draft.hosts.map(async (h) =>
          h.credentialId ? { ...h, connection: await api.get<BuilderConnection>(builderConnectionPath(h.credentialId, draft.kind, h.project)) } : h,
        ),
      )
      setDraft((d) => ({ ...d, hosts }))
    } catch (e) {
      toast({ title: 'Refresh failed', description: e instanceof ApiError ? e.message : 'Could not reach a host', tone: 'danger' })
    } finally {
      setRefreshing(false)
    }
  }

  // In the editor there's no Connect step to populate host connections, so read
  // them once after the draft loads -- otherwise the Images/Placement sections
  // show "no host connected" until Refresh is clicked. Only hosts with an
  // enrolled credential can be re-read; legacy (cert-path) hosts have no probe
  // endpoint and are left as-is.
  useEffect(() => {
    if (!editingName || !initialized || hostsAutoRead) return
    setHostsAutoRead(true)
    if (draft.hosts.some((h) => h.credentialId && !h.connection)) {
      void refreshHosts()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [editingName, initialized, hostsAutoRead])

  const steps: StepId[] =
    draft.kind === 'fake'
      ? ['type', 'review']
      : draft.kind === 'aws'
        ? ['type', 'cloud-connect', 'cloud-region', 'cloud-images', 'cloud-sizes', 'review']
        : isCloud(draft.kind)
          ? ['type', 'cloud', 'review']
          : ['type', 'connect', 'placement', 'images', 'sizes', 'review']
  const step = steps[Math.min(stepIndex, steps.length - 1)]

  function setKind(kind: Kind) {
    setDraft((d) => ({
      ...d,
      kind,
      hosts: (kind === 'microcloud' ? [d.hosts[0] ?? emptyHost()] : d.hosts.length ? d.hosts : [emptyHost()]).map((h) => ({ ...h, networkType: kind === 'incus' ? h.networkType ?? 'bridge' : undefined })),
      oneTeamPerHost: kind === 'incus' ? d.oneTeamPerHost ?? true : false,
      // Seed a cloud builder with the standard size names (empty types to fill
      // in) and one blank image row, so it starts as a form to complete.
      ...(isCloud(kind) && d.images.length === 0 && d.sizes.every((s) => !s.type)
        ? { images: [{ name: '', fingerprint: '', alias: '', vm: false, server: '' }], sizes: DEFAULT_SIZES.map((s) => ({ name: s.name, cpu: '', memory: '', type: '' })) }
        : {}),
    }))
    setFurthest(0)
  }

  const hostReady = (h: Draft['hosts'][number]) => !!h.connection || !!h.legacy || !!h.credentialId
  const blocker: Record<StepId, string | null> = {
    type: !draft.name ? 'Name the builder first.' : !NAME_PATTERN.test(draft.name) ? 'Fix the name first.' : null,
    connect: draft.hosts.length === 0 || !draft.hosts.every(hostReady) ? 'Connect every server first.' : null,
    placement: !draft.hosts.every(isHostPlaced) ? 'Choose storage and supported networking for every server.' : null,
    images: draft.images.some((i) => !i.name)
      ? 'Every offered image needs a name.'
      : draft.images.some((i) => i.source === 'snapshot' && (!i.instance || !i.snapshot))
        ? 'Every snapshot needs an instance and a snapshot name.'
      : new Set(draft.images.map((i) => i.name)).size !== draft.images.length
        ? 'Two images share a name.'
        : null,
    sizes: draft.sizes.some((s) => !s.cpu || !s.memory) ? 'Every size needs CPUs and memory.' : null,
    cloud: cloudBlocker(draft),
    'cloud-connect': !draft.awsCredentialId ? 'Connect an AWS access key first.' : null,
    'cloud-region': !draft.awsConfig?.region ? 'Choose a region first.' : null,
    'cloud-images': awsImageBlocker(draft),
    'cloud-sizes': awsSizeBlocker(draft),
    review: null,
  }

  function goTo(i: number) {
    setStepIndex(i)
    setFurthest((f) => Math.max(f, i))
  }

  async function save() {
    setSaveError(null)
    try {
      const req = toRequest(draft)
      if (editingName) await update.mutateAsync({ name: editingName, ...req })
      else await create.mutateAsync({ name: draft.name, ...req })
      toast({ title: editingName ? 'Builder saved' : 'Builder created', description: draft.name, tone: 'success' })
      if (editingName) {
        navigate({ to: '/admin/infrastructure/$name', params: { name: editingName } })
      } else {
        navigate({ to: '/admin/infrastructure' })
      }
    } catch (e) {
      setSaveError(e instanceof ApiError ? e.message : 'Saving failed')
    }
  }

  const loading = !initialized && (configsLoading || (!!editingName && !existing))
  if (editingName && !configsLoading && configs && !existing) {
    return (
      <div className="mx-auto max-w-2xl p-6">
        <div className="card p-4 text-sm text-fg">No builder named {editingName}.</div>
      </div>
    )
  }

  const saving = create.isPending || update.isPending
  const isLast = stepIndex === steps.length - 1

  // renderStep draws one step's form; shared by the creation wizard (one at a
  // time) and the editor (a section's steps stacked).
  const renderStep = (id: StepId) => {
    switch (id) {
      case 'type':
        return (
          <StepType
            name={draft.name}
            kind={draft.kind}
            color={draft.color ?? ''}
            editing={!!editingName}
            onName={(name) => setDraft((d) => ({ ...d, name }))}
            onKind={setKind}
            onColor={(color) => setDraft((d) => ({ ...d, color }))}
          />
        )
      case 'connect':
        return <StepConnect kind={draft.kind} hosts={draft.hosts} onChange={(hosts) => setDraft((d) => ({ ...d, hosts }))} />
      case 'placement':
        return (
          <StepPlacement
            kind={draft.kind}
            oneTeamPerHost={draft.oneTeamPerHost ?? false}
            onPlacementPolicyChange={(oneTeamPerHost) => setDraft((d) => ({ ...d, oneTeamPerHost }))}
            hosts={draft.hosts}
            onChange={(hosts) => setDraft((d) => ({ ...d, hosts }))}
            microcloudPublicAccess={draft.microcloudPublicAccess}
            onPublicAccessChange={(microcloudPublicAccess) => setDraft((d) => ({ ...d, microcloudPublicAccess }))}
            externalAccessIp={draft.externalAccessIp}
            externalPortMin={draft.externalPortMin}
            externalPortMax={draft.externalPortMax}
            onExternalChange={(patch) => setDraft((d) => ({ ...d, ...patch }))}
          />
        )
      case 'images':
        return (
          <StepImages
            kind={draft.kind}
            images={draft.images}
            available={available}
            hosts={draft.hosts}
            refreshing={refreshing}
            onRefresh={refreshHosts}
            onChange={(images) => setDraft((d) => ({ ...d, images }))}
          />
        )
      case 'sizes':
        return <StepSizes sizes={draft.sizes} onChange={(sizes) => setDraft((d) => ({ ...d, sizes }))} />
      case 'cloud':
        return (
          <StepCloud
            images={draft.images}
            sizes={draft.sizes}
            onImages={(images) => setDraft((d) => ({ ...d, images }))}
            onSizes={(sizes) => setDraft((d) => ({ ...d, sizes }))}
          />
        )
      case 'cloud-connect':
        return (
          <StepCloudConnect
            credentialId={draft.awsCredentialId}
            account={draft.awsConfig?.account}
            onConnected={(awsCredentialId, account) => setDraft((d) => ({ ...d, awsCredentialId, awsConfig: { ...d.awsConfig, account } }))}
          />
        )
      case 'cloud-region':
        return (
          <StepCloudRegion
            credentialId={draft.awsCredentialId}
            awsConfig={draft.awsConfig}
            discovery={draft.awsDiscovery}
            onAwsConfig={(awsConfig) => setDraft((d) => ({ ...d, awsConfig }))}
            onDiscovery={(awsDiscovery) => setDraft((d) => ({ ...d, awsDiscovery }))}
            onRegionChange={() =>
              setDraft((d) => ({
                ...d,
                images: d.images.map((i) => ({ ...i, fingerprint: '' })),
                sizes: d.sizes.map((s) => ({ ...s, type: '' })),
              }))
            }
          />
        )
      case 'cloud-images':
        return (
          <StepCloudImages
            credentialId={draft.awsCredentialId}
            region={draft.awsConfig?.region}
            images={draft.images}
            discovery={draft.awsDiscovery}
            onImages={(images) => setDraft((d) => ({ ...d, images }))}
            onDiscovery={(awsDiscovery) => setDraft((d) => ({ ...d, awsDiscovery }))}
          />
        )
      case 'cloud-sizes':
        return <StepCloudSizes sizes={draft.sizes} discovery={draft.awsDiscovery} onSizes={(sizes) => setDraft((d) => ({ ...d, sizes }))} />
      case 'review':
        return <StepReview draft={draft} />
      default:
        return null
    }
  }

  // Edit mode is a focused EDITOR, not the creation wizard: section tabs
  // (Config / Images / Sizes) and a Save button, instead of a step-by-step flow
  // that only lets you save on the final Review step. See the info page's edit
  // buttons.
  if (editingName) {
    const sections = editorSections(draft.kind)
    const active = sections.find((s) => s.id === activeSection) ?? sections[0]
    const activeBlocker = active?.steps.map((id) => blocker[id]).find(Boolean) ?? null
    return (
      <>
        <PageHeader
          title={`Edit ${editingName}`}
          breadcrumbs={[{ label: 'Infrastructure', href: '/admin/infrastructure' }, { label: editingName }]}
        />
        <div className="page-body">
          <div className="mx-auto max-w-4xl">
            {loading ? (
              <div className="flex items-center gap-2 text-sm text-fg-muted">
                <Spinner /> Loading…
              </div>
            ) : (
              <>
                {sections.length > 1 && (
                  <div className="flex gap-1 border-b border-border">
                    {sections.map((s) => (
                      <button
                        key={s.id}
                        onClick={() => setActiveSection(s.id)}
                        className={cn(
                          'border-b-2 px-3 py-2 text-sm font-medium',
                          active?.id === s.id ? 'border-accent text-fg' : 'border-transparent text-fg-muted hover:text-fg',
                        )}
                      >
                        {s.label}
                      </button>
                    ))}
                  </div>
                )}
                <div className="mt-6 space-y-8">
                  {active?.id === 'config' && (
                    <EditorSection title="Appearance" description="Shown as a colored pill wherever this builder’s name appears.">
                      <ColorPicker
                        value={draft.color ?? ''}
                        onChange={(color) => setDraft((d) => ({ ...d, color }))}
                        previewName={draft.name}
                        previewKind={draft.kind}
                      />
                    </EditorSection>
                  )}
                  {active && active.steps.length > 0
                    ? active.steps.map((id) => (
                        <EditorSection key={id} title={STEP_TITLE[id]}>
                          {renderStep(id)}
                        </EditorSection>
                      ))
                    : active?.id !== 'config' && <div className="text-sm text-fg-muted">This builder has nothing to configure.</div>}
                </div>
                <div className="mt-6 flex items-center justify-between border-t border-border pt-4">
                  <Button variant="ghost" onClick={() => navigate({ to: '/admin/infrastructure/$name', params: { name: editingName } })}>
                    <ArrowLeft size={12} /> Cancel
                  </Button>
                  <div className="flex items-center gap-3">
                    {saveError && <span className="max-w-md text-xs text-danger">{saveError}</span>}
                    {activeBlocker && <span className="text-xs text-fg-muted">{activeBlocker}</span>}
                    <Button variant="primary" onClick={save} disabled={saving || !!activeBlocker}>
                      {saving ? <Spinner /> : <Check size={12} />} Save
                    </Button>
                  </div>
                </div>
              </>
            )}
          </div>
        </div>
      </>
    )
  }

  return (
    <>
      <PageHeader
        title={editingName ? `Edit ${editingName}` : 'New builder'}
        breadcrumbs={[{ label: 'Infrastructure', href: '/admin/infrastructure' }, { label: editingName ?? 'New' }]}
      />
      <div className="page-body">
        <div className="mx-auto max-w-4xl">
          <Stepper steps={steps} current={stepIndex} furthest={furthest} onSelect={goTo} />

          <div className="mt-5">
            {loading ? (
              <div className="flex items-center gap-2 text-sm text-fg-muted">
                <Spinner /> Loading…
              </div>
            ) : (
              renderStep(step)
            )}
          </div>

          <div className="mt-6 flex items-center justify-between border-t border-border pt-4">
            <Button variant="ghost" onClick={() => (stepIndex === 0 ? navigate({ to: '/admin/infrastructure' }) : goTo(stepIndex - 1))}>
              <ArrowLeft size={12} /> {stepIndex === 0 ? 'Cancel' : 'Back'}
            </Button>
            <div className="flex items-center gap-3">
              {saveError && <span className="max-w-md text-xs text-danger">{saveError}</span>}
              {!isLast && blocker[step] && <span className="text-xs text-fg-muted">{blocker[step]}</span>}
              {isLast ? (
                <Button variant="primary" onClick={save} disabled={saving}>
                  {saving ? <Spinner /> : <Check size={12} />} {editingName ? 'Save Builder' : 'Create Builder'}
                </Button>
              ) : (
                <Button onClick={() => goTo(stepIndex + 1)} disabled={!!blocker[step] || loading}>
                  Next <ArrowRight size={12} />
                </Button>
              )}
            </div>
          </div>
        </div>
      </div>
    </>
  )
}

function Stepper({ steps, current, furthest, onSelect }: { steps: StepId[]; current: number; furthest: number; onSelect: (i: number) => void }) {
  return (
    <ol className="flex flex-wrap items-center gap-x-2 gap-y-2">
      {steps.map((s, i) => {
        const done = i < current
        const active = i === current
        const reachable = i <= furthest
        return (
          <li key={s} className="flex items-center gap-2">
            <button
              type="button"
              disabled={!reachable}
              onClick={() => onSelect(i)}
              className={cn(
                'flex items-center gap-2 rounded-token px-2 py-1 text-sm disabled:cursor-default',
                active ? 'text-fg' : reachable ? 'text-fg-muted hover:text-fg' : 'text-fg-subtle',
              )}
            >
              <span
                className={cn(
                  'flex size-6 items-center justify-center rounded-full border text-xs font-semibold',
                  active && 'border-accent bg-accent text-fg-inverted',
                  done && 'border-accent bg-accent-soft text-accent-fg',
                  !active && !done && 'border-border',
                )}
              >
                {done ? <Check size={12} /> : i + 1}
              </span>
              <span className={cn(active && 'font-medium')}>{STEP_TITLE[s]}</span>
            </button>
            {i < steps.length - 1 && <span className="h-px w-6 bg-border" />}
          </li>
        )
      })}
    </ol>
  )
}
