import { useMemo, useState } from 'react'
import {
  ArrowLeft,
  ArrowRight,
  ChevronDown,
  ChevronsDownUp,
  ChevronsUpDown,
  CircleCheck,
  CircleDashed,
  CircleX,
  Loader,
  Network,
  ShieldCheck,
  TriangleAlert,
  Users,
} from 'lucide-react'
import { useDeployedObjects, useRevalidate, useValidationReport, useValidationReports } from '../api/hooks'
import { ApiError } from '../api/client'
import type { RevalidateResponse, ValidationCheck, ValidationReport, ValidationRun, ValidationRunStatus } from '../api/types'
import { Badge, Button, Modal, Spinner, cn, useToast } from '../ui'
import { statusLabel } from './StatusBadge'

// Revalidation UI: re-run every validator on hosts as they are and show what
// each check says now. Nothing on the host changes except its status, and only
// between Finished and Invalid.

const RUN_STYLE: Record<ValidationRunStatus, { tone: 'info' | 'success' | 'danger' | 'neutral' | 'warning'; label: string }> = {
  running: { tone: 'info', label: 'Running' },
  passed: { tone: 'success', label: 'Passed' },
  failed: { tone: 'danger', label: 'Failed' },
  canceled: { tone: 'neutral', label: 'Canceled' },
  timed_out: { tone: 'warning', label: 'Timed Out' },
}

export function RunStatusBadge({ status }: { status: ValidationRunStatus }) {
  const s = RUN_STYLE[status] ?? { tone: 'neutral' as const, label: status }
  return (
    <Badge tone={s.tone} className="rounded-full py-0.5">
      {status === 'running' && <Loader size={11} className="animate-spin" />}
      {s.label}
    </Badge>
  )
}

// "port_open port=22" -- a check's kind with its arguments.
function checkLabel(c: ValidationCheck): string {
  const args = Object.entries(c.args ?? {})
    .map(([k, v]) => `${k}=${typeof v === 'string' ? v : JSON.stringify(v)}`)
    .join(' ')
  return args ? `${c.kind} ${args}` : c.kind
}

function CheckRow({ check, tolerated }: { check: ValidationCheck; tolerated: boolean }) {
  const Icon = check.passed === null ? CircleDashed : check.passed ? CircleCheck : CircleX
  const color = check.passed === null ? 'text-fg-subtle' : check.passed ? 'text-success' : tolerated ? 'text-warning' : 'text-danger'
  return (
    <div className="flex items-start gap-2 py-0.5 text-xs">
      <Icon size={13} className={cn('mt-0.5 shrink-0', color)} />
      <div className="min-w-0">
        <span className="font-mono text-fg">{checkLabel(check)}</span>
        {check.message && <span className="ml-2 text-fg-muted">{check.message}</span>}
      </div>
    </div>
  )
}

// ValidationRunCard is one host's revalidation: every check it ran, with its
// result, and whether its status changed. Failing runs open by default; pass
// open/onOpenChange to control it (the report's Collapse All does).
export function ValidationRunCard({
  run,
  showHost = true,
  open,
  onOpenChange,
}: {
  run: ValidationRun
  showHost?: boolean
  open?: boolean
  onOpenChange?: (open: boolean) => void
}) {
  const changed = run.status_after && run.status_after !== run.status_before
  const failedChecks = run.steps.reduce((n, s) => n + s.checks.filter((c) => c.passed === false && !s.ignore_errors).length, 0)
  const totalChecks = run.steps.reduce((n, s) => n + s.checks.length, 0)
  return (
    <details
      open={open ?? (run.status === 'failed' || run.status === 'running')}
      onToggle={onOpenChange && ((e) => onOpenChange(e.currentTarget.open))}
      className="group rounded-token border border-border bg-surface-raised"
    >
      <summary className="flex cursor-pointer list-none items-center gap-2 px-3 py-2 text-sm">
        {showHost ? (
          <span className="min-w-0 flex-1 truncate font-medium text-fg">
            {run.name ?? run.object_id}
            {run.team_number !== undefined && <span className="ml-1.5 text-xs font-normal text-fg-subtle">team {run.team_number}</span>}
          </span>
        ) : (
          <span className="min-w-0 flex-1 truncate text-xs text-fg-muted">{new Date(run.created_at).toLocaleString()}</span>
        )}
        {changed && (
          <span className="flex items-center gap-1 text-xs text-fg-muted" title="This revalidation changed the host's status">
            {statusLabel(run.status_before)} <ArrowRight size={11} /> <span className="font-medium text-fg">{statusLabel(run.status_after!)}</span>
          </span>
        )}
        {run.status !== 'running' && totalChecks > 0 && (
          <span className="text-xs text-fg-subtle">
            {failedChecks > 0 ? `${failedChecks} of ${totalChecks} failed` : `${totalChecks} passed`}
          </span>
        )}
        <RunStatusBadge status={run.status} />
      </summary>
      <div className="border-t border-border px-3 py-2">
        {run.status === 'timed_out' && <p className="mb-1 text-xs text-warning">The agent didn't run every check in time; the ones it didn't run were dropped.</p>}
        {run.status === 'canceled' && <p className="mb-1 text-xs text-fg-muted">The host was rebuilt or restarted while this ran.</p>}
        {run.steps.length === 0 && <p className="text-xs text-fg-muted">No checks left to show.</p>}
        {run.steps.map((s) => (
          <div key={s.step_index} className="py-0.5">
            {s.checks.map((c, i) => (
              <CheckRow key={i} check={c} tolerated={s.ignore_errors} />
            ))}
            {s.ignore_errors && s.checks.some((c) => c.passed === false) && (
              <p className="ml-5 text-[11px] text-fg-subtle">Its step tolerates errors (ignore_errors), so this doesn't fail the host.</p>
            )}
            {s.error && s.checks.every((c) => c.passed === null) && s.status !== 'pending' && s.status !== 'leased' && (
              <p className="ml-5 text-[11px] text-danger">{s.error}</p>
            )}
          </div>
        ))}
      </div>
    </details>
  )
}

