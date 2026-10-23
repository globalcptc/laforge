import { useEffect, useMemo, useState } from 'react'
import { AlertTriangle, Check, Copy, Download, Eye, EyeOff } from 'lucide-react'
import { useExternalAccessExport } from '../api/hooks'
import type { ExternalAccessExportRow } from '../api/types'
import { Button, Field, Input, Modal, Spinner, Textarea, cn, useToast } from '../ui'

// ScoringExportWizard turns a build's public endpoints into the JSON the
// scoring app's Team Connections bulk import takes (scoring.cp.tc
// /admin/tools/connections; shape in the scoring repo's
// docs/examples/README.md): one entry per team, each with
// { host, url, username, password } per endpoint. url is the bare ip:port a
// captain pastes into an RDP client; username/password are the host's current
// admin login (the last admin set_password LaForge applied). Three steps:
// pick endpoints (usually leaving our own, like OutOfScope, out), fill in the
// scoring season/competition and team numbering, then review and download.

type Step = 0 | 1 | 2
const STEP_TITLES = ['Endpoints', 'Scoring Details', 'Review & Export'] as const

interface ScoringDetails {
  season: string
  competition: string
  teamOffset: number
  instructions: string
}

interface ScoringHost {
  host: string
  url: string
  username: string
  password: string
}

interface ScoringEntry {
  season: string
  competition: string
  team_number: number
  instructions?: string
  hosts: ScoringHost[]
}

const endpointKey = (r: ExternalAccessExportRow) => `${r.deployed_object_id}:${r.protocol}:${r.internal_port}`
// Only a host with a login can be imported -- scoring requires a username and
// password on every entry, and a container has no admin account.
const importable = (r: ExternalAccessExportRow) => r.kind === 'host' && r.username !== '' && r.password !== ''

// Season/competition/offset are remembered per build, so re-exporting after a
// rebuild doesn't mean retyping them. Best-effort: storage can be unavailable.
function loadDetails(buildId: string): ScoringDetails {
  const blank = { season: '', competition: '', teamOffset: 0, instructions: '' }
  try {
    const raw = localStorage.getItem(`laforge.scoringExport.${buildId}`)
    return raw ? { ...blank, ...JSON.parse(raw) } : blank
  } catch {
    return blank
  }
}
function saveDetails(buildId: string, d: ScoringDetails) {
  try {
    localStorage.setItem(`laforge.scoringExport.${buildId}`, JSON.stringify(d))
  } catch {
    /* storage unavailable */
  }
}

export function ScoringExportWizard({ buildId, open, onClose }: { buildId: string; open: boolean; onClose: () => void }) {
  const { data: rows, isLoading, error } = useExternalAccessExport(buildId, open)
  const [step, setStep] = useState<Step>(0)
  const [selected, setSelected] = useState<Set<string> | null>(null)
  const [details, setDetails] = useState<ScoringDetails>(() => loadDetails(buildId))

  // Default selection: every importable endpoint, once the rows arrive.
  useEffect(() => {
    if (rows && selected === null) setSelected(new Set(rows.filter(importable).map(endpointKey)))
  }, [rows, selected])
  useEffect(() => saveDetails(buildId, details), [buildId, details])

  const sel = selected ?? new Set<string>()
  const entries = useMemo(() => buildEntries(rows ?? [], sel, details), [rows, sel, details])
  const detailsValid = details.season.trim() !== '' && details.competition.trim() !== ''
  const canNext = step === 0 ? sel.size > 0 : step === 1 ? detailsValid : false

  function close() {
    setStep(0)
    onClose()
  }

  return (
    <Modal
      open={open}
      onClose={close}
      title="Export for Scoring"
      description="Builds the JSON for the scoring app's Team Connections import."
      className="w-[min(94vw,52rem)]"
      footer={
        <>
          {step > 0 && (
            <Button variant="secondary" onClick={() => setStep((s) => (s - 1) as Step)}>
              Back
            </Button>
          )}
          {step < 2 && (
            <Button variant="primary" disabled={!canNext} onClick={() => setStep((s) => (s + 1) as Step)}>
              Next
            </Button>
          )}
        </>
      }
    >
      <WizardSteps current={step} onSelect={(i) => i < step && setStep(i)} />
      <div className="mt-3">
        {isLoading ? (
          <div className="flex items-center gap-2 text-sm text-fg-muted">
            <Spinner /> Loading endpoints…
          </div>
        ) : error ? (
          <p className="text-sm text-danger">Couldn't load the endpoints: {(error as Error).message}</p>
        ) : step === 0 ? (
          <EndpointsStep rows={rows ?? []} selected={sel} onChange={setSelected} />
        ) : step === 1 ? (
          <DetailsStep details={details} onChange={setDetails} teams={teamsOf(rows ?? [])} />
        ) : (
          <ReviewStep entries={entries} rows={rows ?? []} selected={sel} buildId={buildId} />
        )}
      </div>
    </Modal>
  )
}

