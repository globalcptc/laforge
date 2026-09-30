import { useParams } from '@tanstack/react-router'
import { useState } from 'react'
import { HardDrive, Trash2, Check } from 'lucide-react'
import { useArtifacts, usePurgeArtifacts } from '../api/hooks'
import { EmptyState } from '../components/EmptyState'
import { Button, Spinner, cn } from '../ui'
import type { ArtifactClass } from '../api/types'

function formatBytes(n: number): string {
  if (n <= 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.min(units.length - 1, Math.floor(Math.log(n) / Math.log(1024)))
  const v = n / Math.pow(1024, i)
  return `${v >= 100 || i === 0 ? Math.round(v) : v.toFixed(1)} ${units[i]}`
}

// "Build → Artifacts: Storage by class -- agent binaries, rendered scripts,
// uploaded files, step output -- with sizes and a total. Actions: purge
// artifacts... Each states exactly what it removes and what survives."
// Backed by GET /builds/{id}/artifacts and POST
// /builds/{id}/artifacts/purge (internal/api/artifacts.go). "Delete build"
// is the existing Tear Down action in the header, not duplicated here.
export function BuildArtifacts() {
  const { buildId } = useParams({ from: '/repos/$repoId/builds/$buildId/artifacts' })
  const { data, isLoading } = useArtifacts(buildId)
  const purge = usePurgeArtifacts(buildId)
  const [confirming, setConfirming] = useState(false)

  if (isLoading)
    return (
      <div className="flex items-center gap-2 text-sm text-fg-muted">
        <Spinner /> Loading…
      </div>
    )

  if (!data) {
    return <EmptyState icon={HardDrive} title="No storage information" hint="Artifact storage for this build could not be loaded." />
  }

  const purgeable = data.classes.filter((c) => c.purgeable && c.count > 0)

  function doPurge() {
    purge.mutate(undefined, { onSettled: () => setConfirming(false) })
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <div className="text-xs font-medium uppercase tracking-wide text-fg-muted">Total stored</div>
          <div className="text-2xl font-semibold tracking-tight text-fg">{formatBytes(data.total_bytes)}</div>
        </div>
        {purgeable.length > 0 &&
          (confirming ? (
            <div className="flex items-center gap-2">
              <span className="text-xs text-fg-muted">Remove {purgeable.reduce((a, c) => a + c.count, 0)} agent binary artifact(s)? They regenerate on the next deploy.</span>
              <Button variant="danger" size="sm" onClick={doPurge} disabled={purge.isPending}>
                {purge.isPending ? <Spinner /> : <Trash2 size={14} />} Purge
              </Button>
              <Button variant="ghost" size="sm" onClick={() => setConfirming(false)} disabled={purge.isPending}>
                Cancel
              </Button>
            </div>
          ) : (
            <Button variant="ghost" size="sm" onClick={() => setConfirming(true)} className="text-fg-muted">
              <Trash2 size={14} /> Purge artifacts
            </Button>
          ))}
      </div>

      {purge.isSuccess && !confirming && (
        <div className="flex items-center gap-2 rounded-token border border-success/30 bg-success-soft px-3 py-2 text-xs text-success">
          <Check size={14} /> Purged. Agent binaries will be rebuilt on the next deploy.
        </div>
      )}

      <div className="overflow-hidden rounded-token-lg border border-border">
        {data.classes.map((c, i) => (
          <ArtifactRow key={c.class} c={c} last={i === data.classes.length - 1} />
        ))}
      </div>
    </div>
  )
}

function ArtifactRow({ c, last }: { c: ArtifactClass; last: boolean }) {
  return (
    <div className={cn('flex flex-wrap items-center gap-x-3 gap-y-1 bg-surface-raised px-3 py-2.5', !last && 'border-b border-border')}>
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-2">
          <span className="font-medium text-fg">{c.class}</span>
          {c.purgeable && <span className="rounded-token-sm bg-warning-soft px-1.5 py-0.5 text-[11px] font-medium text-warning">purgeable</span>}
          {c.on_demand && <span className="rounded-token-sm bg-surface-sunken px-1.5 py-0.5 text-[11px] text-fg-muted">on demand</span>}
        </div>
        <div className="mt-0.5 text-xs text-fg-muted">{c.description}</div>
      </div>
      <div className="text-right">
        <div className="font-mono text-sm text-fg">{c.on_demand ? '—' : formatBytes(c.bytes)}</div>
        <div className="text-xs text-fg-subtle">{c.count} item{c.count === 1 ? '' : 's'}</div>
      </div>
    </div>
  )
}
