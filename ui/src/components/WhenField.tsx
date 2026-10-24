import { useEffect, useState } from 'react'
import { CalendarClock, CircleCheck, TriangleAlert } from 'lucide-react'
import { usePreviewSchedule } from '../api/hooks'
import { useTimeFormat } from '../lib/time'
import { Label, Spinner } from '../ui'

export const WHEN_EXAMPLES = [
  'Every hour',
  'Every 30 minutes',
  'Every day at 10:00am and 2:00pm',
  '45 minutes after competition start',
  '30 minutes before access closes',
]

// The "when" half of scheduling, set apart from the action fields above it
// and made honest: as you type an expression, it previews the actual next
// run times (server-side, the same schedule.NextFireAfter the dispatcher
// uses) so you confirm real times before committing, not a phrase you hope
// parsed. onValidityChange reports whether there's at least one real
// upcoming run, which gates the caller's Schedule button.
export function WhenField({
  buildId,
  when,
  onWhenChange,
  onValidityChange,
}: {
  buildId: string
  when: string
  onWhenChange: (v: string) => void
  onValidityChange: (valid: boolean) => void
}) {
  const fmt = useTimeFormat()
  const [debounced, setDebounced] = useState(when)

  useEffect(() => {
    const t = setTimeout(() => setDebounced(when), 400)
    return () => clearTimeout(t)
  }, [when])

  const { data: preview, isFetching } = usePreviewSchedule(buildId, debounced)
  const occurrences = preview?.occurrences ?? []
  const hasRuns = !!preview?.valid && occurrences.length > 0

  // Report validity: only "yes" once a settled preview shows real runs, and
  // only for the expression currently typed (not a stale debounce).
  useEffect(() => {
    onValidityChange(hasRuns && debounced === when)
  }, [hasRuns, debounced, when, onValidityChange])

  return (
    <div className="mt-2 rounded-token-lg border border-border bg-surface-sunken/40 p-3">
      <div className="mb-2 flex items-center gap-1.5 text-xs font-semibold uppercase tracking-wide text-fg-muted">
        <CalendarClock size={13} /> Schedule
      </div>

      <Label>When</Label>
      <input
        list="when-field-examples"
        value={when}
        onChange={(e) => onWhenChange(e.target.value)}
        placeholder="e.g. Every 30 minutes"
        className="h-9 w-full rounded-token border border-border bg-surface-raised px-2.5 text-sm text-fg placeholder:text-fg-subtle outline-none focus-visible:border-accent focus-visible:outline-2 focus-visible:outline-offset-0 focus-visible:outline-accent-soft"
      />
      <datalist id="when-field-examples">
        {WHEN_EXAMPLES.map((ex) => (
          <option key={ex} value={ex} />
        ))}
      </datalist>
      <div className="mt-1 text-xs text-fg-subtle">
        Recurring ("every…") or anchored to an event ("…after competition start", "…before access closes").
      </div>

      {/* Live preview of the real next run times -- confirm before scheduling. */}
      {when.trim() !== '' && (
        <div className="mt-3 border-t border-border pt-2.5">
          {isFetching || debounced !== when ? (
            <div className="flex items-center gap-2 text-xs text-fg-muted">
              <Spinner /> Checking schedule…
            </div>
          ) : preview && !preview.valid ? (
            <div className="flex items-start gap-1.5 text-xs text-danger">
              <TriangleAlert size={13} className="mt-0.5 shrink-0" />
              <span>{preview.error || "Couldn't read that schedule."}</span>
            </div>
          ) : hasRuns ? (
            <div>
              <div className="mb-1.5 flex items-center gap-1.5 text-xs font-medium text-success">
                <CircleCheck size={13} />
                {preview!.fires_once ? 'Runs once, at:' : `Next ${occurrences.length} run${occurrences.length === 1 ? '' : 's'}:`}
              </div>
              <ul className="flex flex-col gap-0.5">
                {occurrences.map((iso) => (
                  <li key={iso} className="font-mono text-xs text-fg">
                    {fmt.dateTime(iso)}
                  </li>
                ))}
                {!preview!.fires_once && <li className="text-xs text-fg-subtle">…and so on</li>}
              </ul>
            </div>
          ) : (
            <div className="flex items-start gap-1.5 text-xs text-warning">
              <TriangleAlert size={13} className="mt-0.5 shrink-0" />
              <span>No upcoming runs — its anchor may have already passed, so nothing would ever fire.</span>
            </div>
          )}
        </div>
      )}
    </div>
  )
}
