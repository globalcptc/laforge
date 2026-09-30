import { useEffect, useRef, useState } from 'react'
import { Link } from '@tanstack/react-router'
import { Bell } from 'lucide-react'
import { useHome } from '../api/hooks'
import { AnchoredPopover, cn } from '../ui'

// App-wide, high-level alerts (direct product feedback: alerts should be
// "app-wide high-level alerts, not per-build"). Driven by the same
// cross-build `attention` aggregate Home already computes server-side
// (internal/api/home.go: a build that failed, has objects that failed to
// deploy, agents missing, or steps failed), polled every 15s -- so a
// problem on any build a person can see surfaces here from anywhere in the
// app, not only while looking at that one build. This is the single
// messages surface: the full per-build event record lives in that build's
// Logs tab, and its important items on the build Overview -- there's no
// separate per-build message bell, which was a confusing second inbox.
export function GlobalAlerts() {
  const { data: home } = useHome()
  const [open, setOpen] = useState(false)
  const anchorRef = useRef<HTMLButtonElement>(null)
  const panelRef = useRef<HTMLDivElement>(null)

  const alerts = home?.attention ?? []
  const count = alerts.length

  useEffect(() => {
    if (!open) return
    const onPointerDown = (e: PointerEvent) => {
      const target = e.target as Node
      if (anchorRef.current?.contains(target) || panelRef.current?.contains(target)) return
      setOpen(false)
    }
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setOpen(false)
    }
    window.addEventListener('pointerdown', onPointerDown)
    window.addEventListener('keydown', onKey)
    return () => {
      window.removeEventListener('pointerdown', onPointerDown)
      window.removeEventListener('keydown', onKey)
    }
  }, [open])

  return (
    <div className="relative">
      <button
        ref={anchorRef}
        type="button"
        onClick={() => setOpen((v) => !v)}
        aria-haspopup="menu"
        aria-expanded={open}
        aria-label={count > 0 ? `Alerts, ${count} active` : 'Alerts'}
        title="Alerts"
        className={cn(
          'relative flex size-8 items-center justify-center rounded-token-sm hover:bg-surface-hover',
          count > 0 ? 'text-danger' : 'text-fg-muted hover:text-fg',
        )}
      >
        <Bell size={16} />
        {count > 0 && (
          <span className="absolute -right-0.5 -top-0.5 flex min-w-[16px] items-center justify-center rounded-full bg-danger px-1 text-[10px] font-semibold leading-none text-fg-inverted">
            {count > 9 ? '9+' : count}
          </span>
        )}
      </button>

      {open && (
        <AnchoredPopover anchorRef={anchorRef} panelRef={panelRef} minWidth={340} maxHeight={400}>
          <div className="border-b border-border px-3 py-2 text-xs font-semibold uppercase tracking-wide text-fg-muted">Alerts</div>
          {count === 0 ? (
            <div className="px-3 py-4 text-sm text-fg-muted">Nothing needs attention right now.</div>
          ) : (
            <div className="divide-y divide-border">
              {alerts.map((a) => (
                <Link
                  key={`${a.build_id}:${a.reason}`}
                  to="/repos/$repoId/builds/$buildId"
                  params={{ repoId: a.repository_id, buildId: a.build_id }}
                  onClick={() => setOpen(false)}
                  className="block px-3 py-2 hover:bg-surface-hover"
                >
                  <div className="text-sm font-medium text-danger">{a.reason}</div>
                  <div className="mt-0.5 truncate text-xs text-fg-muted">
                    {a.environment_name} · {a.repository}
                  </div>
                </Link>
              ))}
            </div>
          )}
        </AnchoredPopover>
      )}
    </div>
  )
}
