import { useState } from 'react'
import { KeyRound, Trash2, Plus, FlaskConical, Boxes, Check, X, ChevronRight, ChevronDown } from 'lucide-react'
import {
  useRegistryCredentials,
  useUpsertRegistryCredential,
  useDeleteRegistryCredential,
  useTestRegistryCredential,
  useRegistryImages,
} from '../api/hooks'
import { ApiError } from '../api/client'
import { EmptyState } from './EmptyState'
import { Badge, Button, Card, CardHeader, Input, Label, Modal, Spinner, Table, TableScroller, Td, Th, useToast } from '../ui'
import { useTimeFormat } from '../lib/time'

// Private Docker registry credentials: a LaForge `container:` whose image ref
// names a private registry (e.g. registry.internal/team/app) gets a
// `docker login` before the pull. Docker Hub public images need no entry;
// authenticated Docker Hub uses the host "docker.io".
export function RegistryCredentials() {
  const { data: creds, isLoading } = useRegistryCredentials()
  const fmt = useTimeFormat()
  const [addOpen, setAddOpen] = useState(false)

  return (
    <Card className="mt-6">
      <CardHeader
        title="Docker Registry Credentials"
        description="Docker login for containers pulled from a private Docker registry. Public Docker Hub images need none."
        actions={
          <Button variant="brand" size="sm" onClick={() => setAddOpen(true)}>
            <Plus size={14} /> Add Credential
          </Button>
        }
      />
      <div className="p-4 pt-2">
        {isLoading && (
          <div className="flex items-center gap-2 text-sm text-fg-muted">
            <Spinner /> Loading…
          </div>
        )}
        {!isLoading && (!creds || creds.length === 0) && (
          <EmptyState icon={KeyRound} title="No registry credentials" hint="Add one for each private registry your containers pull from." />
        )}
        {creds && creds.length > 0 && (
          <TableScroller contain={false}>
            <Table>
              <thead>
                <tr>
                  <Th>Registry host</Th>
                  <Th>Username</Th>
                  <Th>Added</Th>
                  <Th></Th>
                </tr>
              </thead>
              <tbody>
                {creds.map((c) => (
                  <CredentialRow key={c.id} host={c.registry_host} username={c.username} added={fmt.date(c.created_at)} />
                ))}
              </tbody>
            </Table>
          </TableScroller>
        )}
      </div>
      <AddCredentialModal open={addOpen} onClose={() => setAddOpen(false)} />
    </Card>
  )
}

function CredentialRow({ host, username, added }: { host: string; username: string; added: string }) {
  const del = useDeleteRegistryCredential()
  const test = useTestRegistryCredential()
  const toast = useToast()
  const [result, setResult] = useState<{ ok: boolean; message: string } | null>(null)
  const [open, setOpen] = useState(false)

  async function onTest() {
    setResult(null)
    try {
      setResult(await test.mutateAsync(host))
    } catch (e) {
      setResult({ ok: false, message: e instanceof ApiError ? e.message : 'Test failed' })
    }
  }

  async function onDelete() {
    if (!confirm(`Delete credentials for "${host}"? Containers pulling private images from it will fail to log in.`)) return
    try {
      await del.mutateAsync(host)
      toast({ title: 'Credential deleted', description: host, tone: 'success' })
    } catch (e) {
      toast({ title: 'Delete failed', description: e instanceof ApiError ? e.message : 'error', tone: 'danger' })
    }
  }

  return (
    <>
      <tr>
        <Td className="font-mono text-fg">{host}</Td>
        <Td className="text-fg-muted">{username}</Td>
        <Td className="text-fg-muted">{added}</Td>
        <Td className="text-right whitespace-nowrap">
          <div className="inline-flex items-center gap-2">
            {result && (
              <Badge tone={result.ok ? 'success' : 'danger'} title={result.message}>
                {result.ok ? <Check size={11} /> : <X size={11} />}
                {result.ok ? 'Authenticated' : 'Failed'}
              </Badge>
            )}
            <Button variant="ghost" size="sm" onClick={onTest} disabled={test.isPending}>
              {test.isPending ? <Spinner /> : <FlaskConical size={12} />} Test
            </Button>
            <Button variant="ghost" size="sm" onClick={() => setOpen((v) => !v)}>
              {open ? <ChevronDown size={12} /> : <ChevronRight size={12} />}
              <Boxes size={12} /> Images
            </Button>
            <Button variant="ghost" size="sm" onClick={onDelete} disabled={del.isPending} className="text-danger hover:bg-danger-soft hover:text-danger">
              <Trash2 size={12} /> Delete
            </Button>
          </div>
        </Td>
      </tr>
      {open && (
        <tr>
          <Td colSpan={4} className="bg-surface-sunken">
            <RegistryImages host={host} />
          </Td>
        </tr>
      )}
    </>
  )
}