function WizardSteps({ current, onSelect }: { current: Step; onSelect: (i: Step) => void }) {
  return (
    <ol className="flex flex-wrap items-center gap-2">
      {STEP_TITLES.map((title, i) => {
        const done = i < current
        const active = i === current
        return (
          <li key={title} className="flex items-center gap-2">
            <button
              type="button"
              disabled={!done}
              onClick={() => onSelect(i as Step)}
              className={cn(
                'flex items-center gap-2 rounded-token px-1.5 py-1 text-xs disabled:cursor-default',
                active ? 'text-fg' : done ? 'text-fg-muted hover:text-fg' : 'text-fg-subtle',
              )}
            >
              <span
                className={cn(
                  'flex size-5 items-center justify-center rounded-full border text-[11px] font-semibold',
                  active && 'border-accent bg-accent text-fg-inverted',
                  done && 'border-accent bg-accent-soft text-accent-fg',
                  !active && !done && 'border-border',
                )}
              >
                {done ? <Check size={11} /> : i + 1}
              </span>
              <span className={cn(active && 'font-medium')}>{title}</span>
            </button>
            {i < STEP_TITLES.length - 1 && <span className="h-px w-5 bg-border" />}
          </li>
        )
      })}
    </ol>
  )
}

// ── Step 1: endpoints ────────────────────────────────────────────────────────

function EndpointsStep({
  rows,
  selected,
  onChange,
}: {
  rows: ExternalAccessExportRow[]
  selected: Set<string>
  onChange: (s: Set<string>) => void
}) {
  if (rows.length === 0) return <p className="text-sm text-fg-muted">This build has no public endpoints yet.</p>

  // The same placement exists on every team, so toggling a host name here
  // applies across all teams -- the quick way to leave e.g. OutOfScope out.
  const byHost = new Map<string, ExternalAccessExportRow[]>()
  for (const r of rows) byHost.set(r.host, [...(byHost.get(r.host) ?? []), r])
  const hostNames = [...byHost.keys()].sort()

  function setAll(list: ExternalAccessExportRow[], on: boolean) {
    const next = new Set(selected)
    for (const r of list) {
      if (!importable(r)) continue
      if (on) next.add(endpointKey(r))
      else next.delete(endpointKey(r))
    }
    onChange(next)
  }
  const stateOf = (list: ExternalAccessExportRow[]) => {
    const usable = list.filter(importable)
    const on = usable.filter((r) => selected.has(endpointKey(r))).length
    return { checked: usable.length > 0 && on === usable.length, indeterminate: on > 0 && on < usable.length, disabled: usable.length === 0 }
  }

  const teams = teamsOf(rows)
  const unapplied = rows.filter((r) => importable(r) && !r.password_applied).length

  return (
    <div className="flex flex-col gap-4">
      <section>
        <h3 className="mb-1.5 text-xs font-semibold text-fg">Hosts (All Teams)</h3>
        <div className="flex flex-wrap gap-2">
          {hostNames.map((name) => {
            const list = byHost.get(name)!
            const st = stateOf(list)
            return (
              <label
                key={name}
                className={cn(
                  'flex items-center gap-1.5 rounded-token border px-2 py-1 text-xs',
                  st.checked || st.indeterminate ? 'border-accent bg-accent-soft text-fg' : 'border-border text-fg-muted',
                  st.disabled && 'opacity-50',
                )}
                title={st.disabled ? 'No login to export (containers have no admin account)' : undefined}
              >
                <TriCheckbox {...st} onChange={(on) => setAll(list, on)} />
                {name}
                <span className="text-fg-subtle">×{new Set(list.map((r) => r.team)).size}</span>
              </label>
            )
          })}
        </div>
      </section>

      {unapplied > 0 && (
        <p className="flex items-start gap-1.5 rounded-token border border-warning/40 bg-warning-soft px-2.5 py-1.5 text-xs text-fg">
          <AlertTriangle size={13} className="mt-0.5 shrink-0 text-warning" />
          {unapplied} endpoint{unapplied === 1 ? "'s" : "s'"} password hasn't been applied on the box yet (its set_password step
          hasn't run). The planned password is exported; re-export once the build finishes to be sure.
        </p>
      )}

      <section className="flex flex-col gap-2">
        {teams.map((team) => {
          const list = rows.filter((r) => r.team === team)
          const st = stateOf(list)
          return (
            <div key={team} className="overflow-hidden rounded-token border border-border">
              <label className="flex items-center gap-2 bg-cptc-violet px-3 py-1.5 text-xs font-semibold text-cptc-violet-fg">
                <TriCheckbox {...st} onChange={(on) => setAll(list, on)} />
                Team {team}
                <span className="ml-auto font-normal opacity-80">
                  {list.filter((r) => selected.has(endpointKey(r))).length} of {list.length}
                </span>
              </label>
              <ul className="divide-y divide-border">
                {list.map((r) => {
                  const k = endpointKey(r)
                  const ok = importable(r)
                  return (
                    <li key={k} className={cn('flex items-center gap-3 px-3 py-1.5 text-xs', !ok && 'opacity-50')}>
                      <input
                        type="checkbox"
                        disabled={!ok}
                        checked={selected.has(k)}
                        onChange={(e) => setAll([r], e.target.checked)}
                      />
                      <span className="w-40 truncate font-medium text-fg">{r.host}</span>
                      <span className="w-24 font-mono text-fg-muted">
                        {r.protocol}/{r.internal_port}
                      </span>
                      <span className="font-mono text-fg">{r.public_address}</span>
                      <span className="ml-auto flex items-center gap-1 text-fg-muted">
                        {!ok ? (
                          'No login'
                        ) : (
                          <>
                            {r.username}
                            {!r.password_applied && (
                              <span title="Password not applied on the box yet">
                                <AlertTriangle size={12} className="text-warning" />
                              </span>
                            )}
                          </>
                        )}
                      </span>
                    </li>
                  )
                })}
              </ul>
            </div>
          )
        })}
      </section>
    </div>
  )
}

