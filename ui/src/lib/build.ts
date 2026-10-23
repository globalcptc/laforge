// A torn-down, purged or discarded build is history: its infrastructure is gone
// (or, for a discarded planned build, never existed), so the build page shows it
// but offers no mutations (no access changes, no ad-hoc or scheduled tasks, no
// power actions, no teardown). Anything still in flight
// (deploying/building/tearing_down) or live (finished/failed) stays editable.
export function buildIsReadOnly(status?: string): boolean {
  return status === 'torn_down' || status === 'purged' || status === 'discarded'
}