const RUN_ORDER: Record<string, number> = { failed: 0, running: 1, timed_out: 2, canceled: 3, passed: 4 }

// Bar colours per run status, matching RunStatusBadge's tones, in the order the
// segments are drawn: done-and-good first, then problems, then still running.
const BAR_ORDER: ValidationRunStatus[] = ['passed', 'failed', 'timed_out', 'canceled', 'running']
const BAR_FILL: Record<ValidationRunStatus, string> = {
  passed: 'bg-success',
  failed: 'bg-danger',
  timed_out: 'bg-warning',
  canceled: 'bg-border-strong',
  running: 'bg-info animate-pulse',
}

// RunProgress is a group's runs as one stacked bar -- how many have finished
// and how they came out -- with a "done of total" count, like the hosts page's
// per-network bars.
function RunProgress({ runs, className }: { runs: ValidationRun[]; className?: string }) {
  const counts = new Map<ValidationRunStatus, number>()
  for (const r of runs) counts.set(r.status, (counts.get(r.status) ?? 0) + 1)
  const total = runs.length
  const done = total - (counts.get('running') ?? 0)
  const failed = counts.get('failed') ?? 0
  const legend = BAR_ORDER.filter((st) => counts.has(st))
    .map((st) => `${RUN_STYLE[st].label}: ${counts.get(st)}`)
    .join(' · ')
  return (
    <div className={cn('flex shrink-0 items-center gap-2', className)} title={legend}>
      {failed > 0 && <span className="text-[11px] font-medium text-danger">{failed} failed</span>}
      <div className="flex h-2 w-32 overflow-hidden rounded-full bg-surface-sunken">
        {BAR_ORDER.filter((st) => counts.has(st)).map((st) => (
          <div key={st} className={BAR_FILL[st]} style={{ width: `${((counts.get(st) ?? 0) / total) * 100}%` }} />
        ))}
      </div>
      <span className="w-12 text-right text-[11px] tabular-nums text-fg-subtle">
        {done}/{total}
      </span>
    </div>
  )
}

function ReportSummary({ report, runs }: { report: ValidationReport; runs: ValidationRun[] }) {
  return (
    <div className="flex min-w-0 flex-1 flex-wrap items-center gap-x-4 gap-y-1 text-xs text-fg-muted">
      <span>
        {report.hosts} host{report.hosts === 1 ? '' : 's'}
      </span>
      {report.passed > 0 && <span className="text-success">{report.passed} passed</span>}
      {report.failed > 0 && <span className="text-danger">{report.failed} failed</span>}
      {report.running > 0 && (
        <span className="flex items-center gap-1 text-info">
          <Spinner /> {report.running} running
        </span>
      )}
      {report.incomplete > 0 && <span className="text-warning">{report.incomplete} incomplete</span>}
      <span className="text-fg-subtle">{new Date(report.created_at).toLocaleString()}</span>
      {runs.length > 0 && <RunProgress runs={runs} className="ml-auto" />}
    </div>
  )
}