// RegistryImages lists what a registry actually holds -- repositories and their
// tags -- so an author can confirm the image they pushed is there and named the
// way their `container:` references it.
function RegistryImages({ host }: { host: string }) {
  const { data, isLoading, error } = useRegistryImages(host)
  if (isLoading)
    return (
      <div className="flex items-center gap-2 p-2 text-sm text-fg-muted">
        <Spinner /> Loading images…
      </div>
    )
  if (error) return <div className="p-2 text-sm text-danger">{error instanceof ApiError ? error.message : 'Failed to load images'}</div>
  if (!data) return null
  if (!data.supported) return <div className="p-2 text-sm text-fg-muted">{data.message}</div>
  if (data.images.length === 0) return <div className="p-2 text-sm text-fg-muted">No images found in this registry yet.</div>
  return (
    <div className="flex flex-col gap-2 p-2">
      {data.truncated && <div className="text-xs text-fg-subtle">Showing the first {data.images.length} repositories.</div>}
      <ul className="flex flex-col gap-1.5">
        {data.images.map((img) => (
          <li key={img.repository} className="flex flex-wrap items-baseline gap-x-2 gap-y-1">
            <span className="font-mono text-sm text-fg">{img.repository}</span>
            {img.error ? (
              <span className="text-xs text-danger">{img.error}</span>
            ) : img.tags.length === 0 ? (
              <span className="text-xs text-fg-subtle">no tags</span>
            ) : (
              img.tags.map((t) => (
                <Badge key={t} tone="info">
                  {t}
                </Badge>
              ))
            )}
          </li>
        ))}
      </ul>
    </div>
  )
}

function AddCredentialModal({ open, onClose }: { open: boolean; onClose: () => void }) {
  const upsert = useUpsertRegistryCredential()
  const toast = useToast()
  const [host, setHost] = useState('')
  const [username, setUsername] = useState('')
  const [secret, setSecret] = useState('')
  const [error, setError] = useState<string | null>(null)

  const valid = host.trim() !== '' && username.trim() !== '' && secret !== ''

  function reset() {
    setHost('')
    setUsername('')
    setSecret('')
    setError(null)
  }

  async function onAdd() {
    setError(null)
    try {
      await upsert.mutateAsync({ registry_host: host.trim(), username: username.trim(), secret })
      toast({ title: 'Credential saved', description: host.trim(), tone: 'success' })
      reset()
      onClose()
    } catch (e) {
      setError(e instanceof ApiError ? e.message : 'Save failed')
    }
  }

  return (
    <Modal
      open={open}
      onClose={() => {
        reset()
        onClose()
      }}
      title="Add Docker Registry Credential"
      description="A docker login for a private Docker registry. Adding a host that already has a credential replaces it. Authenticated Docker Hub uses the host “docker.io”."
      footer={
        <>
          <Button
            variant="ghost"
            onClick={() => {
              reset()
              onClose()
            }}
          >
            Cancel
          </Button>
          <Button variant="brand" onClick={onAdd} disabled={!valid || upsert.isPending}>
            {upsert.isPending ? <Spinner /> : <Plus size={14} />} Save Credential
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-3">
        <div>
          <Label>Registry host</Label>
          <Input className="mt-1" placeholder="registry.internal (or docker.io)" value={host} onChange={(e) => setHost(e.target.value)} autoFocus />
        </div>
        <div>
          <Label>Username</Label>
          <Input className="mt-1" placeholder="username" value={username} onChange={(e) => setUsername(e.target.value)} />
        </div>
        <div>
          <Label>Password / token</Label>
          <Input className="mt-1" placeholder="password / token" type="password" value={secret} onChange={(e) => setSecret(e.target.value)} />
        </div>
        {error && <div className="rounded-token border border-danger/30 bg-danger-soft p-2 text-xs text-danger">{error}</div>}
      </div>
    </Modal>
  )
}
