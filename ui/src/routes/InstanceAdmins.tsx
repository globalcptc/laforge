import { useState } from 'react'
import { Plus, ShieldCheck, Trash2 } from 'lucide-react'
import { useAddInstanceAdmin, useInstanceAdmins, useMe, useRemoveInstanceAdmin } from '../api/hooks'
import { ApiError } from '../api/client'
import { EmptyState } from '../components/EmptyState'
import type { InstanceAdmin } from '../api/types'
import { Badge, Button, Card, CardHeader, Input, Label, Modal, PageHeader, Spinner, Table, TableScroller, Td, Th, useToast } from '../ui'
import { useTimeFormat } from '../lib/time'

// The people with instance-wide admin: full access to every repository, plus
// GitHub Connections, Infrastructure, and this list. LAFORGE_ADMIN_LOGINS on
// the server only seeds the list on first startup; it's managed here after.
// Instance-admin gated on the backend, same as Infrastructure.
export function InstanceAdmins() {
  const { data: admins, isLoading, error } = useInstanceAdmins()
  const { data: me } = useMe()
  const [addOpen, setAddOpen] = useState(false)

  if (error) {
    const forbidden = error instanceof ApiError && error.status === 403
    return (
      <div className="mx-auto max-w-2xl p-6">
        <Card className="p-4 text-sm text-fg">
          {forbidden
            ? "You're not an instance admin, so you can't see or change who is. Ask an existing admin to add you."
            : error instanceof ApiError
              ? error.message
              : 'Failed to load admins.'}
        </Card>
      </div>
    )
  }

  return (
    <>
      <PageHeader title="Admins" description="People with full access to every repository and to LaForge's own settings." />
      <div className="page-body">
        <Card>
          <CardHeader
            title="Instance Admins"
            description="GitHub accounts that can approve repositories, manage infrastructure, and add or remove other admins."
            actions={
              <Button variant="brand" size="sm" onClick={() => setAddOpen(true)}>
                <Plus size={14} /> Add Admin
              </Button>
            }
          />
          <div className="p-4 pt-2">
            {isLoading && (
              <div className="flex items-center gap-2 text-sm text-fg-muted">
                <Spinner /> Loading…
              </div>
            )}
            {!isLoading && (!admins || admins.length === 0) && <EmptyState icon={ShieldCheck} title="No admins" hint="Add a GitHub account to make it an admin." />}
            {admins && admins.length > 0 && (
              <TableScroller contain={false}>
                <Table>
                  <thead>
                    <tr>
                      <Th>GitHub account</Th>
                      <Th>Added by</Th>
                      <Th>Added</Th>
                      <Th></Th>
                    </tr>
                  </thead>
                  <tbody>
                    {admins.map((a) => (
                      <AdminRow key={a.github_login} admin={a} isMe={a.github_login.toLowerCase() === me?.github_login.toLowerCase()} isLast={admins.length === 1} />
                    ))}
                  </tbody>
                </Table>
              </TableScroller>
            )}
          </div>
        </Card>
      </div>
      <AddAdminModal open={addOpen} onClose={() => setAddOpen(false)} />
    </>
  )
}

function AdminRow({ admin, isMe, isLast }: { admin: InstanceAdmin; isMe: boolean; isLast: boolean }) {
  const remove = useRemoveInstanceAdmin()
  const toast = useToast()
  const fmt = useTimeFormat()

  async function onRemove() {
    const warning = isMe
      ? 'Remove yourself as an admin? You will lose access to this page and will need another admin to add you back.'
      : `Remove @${admin.github_login} as an admin?`
    if (!confirm(warning)) return
    try {
      await remove.mutateAsync(admin.github_login)
      toast({ title: 'Admin removed', description: `@${admin.github_login}`, tone: 'success' })
    } catch (e) {
      toast({ title: 'Remove failed', description: e instanceof ApiError ? e.message : 'error', tone: 'danger' })
    }
  }

  return (
    <tr>
      <Td className="text-fg">
        <span className="inline-flex items-center gap-2">
          {admin.avatar_url ? <img src={admin.avatar_url} alt="" className="size-5 rounded-full" /> : <span className="size-5 rounded-full bg-surface-sunken" />}
          <span className="font-mono">@{admin.github_login}</span>
          {isMe && <Badge tone="info">You</Badge>}
        </span>
      </Td>
      <Td className="text-fg-muted">{admin.added_by ? `@${admin.added_by}` : 'Initial setup'}</Td>
      <Td className="text-fg-muted">{fmt.date(admin.created_at)}</Td>
      <Td className="text-right whitespace-nowrap">
        <Button
          variant="ghost"
          size="sm"
          onClick={onRemove}
          disabled={isLast || remove.isPending}
          title={isLast ? 'Add another admin before removing the only one.' : undefined}
          className="text-danger hover:bg-danger-soft hover:text-danger"
        >
          <Trash2 size={12} /> Remove
        </Button>
      </Td>
    </tr>
  )
}

function AddAdminModal({ open, onClose }: { open: boolean; onClose: () => void }) {
  const add = useAddInstanceAdmin()
  const toast = useToast()
  const [login, setLogin] = useState('')
  const [error, setError] = useState<string | null>(null)

  const trimmed = login.trim().replace(/^@/, '')

  function close() {
    setLogin('')
    setError(null)
    onClose()
  }

  async function onAdd() {
    setError(null)
    try {
      const added = await add.mutateAsync(trimmed)
      toast({ title: 'Admin added', description: `@${added.github_login}`, tone: 'success' })
      close()
    } catch (e) {
      setError(e instanceof ApiError ? e.message : 'Add failed')
    }
  }

  return (
    <Modal
      open={open}
      onClose={close}
      title="Add Admin"
      description="They get full access to every repository and to these admin pages the next time they load LaForge. They don't need to have signed in before."
      footer={
        <>
          <Button variant="ghost" onClick={close}>
            Cancel
          </Button>
          <Button variant="brand" onClick={onAdd} disabled={trimmed === '' || add.isPending}>
            {add.isPending ? <Spinner /> : <Plus size={14} />} Add Admin
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-3">
        <div>
          <Label>GitHub username</Label>
          <Input
            className="mt-1"
            placeholder="octocat"
            value={login}
            onChange={(e) => setLogin(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter' && trimmed !== '' && !add.isPending) onAdd()
            }}
            autoFocus
          />
        </div>
        {error && <div className="rounded-token border border-danger/30 bg-danger-soft p-2 text-xs text-danger">{error}</div>}
      </div>
    </Modal>
  )
}
