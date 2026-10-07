import { useState } from 'react'
import { Link } from '@tanstack/react-router'
import { Pencil, Plus, ServerCog, Trash2, Boxes, CircleCheck } from 'lucide-react'
import { useBuilderConfigs, useDeleteBuilderConfig } from '../api/hooks'
import { ApiError } from '../api/client'
import { EmptyState } from '../components/EmptyState'
import { BuilderImageConsole } from '../components/BuilderImageConsole'
import { RegistryCredentials } from '../components/RegistryCredentials'
import { KIND_LABEL } from '../components/builder-wizard/model'
import type { BuilderConfig } from '../api/types'
import { Badge, Button, Card, CardHeader, PageHeader, Spinner, Table, TableScroller, Td, Th, buttonClass, cn, useToast } from '../ui'

// Which builder kinds have a docker base image to build. MicroCloud (LXD, no
// OCI runtime) boots every container from it; Incus runs a single-image
// container as native OCI and needs it only for Docker Compose containers.
const USES_DOCKER_BASE = (kind: string) => kind === 'microcloud' || kind === 'incus'

// The builders a configured build can deploy with. Adding or editing one is
// a step-by-step workflow (BuilderWizard.tsx) -- this page only lists them.
// Instance-admin gated on the backend, same as Installations.
export function BuilderConfigs() {
  const { data: configs, isLoading, error } = useBuilderConfigs()
  const [consoleFor, setConsoleFor] = useState<string | null>(null)

  if (error) {
    const forbidden = error instanceof ApiError && error.status === 403
    return (
      <div className="mx-auto max-w-2xl p-6">
        <div className="card p-4 text-sm text-fg">
          {forbidden
            ? "You're not an instance admin, so you can't see or change builders."
            : error instanceof ApiError
              ? error.message
              : 'Failed to load builders.'}
        </div>
      </div>
    )
  }

  return (
    <>
      <PageHeader title="Infrastructure" description="The infrastructure a configured build deploys to." />
      <div className="page-body">
        <Card>
          <CardHeader
            title="Builders"
            description="The clusters, hosts, and clouds a configured build can deploy to."
            actions={
              <Link to="/admin/infrastructure/new" className={buttonClass({ variant: 'brand', size: 'sm' })}>
                <Plus size={14} /> New Builder
              </Link>
            }
          />
          <div className="p-4 pt-2">
            {isLoading && (
              <div className="flex items-center gap-2 text-sm text-fg-muted">
                <Spinner /> Loading…
              </div>
            )}

            {!isLoading && (!configs || configs.length === 0) && (
              <EmptyState icon={ServerCog} title="No builders yet" hint="Add a MicroCloud cluster, Incus hosts, or a cloud to start deploying builds." />
            )}

            {configs && configs.length > 0 && (
              <TableScroller contain={false}>
                <Table>
                  <thead>
                    <tr>
                      <Th>Name</Th>
                      <Th>Type</Th>
                      <Th>Servers</Th>
                      <Th>Images</Th>
                      <Th>Base image</Th>
                      <Th></Th>
                    </tr>
                  </thead>
                  <tbody>
                    {configs.map((bc) => (
                      <BuilderRow key={bc.id} bc={bc} onOpenConsole={setConsoleFor} />
                    ))}
                  </tbody>
                </Table>
              </TableScroller>
            )}
          </div>
        </Card>

        <RegistryCredentials />
      </div>
      {consoleFor && <BuilderImageConsole name={consoleFor} onClose={() => setConsoleFor(null)} />}
    </>
  )
}

function serverSummary(bc: BuilderConfig): string {
  if (bc.kind === 'fake') return '—'
  if (bc.kind === 'aws' || bc.kind === 'openstack') return 'runner env creds'
  if (bc.kind === 'incus') return `${bc.incus_hosts.length} host${bc.incus_hosts.length === 1 ? '' : 's'}`
  return bc.incus_credential_id ? '1 cluster' : (bc.incus_api_url ?? '—')
}

function BuilderRow({ bc, onOpenConsole }: { bc: BuilderConfig; onOpenConsole: (name: string) => void }) {
  const del = useDeleteBuilderConfig()
  const toast = useToast()
  const [error, setError] = useState<string | null>(null)
  const imageCount = Object.keys(bc.incus_images ?? {}).length
  const usesBase = USES_DOCKER_BASE(bc.kind)

  async function onDelete() {
    if (!confirm(`Delete builder "${bc.name}"? Any configured build still naming it will fail to deploy.`)) return
    setError(null)
    try {
      await del.mutateAsync(bc.name)
      toast({ title: 'Builder deleted', description: bc.name, tone: 'success' })
    } catch (e) {
      setError(e instanceof ApiError ? e.message : 'Delete failed')
    }
  }

  return (
    <tr>
      <Td className="font-medium text-fg">
        <Link to="/admin/infrastructure/$name" params={{ name: bc.name }} className="text-accent hover:underline">
          {bc.name}
        </Link>
      </Td>
      <Td>
        <Badge tone="accent">{KIND_LABEL[bc.kind]}</Badge>
      </Td>
      <Td className="text-fg-muted">{serverSummary(bc)}</Td>
      <Td className="text-fg-muted">{bc.kind === 'fake' ? '—' : imageCount}</Td>
      <Td>
        {usesBase ? (
          <button onClick={() => onOpenConsole(bc.name)} className="inline-flex items-center gap-1.5 text-xs hover:underline">
            {bc.docker_base_fingerprint ? (
              <>
                <CircleCheck size={13} className="text-success" />
                <span className="font-mono text-fg-muted">{bc.docker_base_fingerprint.slice(0, 10)}</span>
              </>
            ) : (
              <>
                <Boxes size={13} className="text-fg-subtle" />
                <span className="text-fg-muted">Build…</span>
              </>
            )}
          </button>
        ) : (
          <span className="text-xs text-fg-subtle">N/A</span>
        )}
      </Td>
      <Td className="text-right">
        <Link
          to="/admin/infrastructure/$name"
          params={{ name: bc.name }}
          className={cn(buttonClass({ variant: 'ghost', size: 'sm' }), 'text-brand hover:bg-brand-soft hover:text-brand')}
        >
          <Pencil size={12} /> Edit
        </Link>
        <Button variant="ghost" size="sm" onClick={onDelete} disabled={del.isPending} className="text-danger hover:bg-danger-soft hover:text-danger">
          <Trash2 size={12} /> Delete
        </Button>
        {error && <div className="mt-1 text-xs text-danger">{error}</div>}
      </Td>
    </tr>
  )
}
