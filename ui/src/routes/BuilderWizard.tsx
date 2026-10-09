import { useEffect, useMemo, useState } from 'react'
import { useNavigate, useParams } from '@tanstack/react-router'
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
import { StepCloud } from '../components/builder-wizard/StepCloud'
import { StepReview } from '../components/builder-wizard/StepReview'

type StepId = 'type' | 'connect' | 'placement' | 'images' | 'sizes' | 'cloud' | 'review'

const STEP_TITLE: Record<StepId, string> = {
  type: 'Type',
  connect: 'Connect',
  placement: 'Placement',
  images: 'Images',
  sizes: 'Sizes',
  cloud: 'Mappings',
  review: 'Review',
}

// cloudBlocker validates a cloud builder's image/size maps: at least one of
// each, every row complete, no duplicate names. Blank rows are ignored (and
// dropped on save).
function cloudBlocker(d: Draft): string | null {
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

// Adding or editing a builder, one decision at a time: pick the type, paste
// one trust token per server, then choose storage, networks, images, and
// sizes from what the servers actually have. Nothing is saved until the
// review step.
export function BuilderWizard() {
  const params = useParams({ strict: false }) as { name?: string }
  const editingName = params.name
  const navigate = useNavigate()
  const toast = useToast()

  const { data: configs, isLoading: configsLoading } = useBuilderConfigs()
  const create = useCreateBuilderConfig()
  const update = useUpdateBuilderConfig()

  const [draft, setDraft] = useState<Draft>({ name: '', kind: 'microcloud', hosts: [emptyHost()], images: [], sizes: [], externalAccessIp: '' })
  const [initialized, setInitialized] = useState(false)
  const [stepIndex, setStepIndex] = useState(editingName ? 1 : 0)
  const [furthest, setFurthest] = useState(editingName ? 5 : 0)
  const [saveError, setSaveError] = useState<string | null>(null)
  const [refreshing, setRefreshing] = useState(false)

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
        if (img.fingerprint || img.server) return img
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

  const steps: StepId[] = draft.kind === 'fake' ? ['type', 'review'] : isCloud(draft.kind) ? ['type', 'cloud', 'review'] : ['type', 'connect', 'placement', 'images', 'sizes', 'review']
  const step = steps[Math.min(stepIndex, steps.length - 1)]

  function setKind(kind: Kind) {
    setDraft((d) => ({
      ...d,
      kind,
      hosts: kind === 'microcloud' ? [d.hosts[0] ?? emptyHost()] : d.hosts.length ? d.hosts : [emptyHost()],
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
    placement: !draft.hosts.every(isHostPlaced) ? 'Choose a storage pool and uplink for every server.' : null,
    images: draft.images.some((i) => !i.name)
      ? 'Every offered image needs a name.'
      : new Set(draft.images.map((i) => i.name)).size !== draft.images.length
        ? 'Two images share a name.'
        : null,
    sizes: draft.sizes.some((s) => !s.cpu || !s.memory) ? 'Every size needs CPUs and memory.' : null,
    cloud: cloudBlocker(draft),
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
      navigate({ to: '/admin/infrastructure' })
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
              <>
                {step === 'type' && (
                  <StepType
                    name={draft.name}
                    kind={draft.kind}
                    editing={!!editingName}
                    onName={(name) => setDraft((d) => ({ ...d, name }))}
                    onKind={setKind}
                  />
                )}
                {step === 'connect' && <StepConnect kind={draft.kind} hosts={draft.hosts} onChange={(hosts) => setDraft((d) => ({ ...d, hosts }))} />}
                {step === 'placement' && (
                  <StepPlacement
                    kind={draft.kind}
                    hosts={draft.hosts}
                    onChange={(hosts) => setDraft((d) => ({ ...d, hosts }))}
                    externalAccessIp={draft.externalAccessIp}
                    externalPortMin={draft.externalPortMin}
                    externalPortMax={draft.externalPortMax}
                    onExternalChange={(patch) => setDraft((d) => ({ ...d, ...patch }))}
                  />
                )}
                {step === 'images' && (
                  <StepImages
                    kind={draft.kind}
                    images={draft.images}
                    available={available}
                    hosts={draft.hosts}
                    refreshing={refreshing}
                    onRefresh={refreshHosts}
                    onChange={(images) => setDraft((d) => ({ ...d, images }))}
                  />
                )}
                {step === 'sizes' && <StepSizes sizes={draft.sizes} onChange={(sizes) => setDraft((d) => ({ ...d, sizes }))} />}
                {step === 'cloud' && (
                  <StepCloud
                    kind={draft.kind}
                    images={draft.images}
                    sizes={draft.sizes}
                    onImages={(images) => setDraft((d) => ({ ...d, images }))}
                    onSizes={(sizes) => setDraft((d) => ({ ...d, sizes }))}
                  />
                )}
                {step === 'review' && <StepReview draft={draft} />}
              </>
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
