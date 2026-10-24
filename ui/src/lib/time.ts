import { useMe } from '../api/hooks'

// One place that decides how a timestamp renders, so every screen honors
// the viewer's own timezone preference (direct product feedback: timezone
// should be per-user and shown across the UI). The account's IANA zone
// comes from `me.timezone`; empty falls back to the browser's local zone,
// which is the prior behavior. An invalid stored zone would make
// Intl.DateTimeFormat throw, so each formatter guards and degrades to
// local rather than crashing a page.
export function useTimeFormat() {
  const { data: me } = useMe()
  const tz = me?.timezone || undefined

  function opts(base: Intl.DateTimeFormatOptions): Intl.DateTimeFormatOptions {
    return tz ? { ...base, timeZone: tz } : base
  }
  function fmt(iso: string, o: Intl.DateTimeFormatOptions): string {
    const d = new Date(iso)
    try {
      return d.toLocaleString([], opts(o))
    } catch {
      return d.toLocaleString([], o)
    }
  }

  return {
    /** IANA zone in effect, or undefined for browser-local. */
    tz,
    /** Date + time, e.g. "9/27/2026, 2:14:03 PM". */
    dateTime: (iso: string) => fmt(iso, { dateStyle: 'short', timeStyle: 'medium' }),
    /** Time only, e.g. "2:14:03 PM". */
    time: (iso: string) => fmt(iso, { timeStyle: 'medium' }),
    /** Short time, e.g. "2:14 PM". */
    timeShort: (iso: string) => fmt(iso, { hour: '2-digit', minute: '2-digit' }),
    /** Date only, e.g. "9/27/2026". */
    date: (iso: string) => fmt(iso, { dateStyle: 'short' }),
  }
}