function TriCheckbox({
  checked,
  indeterminate,
  disabled,
  onChange,
}: {
  checked: boolean
  indeterminate: boolean
  disabled: boolean
  onChange: (on: boolean) => void
}) {
  return (
    <input
      type="checkbox"
      ref={(el) => {
        if (el) el.indeterminate = indeterminate
      }}
      checked={checked}
      disabled={disabled}
      onChange={(e) => onChange(e.target.checked)}
    />
  )
}

// ── Step 2: scoring details ──────────────────────────────────────────────────

function DetailsStep({
  details,
  onChange,
  teams,
}: {
  details: ScoringDetails
  onChange: (d: ScoringDetails) => void
  teams: number[]
}) {
  const set = (patch: Partial<ScoringDetails>) => onChange({ ...details, ...patch })
  const first = teams[0]
  const last = teams[teams.length - 1]
  return (
    <div className="flex flex-col gap-3">
      <div className="grid gap-3 sm:grid-cols-2">
        <Field label="Season" hint="The scoring season's shortname, e.g. cptc12.">
          <Input value={details.season} onChange={(e) => set({ season: e.target.value })} placeholder="cptc12" />
        </Field>
        <Field label="Competition" hint="The scoring competition's shortname, e.g. us-socal.">
          <Input value={details.competition} onChange={(e) => set({ competition: e.target.value })} placeholder="us-socal" />
        </Field>
      </div>
      <Field
        label="Team Number Offset"
        hint={
          first !== undefined
            ? `Added to each LaForge team number: team ${first} → ${first + details.teamOffset}${
                last !== first ? `, team ${last} → ${last + details.teamOffset}` : ''
              } in scoring.`
            : 'Added to each LaForge team number.'
        }
      >
        <Input
          type="number"
          className="w-32"
          value={details.teamOffset}
          onChange={(e) => set({ teamOffset: Number.isFinite(e.target.valueAsNumber) ? e.target.valueAsNumber : 0 })}
        />
      </Field>
      <Field
        label="Instructions (Optional)"
        hint="Markdown shown to every team's captain. Leave blank to keep whatever instructions scoring already has."
      >
        <Textarea rows={5} value={details.instructions} onChange={(e) => set({ instructions: e.target.value })} />
      </Field>
    </div>
  )
}

// ── Step 3: review & export ──────────────────────────────────────────────────

