import { useState } from 'react'
import { useNavigate, useParams } from '@tanstack/react-router'
import { PowerOff, ShieldAlert, Trash2, TriangleAlert } from 'lucide-react'
import { useBuild, useForceDeleteBuild, useForceTornDown, useMe } from '../api/hooks'
import { ApiError } from '../api/client'
import { EmptyState } from '../components/EmptyState'
import { Button, Input, useToast } from '../ui'

// Build -> Admin: the instance-admin escape hatches for a build whose real
// infrastructure is already gone, moved off the bottom of every build page into
// their own tab at the end. Both change LaForge's records only -- they never ask
// the hoster to remove anything -- so they're deliberately out of the way and, for
// Force Remove, gated behind typing the exact build id.
export function BuildAdmin() {
  const { repoId, buildId } = useParams({ from: '/repos/$repoId/builds/$buildId/admin' })
  const { data: build } = useBuild(buildId)
  const me = useMe()
  const forceDelete = useForceDeleteBuild(buildId, repoId)
  const forceTornDown = useForceTornDown(buildId)
  const navigate = useNavigate()
  const toast = useToast()
  const [forceConfirm, setForceConfirm] = useState('')
  const [forceError, setForceError] = useState<string | null>(null)
  const [forceDownError, setForceDownError] = useState<string | null>(null)

  async function onForceTornDown() {
    setForceDownError(null)
    try {
      await forceTornDown.mutateAsync()
      toast({ title: 'Build marked torn down', description: 'No hosts were destroyed; clean up any that remain by hand.', tone: 'info' })
    } catch (e) {
      const message = e instanceof ApiError ? e.message : 'Force torn down failed'
      setForceDownError(message)
      toast({ title: 'Force torn down failed', description: message, tone: 'danger', duration: 0 })
    }
  }

  async function onForceDelete() {
    setForceError(null)
    try {
      await forceDelete.mutateAsync()
      // The build no longer exists -- leave its page for the repo's build list.
      navigate({ to: '/repos/$repoId', params: { repoId } })
    } catch (e) {
      const message = e instanceof ApiError ? e.message : 'Force remove failed'
      setForceError(message)
      toast({ title: 'Force remove failed', description: message, tone: 'danger', duration: 0 })
    }
  }

  if (!me.data?.is_instance_admin) {
    return <EmptyState icon={ShieldAlert} title="Admin only" hint="These controls are available to instance admins." />
  }
  if (!build) return null

  return (
    <div className="max-w-xl">
      <h2 className="text-sm font-semibold text-fg">Admin</h2>
      <p className="mt-1 text-xs text-fg-muted">
        Escape hatches for a build whose real infrastructure is already gone. They change LaForge’s records only — they never ask the
        hoster to remove anything. Use <span className="font-medium">Tear Down</span> for a normal teardown.
      </p>

      {build.status === 'tearing_down' && (
        <div className="mt-4 rounded-token border border-warning/30 bg-warning-soft/40 p-3.5">
          <div className="flex items-center gap-2 text-sm font-semibold text-warning">
            <TriangleAlert size={14} /> Force Torn Down
          </div>
          <p className="mt-1.5 text-fg-muted">
            Marks this build torn down and every remaining host, container, and network as destroyed{' '}
            <span className="font-semibold text-fg">without</span> asking the hoster to remove anything. Use it for a teardown that
            cannot finish when the real infrastructure is already gone or you will remove it by hand. The build and its history stay.
          </p>
          <div className="mt-2 flex flex-wrap items-center gap-2">
            <Button variant="danger" onClick={onForceTornDown} disabled={forceTornDown.isPending}>
              <PowerOff size={12} /> {forceTornDown.isPending ? 'Marking…' : 'Force Torn Down'}
            </Button>
          </div>
          {forceDownError && <div className="mt-2 text-danger">{forceDownError}</div>}
        </div>
      )}

      <div className="mt-4 rounded-token border border-danger/30 bg-danger-soft/40 p-3.5">
        <div className="flex items-center gap-2 text-sm font-semibold text-danger">
          <TriangleAlert size={14} /> Force remove from database
        </div>
        <p className="mt-1.5 text-fg-muted">
          Deletes this build and all its records from the database. It does <span className="font-semibold text-fg">not</span> tear down
          any hosts, containers, or networks — use <span className="font-medium">Tear Down</span> for that. Only use this for a build whose
          real infrastructure is already gone (so teardown can never complete). This cannot be undone.
        </p>
        <p className="mt-2 text-fg-muted">
          Type the build id <span className="font-mono text-fg-subtle">{buildId}</span> to confirm:
        </p>
        <div className="mt-1.5 flex flex-wrap items-center gap-2">
          <Input
            value={forceConfirm}
            onChange={(e) => setForceConfirm(e.target.value)}
            placeholder="build id"
            className="w-80 max-w-full font-mono"
            aria-label="Type the build id to confirm removal"
          />
          <Button variant="danger" disabled={forceConfirm.trim() !== buildId || forceDelete.isPending} onClick={onForceDelete}>
            <Trash2 size={12} /> {forceDelete.isPending ? 'Removing…' : 'Force Remove'}
          </Button>
        </div>
        {forceError && <div className="mt-2 text-danger">{forceError}</div>}
      </div>
    </div>
  )
}