type NetGroup = { key: string; name: string; runs: ValidationRun[] }
type TeamGroup = { key: string; label: string; runs: ValidationRun[]; networks: NetGroup[] }

const byRunOrder = (a: ValidationRun, b: ValidationRun) =>
  (RUN_ORDER[a.status] ?? 9) - (RUN_ORDER[b.status] ?? 9) || (a.name ?? '').localeCompare(b.name ?? '')

// groupRuns arranges a report's runs as team -> network -> host, teams in
// number order, networks by name, hosts failures first. A host's network comes
// from the build's objects (a run doesn't carry it).
function groupRuns(runs: ValidationRun[], networkOf: Map<string, string>): TeamGroup[] {
  const teams = new Map<string, TeamGroup>()
  for (const r of runs) {
    const tKey = r.team_number === undefined ? 'none' : String(r.team_number)
    let team = teams.get(tKey)
    if (!team) {
      team = { key: tKey, label: r.team_number === undefined ? 'No Team' : `Team ${r.team_number}`, runs: [], networks: [] }
      teams.set(tKey, team)
    }
    team.runs.push(r)
    const netName = networkOf.get(r.object_id) ?? ''
    let net = team.networks.find((n) => n.name === netName)
    if (!net) {
      net = { key: `${tKey}:${netName}`, name: netName, runs: [] }
      team.networks.push(net)
    }
    net.runs.push(r)
  }
  const out = [...teams.values()].sort((a, b) => (a.key === 'none' ? 1 : b.key === 'none' ? -1 : Number(a.key) - Number(b.key)))
  for (const t of out) {
    t.networks.sort((a, b) => a.name.localeCompare(b.name))
    for (const n of t.networks) n.runs.sort(byRunOrder)
  }
  return out
}