function ReviewStep({
  entries,
  rows,
  selected,
  buildId,
}: {
  entries: ScoringEntry[]
  rows: ExternalAccessExportRow[]
  selected: Set<string>
  buildId: string
}) {
  const toast = useToast()
  const [reveal, setReveal] = useState(false)
  const json = JSON.stringify(entries, null, 2)
  const shown = reveal
    ? json
    : JSON.stringify(
        entries.map((e) => ({ ...e, hosts: e.hosts.map((h) => ({ ...h, password: '••••••••' })) })),
        null,
        2,
      )
  const hostCount = entries.reduce((n, e) => n + e.hosts.length, 0)
  const unapplied = rows.filter((r) => selected.has(endpointKey(r)) && !r.password_applied).length

  function download() {
    const blob = new Blob([json + '\n'], { type: 'application/json' })
    const a = document.createElement('a')
    a.href = URL.createObjectURL(blob)
    a.download = `scoring-connections-${buildId.slice(0, 8)}.json`
    a.click()
    URL.revokeObjectURL(a.href)
  }
  async function copy() {
    try {
      await navigator.clipboard.writeText(json)
      toast({ title: 'Copied the scoring JSON', tone: 'success' })
    } catch {
      toast({ title: "Couldn't copy", description: 'Clipboard access was blocked; use Download instead.', tone: 'danger' })
    }
  }

  return (
    <div className="flex flex-col gap-3">
      <p className="text-sm text-fg">
        <span className="font-semibold">{entries.length}</span> team{entries.length === 1 ? '' : 's'},{' '}
        <span className="font-semibold">{hostCount}</span> endpoint{hostCount === 1 ? '' : 's'}. In scoring, open{' '}
        <span className="font-mono text-xs">Admin → Tools → Connections</span>, choose the phase (Test or Live), and upload this
        file. Each team's existing connections for that phase are replaced.
      </p>
      {unapplied > 0 && (
        <p className="flex items-start gap-1.5 rounded-token border border-warning/40 bg-warning-soft px-2.5 py-1.5 text-xs text-fg">
          <AlertTriangle size={13} className="mt-0.5 shrink-0 text-warning" />
          {unapplied} selected endpoint{unapplied === 1 ? ' uses a password' : 's use passwords'} not yet applied on the box.
        </p>
      )}
      <div className="flex items-center gap-2">
        <Button variant="primary" onClick={download}>
          <Download size={14} /> Download JSON
        </Button>
        <Button variant="brand" onClick={copy}>
          <Copy size={14} /> Copy JSON
        </Button>
        <Button variant="ghost" className="ml-auto" onClick={() => setReveal((v) => !v)}>
          {reveal ? <EyeOff size={14} /> : <Eye size={14} />} {reveal ? 'Hide' : 'Show'} Passwords
        </Button>
      </div>
      <pre className="max-h-80 overflow-auto rounded-token border border-border bg-surface-sunken p-3 font-mono text-[11px] leading-relaxed text-fg">
        {shown}
      </pre>
    </div>
  )
}

// ── shaping ──────────────────────────────────────────────────────────────────

function teamsOf(rows: ExternalAccessExportRow[]): number[] {
  return [...new Set(rows.map((r) => r.team))].sort((a, b) => a - b)
}

// buildEntries shapes the selected endpoints into scoring's import format. A
// host with more than one public port gets the port in its label so the
// captain can tell its entries apart; a team with nothing selected is left out
// (scoring rejects an entry with no hosts).
function buildEntries(rows: ExternalAccessExportRow[], selected: Set<string>, d: ScoringDetails): ScoringEntry[] {
  const portsPerObject = new Map<string, number>()
  for (const r of rows) portsPerObject.set(r.deployed_object_id, (portsPerObject.get(r.deployed_object_id) ?? 0) + 1)
  const instructions = d.instructions.trim()
  const out: ScoringEntry[] = []
  for (const team of teamsOf(rows)) {
    const hosts = rows
      .filter((r) => r.team === team && importable(r) && selected.has(endpointKey(r)))
      .map((r) => ({
        host: (portsPerObject.get(r.deployed_object_id) ?? 0) > 1 ? `${r.host} (${r.protocol}/${r.internal_port})` : r.host,
        url: r.public_address,
        username: r.username,
        password: r.password,
      }))
    if (hosts.length === 0) continue
    out.push({
      season: d.season.trim(),
      competition: d.competition.trim(),
      team_number: team + d.teamOffset,
      ...(instructions ? { instructions } : {}),
      hosts,
    })
  }
  return out
}
