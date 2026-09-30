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

// "Time until the next transition, per-team exceptions visible."
// Computed purely from
// the environment's own authored schedule (the
// `access:` windows, now real and flowing through GET /builds/{id} --
// see internal/api/builds_ui.go); a per-team override_until is passed in
// separately by the caller since it can push an individual team's own
// close later than the schedule's default.
// `state` is the object's live access_state (open/closed), when the caller has
// it. The live state is authoritative for whether access is open right now (the
// badge shows it); this countdown only describes the *next scheduled window
// transition* as an event, so it never contradicts the badge -- e.g. access can
// be live-open for setup while the first competition window is still hours away.
export function AccessCountdown({ windows, overrideUntil, state }: { windows: AccessWindow[] | undefined; overrideUntil?: string | null; state?: string }) {
  const [now, setNow] = useState(() => new Date())
  useEffect(() => {
    const t = setInterval(() => setNow(new Date()), 1000)
    return () => clearInterval(t)
  }, [])

  if (!windows || windows.length === 0) return null

  const override = overrideUntil ? new Date(overrideUntil) : null
  const parsed = windows.map((w) => ({ open: new Date(w.open), close: new Date(w.close) })).sort((a, b) => a.open.getTime() - b.open.getTime())

  // In a scheduled window and not manually closed: count down to when access is
  // scheduled to close (or the extended override, if later).
  const active = parsed.find((w) => now >= w.open && now <= w.close)
  if (active && state !== 'closed') {
    const closesAt = override && override > active.close ? override : active.close
    if (now < closesAt) {
      return (
        <span className="flex items-center gap-1.5 text-xs text-fg-muted">
          <Clock size={12} className="text-success" />
          Access closes in {formatDuration(closesAt.getTime() - now.getTime())}
          {override && override > active.close && <span className="text-warning"> (extended)</span>}
        </span>
      )
    }
  }

  // Otherwise, the next scheduled window -- phrased as the upcoming event, not a
  // current open/closed claim (that's the badge's job).
  const next = parsed.find((w) => w.open > now)
  if (next) {
    return (
      <span className="flex items-center gap-1.5 text-xs text-fg-muted">
        <Clock size={12} className="text-fg-subtle" />
        Next window opens in {formatDuration(next.open.getTime() - now.getTime())}
      </span>
    )
  }

  return (
    <span className="flex items-center gap-1.5 text-xs text-fg-muted">
      <Clock size={12} />
      No more scheduled access windows
    </span>
  )
}
