import { useState } from 'react'
import { ArrowLeft, ArrowRight, CircleCheck, CircleDashed, CircleX, Loader, ShieldCheck, TriangleAlert } from 'lucide-react'
import { useRevalidate, useValidationReport, useValidationReports } from '../api/hooks'
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
// result, and whether its status changed. Failing runs open by default.
export function ValidationRunCard({ run, showHost = true }: { run: ValidationRun; showHost?: boolean }) {
  const changed = run.status_after && run.status_after !== run.status_before
  const failedChecks = run.steps.reduce((n, s) => n + s.checks.filter((c) => c.passed === false && !s.ignore_errors).length, 0)
  const totalChecks = run.steps.reduce((n, s) => n + s.checks.length, 0)
  return (
    <details open={run.status === 'failed' || run.status === 'running'} className="group rounded-token border border-border bg-surface-raised">
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

function ReportSummary({ report }: { report: ValidationReport }) {
  return (
    <div className="mb-3 flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-fg-muted">
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
      <span className="ml-auto text-fg-subtle">{new Date(report.created_at).toLocaleString()}</span>
    </div>
  )
}

// ValidationReportBody is a whole revalidation: a summary, then every host's
// run, failures first. Live while any host is still running.
export function ValidationReportBody({ buildId, reportId }: { buildId: string; reportId: string }) {
  const { data: report, isLoading, error } = useValidationReport(buildId, reportId)
  if (isLoading) {
    return (
      <div className="flex items-center gap-2 text-xs text-fg-muted">
        <Spinner /> Loading the report…
      </div>
    )
  }
  if (error || !report) return <div className="rounded-token border border-danger bg-danger-soft p-2 text-xs text-danger">Couldn't load this report.</div>
  const runs = [...(report.runs ?? [])].sort((a, b) => (RUN_ORDER[a.status] ?? 9) - (RUN_ORDER[b.status] ?? 9))
  return (
    <div>
      <ReportSummary report={report} />
      <div className="flex max-h-[60vh] flex-col gap-2 overflow-auto">
        {runs.map((r) => (
          <ValidationRunCard key={r.id} run={r} />
        ))}
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
        className="max-w-2xl"
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
      className="max-w-2xl"
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
