import { useEffect, useState } from 'react'
import { Clock } from 'lucide-react'
import type { AccessWindow } from '../api/types'

function formatDuration(ms: number): string {
  const totalSeconds = Math.max(0, Math.floor(ms / 1000))
  const h = Math.floor(totalSeconds / 3600)
  const m = Math.floor((totalSeconds % 3600) / 60)
  const s = totalSeconds % 60
  return [h, m, s].map((n) => String(n).padStart(2, '0')).join(':')
}

// The next scheduled access-window transition, as one line:
//   "Next window opens at <datetime> in <hh:mm:ss>"  (before a window)
//   "Next window closes at <datetime> in <hh:mm:ss>" (inside a window)
// Computed from the environment's authored `access:` windows (now real via
// GET /builds/{id}). This describes the upcoming WINDOW boundary, not the live
// open/closed state -- that's the badge's job -- so it never contradicts it (a
// team can be manually open for setup while the window still opens later).
//
// A live open-direction override that runs past the next close is a manual
// EXTENSION: the close moves out to the override time and is flagged. (A
// pre-window open override is NOT an extension -- it only holds access open
// until the window takes over, so it doesn't change the window boundary shown.)
export function AccessCountdown({
  windows,
  overrideUntil,
  overrideState,
  formatAt,
}: {
  windows: AccessWindow[] | undefined
  overrideUntil?: string | null
  overrideState?: string
  formatAt?: (d: Date) => string
}) {
  const [now, setNow] = useState(() => new Date())
  useEffect(() => {
    const t = setInterval(() => setNow(new Date()), 1000)
    return () => clearInterval(t)
  }, [])

  if (!windows || windows.length === 0) return null

  const at = formatAt ?? ((d: Date) => d.toLocaleString())
  const override = overrideUntil ? new Date(overrideUntil) : null

  // The soonest window boundary (open or close) strictly after now.
  const events: { time: Date; verb: 'opens' | 'closes' }[] = []
  for (const w of windows) {
    events.push({ time: new Date(w.open), verb: 'opens' })
    events.push({ time: new Date(w.close), verb: 'closes' })
  }
  const future = events.filter((e) => e.time > now).sort((a, b) => a.time.getTime() - b.time.getTime())
  if (future.length === 0) {
    return (
      <span className="flex items-center gap-1.5 text-xs text-fg-muted">
        <Clock size={12} />
        No more scheduled access windows
      </span>
    )
  }

  let { time, verb } = future[0]
  // An open override that reaches past the next close extends it.
  let extended = false
  if (verb === 'closes' && overrideState === 'open' && override && override > time) {
    time = override
    extended = true
  }

  return (
    <span className="flex items-center gap-1.5 text-xs text-fg-muted">
      <Clock size={12} className={verb === 'closes' ? 'text-success' : 'text-fg-subtle'} />
      Next window {verb} at {at(time)} in {formatDuration(time.getTime() - now.getTime())}
      {extended && <span className="text-warning"> (extended)</span>}
    </span>
  )
}