// ValidationReportBody is a whole revalidation: a summary, then its runs grouped
// by team and network, each group with a progress bar so the report reads at a
// glance without opening every host. Networks start collapsed unless one of
// their hosts failed. Live while any host is still running.
export function ValidationReportBody({ buildId, reportId }: { buildId: string; reportId: string }) {
  const { data: report, isLoading, error } = useValidationReport(buildId, reportId)
  const { data: objects } = useDeployedObjects(buildId)
  const networkOf = useMemo(() => new Map((objects ?? []).map((o) => [o.id, o.network_name ?? ''])), [objects])
  const groups = useMemo(() => groupRuns(report?.runs ?? [], networkOf), [report?.runs, networkOf])
  const [collapsedTeams, setCollapsedTeams] = useState<Set<string>>(new Set())
  // Networks the user has opened or closed by hand; the rest follow the default
  // (open only when something in them failed).
  const [netOverride, setNetOverride] = useState<Map<string, boolean>>(new Map())
  // Host cards the user opened or closed, or Collapse All closed; the rest open
  // when they failed or are still running.
  const [cardOverride, setCardOverride] = useState<Map<string, boolean>>(new Map())

  if (isLoading) {
    return (
      <div className="flex items-center gap-2 text-xs text-fg-muted">
        <Spinner /> Loading the report…
      </div>
    )
  }
  if (error || !report) return <div className="rounded-token border border-danger bg-danger-soft p-2 text-xs text-danger">Couldn't load this report.</div>

  const netOpen = (n: NetGroup) => netOverride.get(n.key) ?? n.runs.some((r) => r.status === 'failed')
  const allCollapsed = groups.length > 0 && groups.every((t) => collapsedTeams.has(t.key))
  function toggleTeam(key: string) {
    setCollapsedTeams((prev) => {
      const next = new Set(prev)
      if (next.has(key)) next.delete(key)
      else next.add(key)
      return next
    })
  }
  function toggleNet(n: NetGroup) {
    setNetOverride((prev) => new Map(prev).set(n.key, !netOpen(n)))
  }
  // Same as the hosts page: Collapse All closes teams, their networks and every
  // host's checks; Expand All opens the teams and leaves the rest closed, for
  // the overview.
  function collapseAll() {
    setCollapsedTeams(new Set(groups.map((t) => t.key)))
    setNetOverride(new Map(groups.flatMap((t) => t.networks.map((n) => [n.key, false] as const))))
    setCardOverride(new Map(groups.flatMap((t) => t.runs.map((r) => [r.id, false] as const))))
  }
  function setCardOpen(id: string, next: boolean) {
    setCardOverride((prev) => (prev.get(id) === next ? prev : new Map(prev).set(id, next)))
  }
  function expandAll() {
    setCollapsedTeams(new Set())
  }

  return (
    <div>
      <div className="sticky -top-3 z-10 -mx-4 -mt-3 mb-3 flex items-center gap-3 border-b border-border bg-surface-raised px-4 py-2">
        <ReportSummary report={report} runs={report.runs ?? []} />
        {groups.length > 0 && (
          <Button
            variant="secondary"
            size="icon"
            onClick={allCollapsed ? expandAll : collapseAll}
            title={allCollapsed ? 'Expand All' : 'Collapse All'}
            aria-label={allCollapsed ? 'Expand All' : 'Collapse All'}
          >
            {allCollapsed ? <ChevronsUpDown size={14} /> : <ChevronsDownUp size={14} />}
          </Button>
        )}
      </div>
      <div className="flex flex-col gap-2">
        {groups.map((team) => {
          const teamOpen = !collapsedTeams.has(team.key)
          return (
            <div key={team.key} className="rounded-token border border-border">
              <button
                type="button"
                onClick={() => toggleTeam(team.key)}
                className="flex w-full items-center gap-2 rounded-token bg-surface-sunken/60 px-3 py-2 text-left hover:bg-surface-hover"
              >
                <ChevronDown size={14} className={cn('shrink-0 text-fg-subtle transition-transform', !teamOpen && '-rotate-90')} />
                <Users size={14} className="shrink-0 text-fg-subtle" />
                <span className="text-sm font-semibold text-fg">{team.label}</span>
                <span className="text-xs text-fg-subtle">({team.runs.length})</span>
                <RunProgress runs={team.runs} className="ml-auto" />
              </button>
              {teamOpen && (
                <div className="flex flex-col gap-1 border-t border-border p-2">
                  {team.networks.map((net) => {
                    const open = netOpen(net)
                    return (
                      <div key={net.key}>
                        <button
                          type="button"
                          onClick={() => toggleNet(net)}
                          className="flex w-full items-center gap-2 rounded-token px-2 py-1 text-left hover:bg-surface-hover"
                        >
                          <ChevronDown size={12} className={cn('shrink-0 text-fg-subtle transition-transform', !open && '-rotate-90')} />
                          <Network size={13} className="shrink-0 text-fg-subtle" />
                          <span className="text-xs font-medium text-fg-muted">{net.name || 'No Network'}</span>
                          <span className="text-xs text-fg-subtle">({net.runs.length})</span>
                          <RunProgress runs={net.runs} className="ml-auto" />
                        </button>
                        {open && (
                          <div className="flex flex-col gap-1.5 py-1.5 pl-6">
                            {net.runs.map((r) => (
                              <ValidationRunCard
                                key={r.id}
                                run={r}
                                open={cardOverride.get(r.id) ?? (r.status === 'failed' || r.status === 'running')}
                                onOpenChange={(o) => setCardOpen(r.id, o)}
                              />
                            ))}
                          </div>
                        )}
                      </div>
                    )
                  })}
                </div>
              )}
            </div>
          )
        })}
      </div>
    </div>
  )
}

function SkippedList({ skipped }: { skipped: RevalidateResponse['skipped'] }) {
  if (skipped.length === 0) return null
  return (
    <div className="mb-3 rounded-token border border-warning bg-warning-soft p-2 text-xs text-fg">
      <div className="mb-1 flex items-center gap-1.5 font-medium">
        <TriangleAlert size={13} className="text-warning" /> {skipped.length} host{skipped.length === 1 ? '' : 's'} not revalidated
      </div>
      {skipped.map((h) => (
        <div key={h.object_id} className="flex justify-between gap-2 py-0.5">
          <span className="truncate">
            {h.name} <span className="text-fg-subtle">team {h.team_number}</span>
          </span>
          <span className="shrink-0 text-fg-muted">{h.reason}</span>
        </div>
      ))}
    </div>
  )
}

