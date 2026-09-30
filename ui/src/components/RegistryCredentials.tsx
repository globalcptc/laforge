import { useState } from 'react'
import { KeyRound, Trash2, Plus } from 'lucide-react'
import { useRegistryCredentials, useUpsertRegistryCredential, useDeleteRegistryCredential } from '../api/hooks'
import { ApiError } from '../api/client'
import { EmptyState } from './EmptyState'
import { Button, Card, CardHeader, Input, Label, Modal, Spinner, Table, TableScroller, Td, Th, useToast } from '../ui'
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
  const toast = useToast()
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
    <tr>
      <Td className="font-mono text-fg">{host}</Td>
      <Td className="text-fg-muted">{username}</Td>
      <Td className="text-fg-muted">{added}</Td>
      <Td className="text-right">
        <Button variant="ghost" size="sm" onClick={onDelete} disabled={del.isPending} className="text-danger hover:bg-danger-soft hover:text-danger">
          <Trash2 size={12} /> Delete
        </Button>
      </Td>
    </tr>
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
