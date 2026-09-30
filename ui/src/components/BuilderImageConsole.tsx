import { useEffect, useRef, useState } from 'react'
import { RotateCw, CircleCheck, CircleX, Loader, Boxes } from 'lucide-react'
import { useBuilderImageBuilds, useImageBuildLog, useRebuildBuilderImage } from '../api/hooks'
import { Badge, Button, InspectorPanel, Spinner, cn, type BadgeProps } from '../ui'

// BuilderImageConsole is the per-builder "docker base image" transparency
// surface (Docker-container design): it shows the current build status, a
// Rebuild button, and a live console that tails exactly what the job is doing
// on the cluster (import base → install Docker → publish). Opened from the
// Infrastructure page; the build itself runs in the orchestrator
// (internal/orchestrator/image_build.go) and this just streams its log.
export function BuilderImageConsole({ name, onClose }: { name: string; onClose: () => void }) {
  const { data: builds } = useBuilderImageBuilds(name)
  const rebuild = useRebuildBuilderImage(name)
  const [activeId, setActiveId] = useState<string | undefined>(undefined)

  // Default to the most recent build (e.g. the one auto-started on create).
  useEffect(() => {
    if (!activeId && builds && builds.length > 0) setActiveId(builds[0].id)
  }, [builds, activeId])

  const { data: log } = useImageBuildLog(activeId)
  const status = log?.status ?? builds?.find((b) => b.id === activeId)?.status

  async function onRebuild() {
    const b = await rebuild.mutateAsync()
    setActiveId(b.id)
  }

  // Keep the console scrolled to the newest line as it streams.
  const preRef = useRef<HTMLPreElement>(null)
  useEffect(() => {
    if (preRef.current) preRef.current.scrollTop = preRef.current.scrollHeight
  }, [log?.lines?.length])

  return (
    <InspectorPanel open overlay onClose={onClose} title="Container Base Image" subtitle={name} width="min(94vw, 46rem)">
      <div className="mb-3 flex items-center gap-2">
        {status ? <BuildStatusBadge status={status} /> : <span className="text-xs text-fg-muted">No build yet</span>}
        {log?.image_fingerprint && (
          <span className="font-mono text-xs text-fg-muted">{log.image_fingerprint.slice(0, 12)}</span>
        )}
        <Button
          variant="coral"
          size="sm"
          className="ml-auto"
          onClick={onRebuild}
          disabled={rebuild.isPending || status === 'running' || status === 'pending'}
        >
          {rebuild.isPending ? <Spinner /> : <RotateCw size={14} />} Rebuild
        </Button>
      </div>

      <p className="mb-2 text-xs text-fg-muted">
        Builds the docker-ready base image on this cluster — imports an Ubuntu container base, installs Docker, and
        publishes it as <code className="rounded bg-surface-sunken px-1">laforge-docker-base</code>. LaForge containers
        boot from it.
      </p>

      {log?.error && (
        <div className="mb-2 rounded-token border border-danger/30 bg-danger-soft px-2 py-1.5 text-xs text-danger">
          {log.error}
        </div>
      )}

      <pre
        ref={preRef}
        className="h-[60vh] overflow-auto whitespace-pre-wrap rounded-token border border-border bg-[#0b0618] p-3 font-mono text-xs leading-relaxed text-[#c7f0d8]"
      >
        {log?.lines && log.lines.length > 0
          ? log.lines.map((l) => l.line).join('\n')
          : status === 'pending' || status === 'running'
            ? 'Waiting for the orchestrator to start the build…'
            : 'No output yet. Click Rebuild to start a build.'}
        {(status === 'running' || status === 'pending') && <span className="animate-pulse"> ▍</span>}
      </pre>
    </InspectorPanel>
  )
}

const STATUS: Record<string, { icon: typeof CircleCheck; tone: BadgeProps['tone']; label: string; spin?: boolean }> = {
  pending: { icon: Loader, tone: 'info', label: 'Queued', spin: true },
  running: { icon: Loader, tone: 'info', label: 'Building', spin: true },
  succeeded: { icon: CircleCheck, tone: 'success', label: 'Built' },
  failed: { icon: CircleX, tone: 'danger', label: 'Failed' },
}

function BuildStatusBadge({ status }: { status: string }) {
  const s = STATUS[status] ?? { icon: Boxes, tone: 'neutral' as const, label: status }
  const Icon = s.icon
  return (
    <Badge tone={s.tone} className="rounded-full py-0.5">
      <Icon size={12} className={cn(s.spin && 'animate-spin')} /> {s.label}
    </Badge>
  )
}
