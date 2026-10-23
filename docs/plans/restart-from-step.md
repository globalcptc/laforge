# Plan: restart a host's build from a step

**Status:** proposal for `3.0`. **Written:** 2026-10-23, against `3.0` at `87d2e5ba`.

A host whose build stopped — a step failed, a validator didn't pass — can today only be
**rebuilt**: destroyed at the hoster and deployed again from nothing, every step re-run.
That's the right tool when the box is wrong; it's the wrong tool when step 7 of 12 failed
because a package mirror timed out, or because the script had a typo that's now fixed.
This adds **Restart from step**: pick an authored step, and the host runs that step and
everything after it again, on the box it already has.

## Contents

1. [What exists](#1-what-exists)
2. [The action](#2-the-action)
3. [What happens underneath](#3-what-happens-underneath)
4. [Dependents](#4-dependents)
5. [Content fixes: restart instead of recreate](#5-content-fixes-restart-instead-of-recreate)
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
  **recreate**. A one-line script fix therefore rebuilds every host that runs the script.

## 2. The action

**Restart from step** is offered on a host or container (never a network) whose status
is `build_failed` or `invalid`, to anyone with `manage` on the repository — the level
that can rebuild, power-cycle and run commands.

The operator picks an **authored step** (the groups the steps panel already shows:
"install-tool", "join-domain"…), by default the one that failed. LaForge:

1. shows what will happen (a dry run): which steps will run again, that the host stays
   as it is, and which dependents will resume (§4);
2. on confirm, throws away the chosen step and everything after it, queues them again
   **from the build's current content**, and puts the host back to `building`;
3. the agent, on its next poll, picks up the first re-queued step; the lifecycle poll
   finishes or fails the host as it always does.

Nothing is destroyed, no deploy task is made, the hoster isn't called. Steps before the
chosen one keep their results and output; the re-run steps' history is preserved in the
event log (`step_done`/`step_failed` events are append-only, keyed by object).

The step re-runs **as written**: a `script:` step writes the file again and executes it
again. Steps already have to tolerate that — a rebuild re-runs them all on a fresh box —
but a restart runs them on a box where they may have half-happened, so the content
rules get one sentence: *steps should be safe to run twice* (install if missing, write
the whole file, idempotent commands). The confirm dialog says this.

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

## 5. Content fixes: restart instead of recreate

The action alone covers transient failures. The common case — *the script was wrong,
I fixed it* — runs into the fingerprint rule: pushing the fix changes the host's
fingerprint, and the in-place update **recreates** the host before anyone can restart
anything. So the second half of this plan is a change to that rule:

- **Split the fingerprint.** `Fingerprint` keeps hashing everything, but records two
  parts on the object: an **infrastructure** fingerprint (os, image, size, disk, compose
  archive, ports, depends_on — what the hoster built) and a **steps** fingerprint (the
  rendered steps and script bodies). `deployed_object` gains `steps_fingerprint`.
- **Infrastructure change → recreate**, as today. **Steps-only change → mark, don't
  recreate**: the host gets `steps_outdated_from = G`, where *G* is the first authored
  step whose expanded commands differ between the revision it was built from and the
  new one (a straight comparison of the two `ExpandSteps` outputs, group by group). Its
  row shows *steps outdated* on the hosts page with the step named.
- **Then either**: the operator restarts from *G* (the button is pre-filled), or, with
  the build's **auto-apply step changes** switch on (off by default; a per-configured-
  build setting beside auto-deploy), the orchestrator does it on the reconcile pass that
  found the difference — for hosts that are `finished`, `build_failed` or `invalid`, never
  one that's mid-build, and never a host whose infrastructure also changed.
- A **finished** host restarted this way goes `finished → building` and finishes again;
  its dependents aren't touched (their dependency is still configured; it's being
  reconfigured, which is the operator's judgement — the event log says it happened).

This is what makes a competition fix cheap: push, watch the hosts that run that script
re-run from the changed step, no boxes recreated, no agents re-enrolled. The old
behaviour is one switch away — recreating remains the answer whenever the author wants
a clean box (rebuild) or infrastructure changed.

## 6. API, UI, CLI

**API** — one endpoint, modelled on rebuild:

```
POST /builds/{id}/objects/{objectId}/restart-steps
{ "from_group": 3, "resume_dependents": true, "force": false, "dry_run": false }
```

- `dry_run` returns: the steps that will re-run (group index, label, commands), whether
  the content revision differs from the host's, the dependents that will resume, and
  whether a step is running now.
- Otherwise it performs §3 and §4 and returns the new tasks and the resumed dependents.
- `409` when the host isn't `build_failed`/`invalid` (or, for §5, `finished` with
  `steps_outdated_from` set), when `from_group` names no step, or when a step is running
  and `force` is false. Manage level; the build must be deploying/building/finished/failed,
  as for rebuild.

**UI** — on the host's steps panel (`ObjectLogPanel`), each authored step group gets a
**Restart from here** control when the host is in a restartable state; the failed group
also gets it inline next to its red badge. The dialog is the dry run: "Steps 7–12 will
run again on *web01* as it is. The content has changed since this host was built
(install-tool.sh). 2 hosts that depend on web01 will resume once it finishes. Steps
should be safe to run twice." On the hosts page, a `build_failed`/`invalid` row's
hover controls gain *Restart steps…* beside *Rebuild*, and a *steps outdated from
"install-tool"* note appears when §5 marks one. The configured-build page gets the
**auto-apply step changes** switch.

**CLI** — `laforge build restart-steps <build> --host web01 --from install-tool
[--team 3] [--no-dependents] [--force] [--dry-run]`, resolving the step by label or
index, printing the dry run and asking before acting (as `laforge run` does).

## 7. Database changes

One migration:

- `agent_task.step_group integer` — which authored step a row came from (NULL for
  ad-hoc); set by `materializeSteps` from `PlannedCommand.Group`. The steps API reads it
  instead of re-expanding content to attribute rows, which also fixes attribution after a
  content change (today's re-expansion attributes old rows against new content).
- `deployed_object.steps_fingerprint text`, `deployed_object.steps_outdated_from
  integer` (§5).
- `configured_build.auto_apply_steps boolean NOT NULL DEFAULT false` (§5).

New queries: `DeleteAuthoredAgentTasksFromGroup(object, group)`,
`SetDeployedObjectRestarting(object)` (`build_failed`/`invalid`/`finished → building`,
clear `last_error`), `ResumeDependentForRestart(object)` (`build_failed → running` where
`steps_materialized_at IS NULL`), `SetDeployedObjectStepsOutdated(object, group|NULL)`.

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
| Fingerprint rule and a restart | A manual restart doesn't touch fingerprints; after §5, restarting from the outdated step clears `steps_outdated_from` and sets `steps_fingerprint` to the new revision's. |

## 9. Phases

| Phase | Size | What | Done when |
| --- | --- | --- | --- |
| **R1 The action** | M | `step_group` on tasks (and the steps API reading it), the endpoint with dry run, the transaction in §3, dependents in §4, the gateway's unknown-task tolerance, the steps-panel and hosts-page UI, the CLI, `steps.restarted` events | a fake-builder host whose step 2 of 4 failed is restarted from step 2: steps 0–1 keep their output, 2–3 run again with new task ids, the host goes `building → finished`; a dependent failed on it resumes and finishes; a restart while a task is leased is refused without `force`; a late report for the deleted task is dropped |
| **R2 Content fixes** | M | the split fingerprint, `steps_outdated_from` with the first-changed-group diff, the hosts-page note, pre-filled restart, the `auto_apply_steps` switch and the orchestrator's automatic restart | pushing a script change to a live build marks its hosts *steps outdated from* that step and recreates none; with the switch on they re-run from it and finish; changing a host's image still recreates it |

R1 stands alone and is worth shipping first.

## 10. Open decisions

- **Finished hosts without a content change.** Should *Restart from step* be offered on
  a `finished` host at all (re-run a validator, redo a step by hand)? Harmless to allow
  under `manage`; the plan allows it only via §5's outdated marker to keep the first
  version's surface small.
- **Where a restarted step's old output goes.** The deleted rows' `output` is gone from
  the steps panel (the events keep `step_done`/`step_failed` with the output's tail).
  Keeping the last run's rows as a collapsed "previous attempt" would need a `run` column
  on `agent_task`; not planned unless operators miss it.
- **Auto-apply granularity.** The switch is per configured build. Per-host or per-step
  (`restart_on_change: false` on a step whose re-run is expensive) can come later if a
  real content repository needs it.