// RevalidateDialog confirms, starts the revalidation, then turns into its live
// report.
export function RevalidateDialog({ buildId, targets, onClose }: { buildId: string; targets: { id: string; name: string }[]; onClose: () => void }) {
  const revalidate = useRevalidate(buildId)
  const toast = useToast()
  const [result, setResult] = useState<RevalidateResponse | null>(null)

  async function run() {
    try {
      setResult(await revalidate.mutateAsync({ target: { ids: targets.map((t) => t.id) } }))
    } catch (e) {
      toast({ title: 'Revalidation Failed', description: e instanceof ApiError ? e.message : undefined, tone: 'danger', duration: 0 })
    }
  }

  if (result) {
    return (
      <Modal
        open
        onClose={onClose}
        className="w-[min(96vw,72rem)]"
        title="Validation Report"
        description="Every validator re-run on the host as it is. A failing check moves a Finished host to Invalid; all passing moves an Invalid host back to Finished."
        footer={<Button onClick={onClose}>Close</Button>}
      >
        <SkippedList skipped={result.skipped} />
        {result.report_id ? (
          <ValidationReportBody buildId={buildId} reportId={result.report_id} />
        ) : (
          <p className="text-xs text-fg-muted">Nothing was revalidated.</p>
        )}
      </Modal>
    )
  }

  return (
    <Modal
      open
      onClose={onClose}
      title={`Revalidate ${targets.length} Host${targets.length === 1 ? '' : 's'}`}
      description="Re-runs every validator on the selected hosts, on the boxes as they are, and reports what each check says now."
      footer={
        <>
          <Button variant="ghost" onClick={onClose}>
            Cancel
          </Button>
          <Button variant="primary" onClick={run} disabled={revalidate.isPending}>
            <ShieldCheck size={14} />
            {revalidate.isPending ? 'Starting…' : `Revalidate ${targets.length}`}
          </Button>
        </>
      }
    >
      <ul className="mb-3 list-disc space-y-0.5 pl-4 text-xs text-fg-muted">
        <li>Only validators run — no steps, no fixes, no restarts.</li>
        <li>The host's build and Steps are left as they are.</li>
        <li>
          The only change is the host's status: <span className="text-fg">Finished → Invalid</span> if a check fails,{' '}
          <span className="text-fg">Invalid → Finished</span> if every check passes.
        </li>
      </ul>
      <div className="max-h-40 overflow-auto rounded-token border border-border bg-surface-sunken p-2 text-xs text-fg-muted">
        {targets.map((t) => (
          <div key={t.id} className="truncate py-0.5">
            {t.name}
          </div>
        ))}
      </div>
    </Modal>
  )
}

// ValidationReportsDialog lists this build's past revalidations; one opens to
// its full report.
export function ValidationReportsDialog({ buildId, onClose }: { buildId: string; onClose: () => void }) {
  const { data: reports, isLoading } = useValidationReports(buildId)
  const [open, setOpen] = useState<string | null>(null)
  return (
    <Modal
      open
      onClose={onClose}
      className={open ? 'w-[min(96vw,72rem)]' : 'w-[min(92vw,42rem)]'}
      title={open ? 'Validation Report' : 'Validation Reports'}
      description={open ? undefined : 'Every revalidation run on this build, newest first.'}
      footer={
        <>
          {open && (
            <Button variant="ghost" onClick={() => setOpen(null)}>
              <ArrowLeft size={14} /> All Reports
            </Button>
          )}
          <Button onClick={onClose}>Close</Button>
        </>
      }
    >
      {open ? (
        <ValidationReportBody buildId={buildId} reportId={open} />
      ) : isLoading ? (
        <div className="flex items-center gap-2 text-xs text-fg-muted">
          <Spinner /> Loading…
        </div>
      ) : !reports?.length ? (
        <p className="text-xs text-fg-muted">No revalidations yet. Select hosts and use Revalidate to run one.</p>
      ) : (
        <div className="flex flex-col gap-1">
          {reports.map((r) => (
            <button
              key={r.report_id}
              type="button"
              onClick={() => setOpen(r.report_id)}
              className="flex items-center gap-3 rounded-token border border-border bg-surface-raised px-3 py-2 text-left text-sm hover:bg-surface-hover"
            >
              <span className="flex-1 text-fg">{new Date(r.created_at).toLocaleString()}</span>
              <span className="text-xs text-fg-muted">
                {r.hosts} host{r.hosts === 1 ? '' : 's'}
              </span>
              {r.failed > 0 && <Badge tone="danger">{r.failed} failed</Badge>}
              {r.passed > 0 && <Badge tone="success">{r.passed} passed</Badge>}
              {r.running > 0 && <Badge tone="info">{r.running} running</Badge>}
              {r.incomplete > 0 && <Badge tone="warning">{r.incomplete} incomplete</Badge>}
            </button>
          ))}
        </div>
      )}
    </Modal>
  )
}
