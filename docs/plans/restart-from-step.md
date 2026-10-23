# Plan: restart a host's build from a step

**Status:** proposal for `3.0`. **Written:** 2026-10-23, against `3.0` at `87d2e5ba`.

A host whose build stopped — a step failed, a validator didn't pass — can today only be
**rebuilt**: destroyed at the hoster and deployed again from nothing, every step re-run.
That's the right tool when the box is wrong; it's the wrong tool when step 7 of 12 failed
because a package mirror timed out, or because the script had a typo that's now fixed.
This adds **Restart from step**: pick an authored step, and the host runs that step and
everything after it again — in full, downloads included — on the box it already has.
Pick it once for every instance of that host across the build's teams.

## Contents

1. [What exists](#1-what-exists)
2. [The action](#2-the-action)
3. [What happens underneath](#3-what-happens-underneath)
4. [Dependents](#4-dependents)
5. [Many hosts at once](#5-many-hosts-at-once)
6. [API, UI, CLI](#6-api-ui-cli)
7. [Database changes](#7-database-changes)
8. [Edge cases and rules](#8-edge-cases-and-rules)
9. [Phases](#9-phases)
10. [Open decisions](#10-open-decisions)

## 1. What exists

The pieces this builds on, as the code has them today:

- **Steps become `agent_task` rows** when a host's box is up
  (`internal/orchestrator/materialize_steps.go`): `gateway.ExpandSteps` turns each
  authored step into one or more commands — a `script:` step is *write file* + *execute*
  (+ *validate*) — numbered by `step_index`, with `PlannedCommand.Group` saying which
  authored step each came from. The group isn't stored; the steps API
  (`internal/api/builds_ui.go`) re-expands the content to attribute rows to steps.
- **The agent is stateless about order.** It asks the gateway for its next task
  (`NextAgentTaskForHost`): the lowest authored step not yet `done`/`ignored`, and the
  sequence stops at a `failed` one. Ad-hoc tasks (`ad_hoc = true`) are a separate,
  always-eligible lane. So "what runs next" is entirely a question of what's in the table.
- **A step fails terminally** after `MaxAttempts` tries (`internal/gateway/server.go`
  `handleReportStatus`); the lifecycle poll (`internal/orchestrator/lifecycle.go`) then
  moves the host `building → build_failed` (or `invalid` for a failed validator with
  all steps done). Both are terminal: nothing moves a host out of them but a rebuild.
- **A dependent** of a failed host is failed too, before its steps are ever queued
  (`materializeStepsIfReady`: "dependency %q failed to configure"), with
  `steps_materialized_at` left NULL.
- **Rebuild** (`internal/api/rebuild.go`) destroys and recreates, optionally cascading
  down `depends_on`, with a dry run that shows the blast radius. **Redeploy** clears the
  host's tasks (`DeleteAgentTasksForObject`) and `steps_materialized_at`, so steps
  re-materialize on the new box.
- **In-place content updates** (`internal/api/webhook.go`, "applying commit to live
  build in place") move a live build to the new revision; each object's **fingerprint**
  (`internal/orchestrator/fingerprint.go`: image, size, disk, ports, depends_on, the
  rendered steps and script bodies) is recomputed, and a changed fingerprint means
  **recreate**. That rule is out of scope here (§10).
- **Ad-hoc targets** (`orchestrator.MatchAdHocTargets`, used by `run`, schedules and
  rebuild) already select hosts across teams by name, tag, kind, id or search.

## 2. The action

**Restart from step** is offered on hosts and containers (never networks) whose status
is `build_failed` or `invalid`, to anyone with `manage` on the repository — the level
that can rebuild, power-cycle and run commands. It's a manual action: an operator picks
it; nothing restarts a step on its own.

The operator picks an **authored step** (the groups the steps panel already shows:
"install-tool", "join-domain"…), by default the one that failed, and which instances of
the host to apply it to (§5). LaForge:

1. shows what will happen (a dry run): which steps will run again, on which hosts, that
   the boxes stay as they are, and which dependents will resume (§4);
2. on confirm, throws away the chosen step and everything after it on each host, queues
   them again **from the build's current content**, and puts each host back to
   `building`;
3. each agent, on its next poll, picks up its first re-queued step; the lifecycle poll
   finishes or fails each host as it always does.

Nothing is destroyed, no deploy task is made, the hoster isn't called. Steps before the
chosen one keep their results and output; the re-run steps' history is preserved in the
event log (`step_done`/`step_failed` events are append-only, keyed by object).

**The whole step runs again, from the top.** A `script:` step writes its file again and
executes it again; a `download:` step fetches the file again and overwrites what's
there; an `extract:` extracts again; a `compose:` pulls and brings the project up
again; a `validate:` re-checks. Nothing from the earlier attempt is trusted — the
failure may have left a half-written file or a half-finished install, and a partial
re-run would build on it. The operator chose the step because something from there on
is wrong; everything from there on is redone. Steps therefore have to be safe to run
twice (install if missing, write the whole file, idempotent commands) — they already
must be, since a rebuild runs them all again — and the confirm dialog says so.

## 3. What happens underneath

In one transaction, for the host and chosen group *G*:

1. **Refuse if a step is running right now** — an authored task `leased` with an
   unexpired lease: the agent is mid-step and would carry on alongside the restart.
   The dialog says so, with *wait* or *force* (which ignores it; the late result for the
   old task id is dropped by the gateway, below).
2. **Delete** the host's authored (`ad_hoc = false`) `agent_task` rows with
   `step_group >= G` — their `validator_result` rows cascade. Ad-hoc tasks are
   untouched; a `run` the operator did while debugging stays in the history.
3. **Re-expand** the host's steps from the build's **current** content revision with
   `gateway.ExpandSteps` (the same call `materializeSteps` makes, with the same registry
   auth and public address), keep the commands with `Group >= G`, and insert them with
   `CreateAgentTask` at `NextStepIndexForHost` onward, `step_group` set. They land after
   every existing row, which is fine: all earlier authored steps are `done`/`ignored`,
   so the lease order is unchanged, and new ids mean a late report for a deleted task
   simply finds nothing.
4. **Reopen the host**: `build_failed`/`invalid → building`, `last_error` cleared (a new
   guarded setter; today nothing leaves those states). `steps_materialized_at` stays
   set — the steps *are* materialized.
5. **Event**: `steps.restarted` on the build — who, which host, from which step, how
   many steps, and whether the content revision differs from the one the host was
   built from.

The gateway needs one defensive change: `handleReportStatus` for a task id that no
longer exists logs and answers `ok: false` instead of erroring (the agent moves on). The
lifecycle poll needs none: a `building` host with open steps is its normal case.

Why delete-and-recreate rather than reset in place: it picks up content fixes (§5), it
makes a stale result from the agent harmless (new ids), and it reuses `materializeSteps`
instead of adding a second expansion path.

## 4. Dependents

A host that `depends_on` the restarted one may have been failed with *"dependency X
failed to configure"* before any of its steps ran. The dry run lists these, and the
action offers **resume dependents** (default on): each goes `build_failed → running`
with `last_error` cleared, `steps_materialized_at` still NULL. Nothing else is needed:
`materializeStepsIfReady` queues their steps the moment the restarted host finishes,
exactly as on a first deploy — and fails them again, with the same message, if it
doesn't. Dependents that had already finished are left alone: their configuration ran
against the dependency's earlier, working state, and restarting them is the operator's
call (they can restart those from their own step 0).

Only direct and transitive dependents in that specific state are touched; the graph
comes from the build's content, as in rebuild.

## 5. Many hosts at once

A failed step usually fails the same way on every team's copy of the host — the mirror
was down for all of them, the script was wrong for all of them. So the action is picked
once for a **host definition** and applied to its instances:

- **Who's eligible.** Every deployed object in the build with the same `object_name`
  (the same host or container in the content, hence the same authored steps), in a
  restartable state. Instances of *other* hosts aren't offered, even if they happen to
  run the same script — their step numbering differs and "step 3" means something else
  there; restart those from their own panel.
- **Picking.** The dialog opened from one host lists every team's instance of it with its
  state and failed step, all restartable ones ticked; untick to leave some alone. The
  hosts page's bulk selection (the same team/tag/search selectors `run` and `rebuild`
  use) can start it too, as long as the selection is one host definition.
- **One step, per host.** The chosen step is a group index into that definition's steps
  — the same on every instance — and each host is handled independently (§3): a host
  with a running step is skipped and named unless `force`, a host whose agent is gone
  just queues. The result lists what happened to each.
- **Dependents** (§4) are resolved per host: team 3's web01 resumes team 3's dependents.

## 6. API, UI, CLI

**API** — one endpoint, modelled on rebuild and `run`:

```
POST /builds/{id}/restart-steps
{ "target": { "ids": [...] } | { "name": "web01", "teams": [1,3] },
  "from_group": 3, "resume_dependents": true, "force": false, "dry_run": false }
```

- The target resolves through `MatchAdHocTargets`; every matched object must share one
  `object_name`, or `400` naming the odd ones out.
- `dry_run` returns, per host: its state, the steps that will re-run (group index,
  label, commands), whether a step is running now, and the dependents that will resume.
- Otherwise it performs §3 and §4 on each eligible host and returns, per host, the new
  tasks and resumed dependents, or why it was skipped.
- `409` when no target is in a restartable state or `from_group` names no step of that
  host; a host with a running step is skipped (not an error) unless `force`. Manage
  level; the build must be deploying/building/finished/failed, as for rebuild.

**UI** — on a host's steps panel (`ObjectLogPanel`), each authored step group gets a
**Restart from here** control when the host is in a restartable state; the failed group
also gets it inline next to its red badge. The dialog is the dry run: the step list from
there on, then every team's instance of this host with its state and a tick box (all
restartable ones ticked), the dependents that will resume, a note if the content has
changed since these hosts were built, and the "steps run again in full; they should be
safe to run twice" line. On the hosts page, a `build_failed`/`invalid` row's hover
controls gain *Restart steps…* beside *Rebuild*, and the bulk-action bar offers it when
the selection is one host definition.

**CLI** — `laforge build restart-steps <build> --host web01 --from install-tool
[--team 3,4] [--no-dependents] [--force] [--dry-run]`, resolving the step by label or
index, printing the dry run and asking before acting (as `laforge run` does).

## 7. Database changes

One migration: `agent_task.step_group integer` — which authored step a row came from
(NULL for ad-hoc); set by `materializeSteps` from `PlannedCommand.Group`. The steps API
reads it instead of re-expanding content to attribute rows, which also fixes attribution
after a content change (today's re-expansion attributes old rows against new content).

New queries: `DeleteAuthoredAgentTasksFromGroup(object, group)`,
`SetDeployedObjectRestarting(object)` (`build_failed`/`invalid → building`, clear
`last_error`), `ResumeDependentForRestart(object)` (`build_failed → running` where
`steps_materialized_at IS NULL`).

## 8. Edge cases and rules

| Case | Rule |
| --- | --- |
| The agent isn't checking in (host off, agent dead) | The restart is accepted; the steps wait as pending. The dialog notes the agent's last check-in. Power the host on or rebuild it. |
| A step is `leased` and running | Refused without `force`; with it, the old task is deleted and its late result is dropped. |
| `from_group` is after the last step | `409`: nothing to run again. |
| `from_group` is 0 | Every step re-runs on the existing box — the "rebuild without recreating" case. |
| The content revision changed and now has fewer steps than *G* | `409` with the current step list; pick again. |
| Ad-hoc tasks exist | Untouched, and still take priority in the lease order, as today. |
| An `ignore_errors` step that was `ignored` is after *G* | Re-runs like any other; may be ignored again. |
| The host is `destroying`/`destroyed`/`pending`/`deploying` | Not offered: there's no box, or it's not up yet. |
| The build is `tearing_down`/`torn_down`/`purged` | `409`, as for rebuild. |
| Two operators restart the same host at once | The second finds the host `building` and gets `409` (the state check is in the transaction). |
| Validators (`validate:` steps) after *G* | Re-run; their old `validator_result` rows went with the deleted tasks, so the build's findings reflect the new run. |
| A restart and the fingerprint | A restart doesn't touch the host's fingerprint; it re-expands steps from the build's current revision, which is the one the fingerprint was computed from. |
| Several hosts, one with a running step | That host is skipped and named in the result; the others proceed. `force` includes it. |

## 9. Phases

| Phase | Size | What | Done when |
| --- | --- | --- | --- |
| **R1 The action** | M | `step_group` on tasks (and the steps API reading it), the endpoint with targets and dry run, the transaction in §3 per host, dependents in §4, the gateway's unknown-task tolerance, the steps-panel dialog with the instance list, the hosts-page controls, the CLI, `steps.restarted` events | a fake-builder build with three teams: web01 fails at step 2 of 4 on all three; restarting from step 2 for all of them keeps steps 0–1's output, runs 2–3 again with new task ids (a `download:` among them fetches again), and each goes `building → finished`; a dependent failed on team 2's web01 resumes and finishes; the instance with a leased task is skipped without `force`; a late report for a deleted task is dropped |

## 10. Open decisions

- **Finished hosts.** Should *Restart from step* be offered on a `finished` host (re-run
  a validator, redo a step by hand)? Harmless to allow under `manage`; left out of the
  first version to keep its surface small.
- **Content fixes and the fingerprint rule.** A pushed fix to a script changes the
  fingerprint of every host that runs it, and the in-place update recreates them before
  anyone can restart a step. So this action helps with failures the content didn't
  cause (a mirror down, a flaky install) and with fixes that don't reach the
  fingerprint; a fix that does still means a rebuild. Changing that rule — treating
  step-only changes as "restart from the first changed step" rather than "recreate" —
  is a separate decision, deliberately not taken here.
- **Hosts that share steps through `extends`.** Two host definitions with the same
  base have the same early steps but may add their own; the action keeps to one
  definition. If restarting a shared base step across definitions turns out to matter,
  the match can become "same expanded steps up to and including the chosen one".
- **Where a restarted step's old output goes.** The deleted rows' `output` is gone from
  the steps panel (the events keep `step_done`/`step_failed` with the output's tail).
  Keeping the last run's rows as a collapsed "previous attempt" would need a `run` column
  on `agent_task`; not planned unless operators miss it.
- **Auto-apply granularity.** The switch is per configured build. Per-host or per-step
  (`restart_on_change: false` on a step whose re-run is expensive) can come later if a
  real content repository needs it.
