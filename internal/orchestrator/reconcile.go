// Package orchestrator is "the orchestrator diffs desired vs observed
// state in Postgres and creates tasks". It never talks to a hypervisor or a builder itself -- it only
// ever reads content from a checkout, reads/writes Postgres, and creates
// task rows for internal/runner to pick up. Orchestrators are stateless
// replicas: Reconcile is safe to call concurrently, repeatedly, and from
// more than one process, because every write it makes is either an
// idempotent "ensure" (EnsureTeam, EnsureDeployedObject) or gated by
// task's partial unique index (CreateTaskIfNoneOpen).
package orchestrator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/globalcptc/laforge/internal/db"
	"github.com/globalcptc/laforge/internal/loader"
	"github.com/globalcptc/laforge/internal/render"
	"github.com/globalcptc/laforge/internal/schedule"
)

// Reconcile expands buildID's desired state -- its environment's teams x
// topology, resolved from repoRoot's content at whatever commit it's
// checked out to -- and diffs it against deployed_object rows, creating
// deploy/destroy tasks for anything that doesn't match yet. One pass does
// not necessarily finish a build: an object whose fingerprint changed
// needs a destroy this pass and a deploy next pass, once the destroy has
// actually completed ("rebuild means recreate"). Call it in a loop (see
// cmd/laforge-orchestrator) until nothing changes.
// onPhase, when non-nil, is called with the duration of each major internal
// phase (content parse, topology resolve + render) so the caller can surface it
// -- the orchestrator passes a callback that writes build.phase timeline events
// while a build is still preparing, so the startup gap shows where the time
// goes. nil disables it (every test, and steady-state passes).
func Reconcile(ctx context.Context, pool *pgxpool.Pool, repoRoot string, buildID pgtype.UUID, phase ...func(label string, d time.Duration)) error {
	// Optional, so the many callers that don't time phases (tests) pass nothing.
	var onPhase func(label string, d time.Duration)
	if len(phase) > 0 {
		onPhase = phase[0]
	}
	q := db.New(pool)

	build, err := q.GetBuild(ctx, buildID)
	if err != nil {
		return fmt.Errorf("loading build: %w", err)
	}

	loadStart := time.Now()
	c, err := loader.LoadCached(repoRoot)
	if err != nil {
		return fmt.Errorf("loading content: %w", err)
	}
	if onPhase != nil {
		onPhase("content parsed", time.Since(loadStart))
	}
	resolveStart := time.Now()
	if len(c.Errors) > 0 {
		return fmt.Errorf("content at %s is not valid, refusing to reconcile: %d schema/reference error(s) (first: %s)",
			repoRoot, len(c.Errors), c.Errors[0].Message)
	}

	env := findEnvironment(c, build.EnvironmentName)
	if env == nil {
		return fmt.Errorf("environment %q not found in content at %s", build.EnvironmentName, repoRoot)
	}

	containerNames := make(map[string]bool, len(c.Containers))
	for _, ct := range c.Containers {
		containerNames[ct.Name] = true
	}

	if err := checkBuilderCompatibility(ctx, q, pool, build, c, env, containerNames); err != nil {
		return fmt.Errorf("build/builder compatibility: %w", err)
	}

	// "Build and deploy are separate verbs... Build: resolve the
	// environment... touches no hoster... Deploy: apply that build" --
	// deployTasks is that gate, made real. Every EnsureTeam/
	// EnsureDeployedObject call below still runs regardless of status:
	// a `planned` build's whole point is to be "fully inspectable" --
	// real teams, real resolved topology, real fingerprints, all
	// computed and stored -- while genuinely touching no hoster. Only
	// the actual task-creation calls (reconcileObject, requestDestroy)
	// check this. Found missing entirely: nothing
	// before this ever consulted build.Status here, so a `planned`
	// build reconciled by cmd/laforge-orchestrator's own poll loop
	// (which already lists both "planned" and "deploying") would have
	// silently created real deploy tasks -- exactly the failure "A
	// build that looks wrong is discarded rather than torn down"
	// exists to prevent, since a "planned" build was never actually
	// safe to look at and discard.
	deployTasks := build.Status == "deploying"
	// Hold hosts on step-only changes (docs/plans/restart-from-step.md §6): a
	// per-configured-build switch read once per pass.
	holdOnStepChange := false
	if build.ConfiguredBuildID.Valid {
		if cb, err := q.GetConfiguredBuild(ctx, build.ConfiguredBuildID); err == nil {
			holdOnStepChange = cb.HoldOnStepChange
		}
	}
	// Step materialization must continue through the "building" phase, not just
	// "deploying": the build leaves "deploying" as soon as all infra is up and
	// the first agent starts working, but dependents' steps are queued only as
	// their dependencies FINISH, which happens during "building". A planned
	// build still materializes nothing (it touches neither hoster nor agent).
	materializeActive := build.Status == "deploying" || build.Status == "building"
	anchors := anchorsFromEnvironment(env)
	// Environment-wide People + DNS are identical for every object in every team,
	// so compute them ONCE per pass and share them across all reconcileCopy calls
	// instead of rebuilding them inside each per-object render.Resolve (which was
	// O(objects^2) work -- a dominant cost of a reconcile pass).
	envShared := render.NewEnvShared(c, env)
	// Object + network names placed in this environment's topology -- the set
	// depends_on ordering can actually wait on (a dependency not placed here
	// can't be ordered on, so it's ignored rather than wedging the build).
	placed := placedNames(env.Networks)
	// Every copy this environment places, ordered roots-first by depends_on
	// depth: a dependency's box (and its configuration) gets underway before
	// the dependents that wait on it. The topology is identical for every
	// team, so this is computed once and reused across the team loop. See
	// "prioritize image deployments based on the tree" -- deploy tasks are no
	// longer gated on depends_on (the box deploys ahead of time); ordering
	// their creation is what makes the tree-priority real, since LeaseTask
	// draws pending tasks oldest-first.
	//
	// Creation order alone can't hold that priority once every task is leased at
	// once (an unbounded pool), so each deploy task also carries its critical-path
	// priority (DependentCounts: how many objects wait on it) in its payload. The
	// lease queries serve higher priorities first, across every team, and the
	// Incus builder's per-host admission queue does the same.
	placements := orderedPlacements(env, containerNames, c.DependencyDepth(), c.DependentCounts())
	usesWaitFor := contentUsesWaitFor(c)

	for teamNum := 1; teamNum <= env.Teams; teamNum++ {
		team, err := q.EnsureTeam(ctx, db.EnsureTeamParams{BuildID: buildID, TeamNumber: int32(teamNum)})
		if err != nil {
			return fmt.Errorf("team %d: %w", teamNum, err)
		}

		// This team's dependency readiness as of the start of this pass, used
		// to gate STEP execution (not the box deploy): finishedDeps[name] is
		// true only when every copy of that object has fully FINISHED
		// configuring; failedDeps[name] is true when any copy failed terminally
		// (so a dependent can be failed rather than waiting on it forever).
		finishedDeps, failedDeps, networkUp, networkFailed, err := teamDepReadiness(ctx, q, team.ID)
		if err != nil {
			return fmt.Errorf("team %d readiness: %w", teamNum, err)
		}
		// Step-level wait_for readiness (milestones/steps reached per copy),
		// read only when some step actually waits.
		var gates *teamGates
		if usesWaitFor && materializeActive {
			if gates, err = loadTeamGates(ctx, q, team.ID, placed, finishedDeps, failedDeps); err != nil {
				return fmt.Errorf("team %d step gates: %w", teamNum, err)
			}
		}

		desired := make(map[string]bool)

		for networkName := range env.Networks {
			desired["network/"+networkName] = true
			if err := reconcileNetwork(ctx, q, c, team.ID, buildID, networkName, deployTasks); err != nil {
				return fmt.Errorf("team %d, network %q: %w", teamNum, networkName, err)
			}
		}

		for _, pl := range placements {
			desired[pl.kind+"/"+pl.as] = true
			if err := reconcileCopy(ctx, q, repoRoot, c, envShared, env.Name, team.ID, buildID, build.ContentRevisionID, teamNum, pl.kind, pl.objectName, pl.as, pl.networkName, pl.priority, deployTasks, materializeActive, holdOnStepChange, anchors, placed, finishedDeps, failedDeps, gates, networkUp, networkFailed); err != nil {
				return fmt.Errorf("team %d, %s %q: %w", teamNum, pl.kind, pl.as, err)
			}
		}

		// DNS is not a builder responsibility (LaForge doesn't run DNS): the
		// resolved records are exposed in the template data for a script on the
		// DC/Bind host to consume (see render.Context.TemplateData). No "dns"
		// deployed_object, no builder DNS call.

		if err := reconcileRemovals(ctx, q, buildID, team.ID, desired, deployTasks); err != nil {
			return fmt.Errorf("team %d: %w", teamNum, err)
		}
	}

	if onPhase != nil {
		onPhase("topology resolved & rendered", time.Since(resolveStart))
	}

	// Announce the resolve on the build's own timeline, once per content
	// revision (the first resolve, and again on a rebuild to a new commit).
	// This fills the otherwise-silent gap between build.triggered and the first
	// deploy task -- where the orchestrator is fetching the checkout and
	// resolving + rendering the whole topology with nothing to show for it --
	// so a freshly triggered build no longer looks like it's doing nothing.
	// Gated via resolved_revision_id so it never repeats on the 2s cadence.
	if !build.ResolvedRevisionID.Valid || build.ResolvedRevisionID.Bytes != build.ContentRevisionID.Bytes {
		hosts, containers, scripts := 0, 0, 0
		for _, pl := range placements {
			if pl.kind == "container" {
				containers++
			} else {
				hosts++
			}
			scripts += countScriptSteps(c, pl.kind, pl.objectName)
		}
		hosts *= env.Teams
		containers *= env.Teams
		scripts *= env.Teams
		networks := len(env.Networks) * env.Teams
		payload, _ := json.Marshal(map[string]any{
			"teams": env.Teams, "hosts": hosts, "containers": containers,
			"networks": networks, "scripts_rendered": scripts,
			"content_revision_id": build.ContentRevisionID.String(),
		})
		q.CreateEvent(ctx, db.CreateEventParams{
			BuildID: buildID, Kind: "build.resolved",
			Message: fmt.Sprintf("resolved content: %d team(s), %d host(s), %d container(s), %d network(s), %d step script(s) rendered",
				env.Teams, hosts, containers, networks, scripts),
			Payload: payload,
		})
		if err := q.SetBuildResolvedRevision(ctx, db.SetBuildResolvedRevisionParams{ID: buildID, ResolvedRevisionID: build.ContentRevisionID}); err != nil {
			return fmt.Errorf("recording resolved revision: %w", err)
		}
	}

	return nil
}

// countScriptSteps is how many of an object's authored steps are `script`
// steps -- the ones Fingerprint actually renders (see fingerprint.go). Used
// only for the build.resolved event's "N step scripts rendered" summary, so an
// unresolvable object (caught elsewhere) just contributes zero.
func countScriptSteps(c *loader.Content, kind, objectName string) int {
	var steps []loader.Step
	if kind == "container" {
		if ct := findContainer(c, objectName); ct != nil {
			steps = ct.Steps
		}
	} else if h := findHost(c, objectName); h != nil {
		steps = h.Steps
	}
	n := 0
	for _, s := range steps {
		if s.ActionKey() == "script" {
			n++
		}
	}
	return n
}

// infraUp reports whether an object's status means its instance actually
// exists at the hoster -- so reconcile treats it as already provisioned
// (only a fingerprint change triggers a recreate) rather than requesting a
// fresh deploy. All the states past a successful create count: running (up,
// agent not in yet), building (agent working), finished (done), and the two
// post-create failures (build_failed / invalid) whose instance is still up
// and should not be silently torn down and recreated.
func infraUp(status string) bool {
	switch status {
	case "running", "building", "finished", "build_failed", "invalid":
		return true
	}
	return false
}

// AdvanceBuildStatus moves a build through the deploy -> build -> finished
// phases from its objects' own states, ahead of (and independent of)
// content resolution -- so a build whose objects have all settled advances
// even when its recorded revision can no longer be fetched (the branch has
// moved past it). The poll loop calls this before it resolves content, for
// exactly that reason. Returns whether the build reached a TERMINAL state
// (finished/failed) this call, in which case there's nothing left to
// reconcile this pass; a move into the non-terminal "building" phase
// returns false so live reconcile keeps running. A no-op for any build not
// currently deploying/building.
func AdvanceBuildStatus(ctx context.Context, pool *pgxpool.Pool, buildID pgtype.UUID) (bool, error) {
	q := db.New(pool)
	build, err := q.GetBuild(ctx, buildID)
	if err != nil {
		return false, fmt.Errorf("loading build: %w", err)
	}
	if build.Status != "deploying" && build.Status != "building" {
		return false, nil
	}
	return finalizeBuildStatus(ctx, q, buildID, build.Status)
}

// finalizeBuildStatus computes the build's phase from its objects. The infra
// phase ("deploying") ends once nothing is still pending/mid-deploy; then, if
// any object's agent is still running/building, the build is "building"
// (infra up, hosts building -- deliberately NOT "finished," and never green,
// until every step and validator has actually completed); once every object
// is terminal, the build is "finished" if all succeeded or "failed" if any
// failed at any layer (deploy, build step, or validator). Destroyed/
// destroying objects are teardown state and don't count. Reads only object
// statuses -- no content or builder -- so it runs before content resolution
// and works even for a build whose revision can no longer be fetched.
// Idempotent. Returns whether it reached a terminal state.
func finalizeBuildStatus(ctx context.Context, q *db.Queries, buildID pgtype.UUID, current string) (bool, error) {
	objs, err := q.ListDeployedObjectsByBuild(ctx, buildID)
	if err != nil {
		return false, fmt.Errorf("listing deployed objects: %w", err)
	}
	infraPending, agentWorking, live, failed := 0, 0, 0, 0
	for _, o := range objs {
		switch o.Status {
		case "pending", "deploying", "destroy_queued", "destroying":
			// "destroying" counts as infra-in-progress too: during a rebuild the
			// runner briefly marks the object destroying before resetting it to
			// pending, and the build must not be judged finished/failed in that
			// window -- otherwise a concurrent pass advances it past the deploy
			// phase and the pending replacement is never redeployed.
			infraPending++
		case "running", "building":
			agentWorking++
			live++
		case "finished":
			live++
		case "deploy_failed", "build_failed", "invalid":
			failed++
			live++
		}
	}
	// Infra still being created/destroyed, or nothing actually stood up (an
	// all-teardown or empty build) -- not a deploy to advance yet.
	if infraPending > 0 || live == 0 {
		// Recover an active build that left the deploy phase but now has infra
		// pending again (a rebuild reset an object to pending while the build was
		// already "building"). Reconcile only queues deploy tasks in "deploying",
		// so without this the pending replacement strands indefinitely.
		if infraPending > 0 && current == "building" {
			if _, err := q.SetBuildStatus(ctx, db.SetBuildStatusParams{ID: buildID, Status: "deploying"}); err != nil {
				return false, fmt.Errorf("re-entering deploy phase for pending infra: %w", err)
			}
			decide(ctx, q, buildID, pgtype.UUID{}, "build.status", fmt.Sprintf("build is deploying again: %d object(s) have infrastructure pending (a rebuild or redeploy)", infraPending), nil)
		}
		return false, nil
	}

	var status string
	terminal := false
	switch {
	case agentWorking > 0:
		status = "building" // infra up, hosts still building -- not finished
	case failed > 0:
		status = "failed"
		terminal = true
	default:
		status = "finished"
		terminal = true
	}
	if status != current {
		if _, err := q.SetBuildStatus(ctx, db.SetBuildStatusParams{ID: buildID, Status: status}); err != nil {
			return false, fmt.Errorf("marking build %s: %w", status, err)
		}
		decide(ctx, q, buildID, pgtype.UUID{}, "build.status", fmt.Sprintf("build is now %s (was %s): %d object(s) live, %d still building, %d failed", status, current, live, agentWorking, failed),
			map[string]any{"from": current, "to": status, "live": live, "building": agentWorking, "failed": failed})
		// A failed build is terminal and will never run its schedule:
		// rows, so cancel them -- otherwise they stay pending and overdue
		// and the dispatch loop keeps resolving content + minting a GitHub
		// token every pass. Deliberately NOT done for "finished": a
		// finished build is the competition running, when schedule: and
		// access: windows are exactly what should keep firing.
		if status == "failed" {
			if _, err := q.CancelScheduledTasksForBuild(ctx, buildID); err != nil {
				return false, fmt.Errorf("canceling scheduled tasks for failed build: %w", err)
			}
		}
	}
	return terminal, nil
}

// reconcileRemovals destroys anything still recorded for this team that's
// no longer in the environment's desired topology at all -- a network,
// host, or container that was deleted from the environment file, not one
// that merely changed. Those get `remove: true` in the destroy task's
// payload, so the runner deletes the row afterward instead of resetting
// it for redeploy.
func reconcileRemovals(ctx context.Context, q *db.Queries, buildID, teamID pgtype.UUID, desired map[string]bool, deployTasks bool) error {
	existing, err := q.ListDeployedObjectsByTeam(ctx, teamID)
	if err != nil {
		return fmt.Errorf("listing deployed objects: %w", err)
	}
	for _, obj := range existing {
		identity := db.StrOrEmpty(obj.AsName)
		if identity == "" {
			identity = obj.ObjectName
		}
		if desired[obj.Kind+"/"+identity] {
			continue
		}
		if obj.Status == "destroyed" {
			continue
		}
		requested, err := requestDestroy(ctx, q, buildID, obj, true, deployTasks)
		if err != nil {
			return fmt.Errorf("removing %s %q: %w", obj.Kind, identity, err)
		}
		if requested {
			decide(ctx, q, buildID, obj.ID, "reconcile.remove", fmt.Sprintf("%s: no longer in the content; destroying and removing it", objectLabel(obj, 0)), nil)
		}
	}
	return nil
}

func reconcileNetwork(ctx context.Context, q *db.Queries, c *loader.Content, teamID, buildID pgtype.UUID, networkName string, deployTasks bool) error {
	net := findNetwork(c, networkName)
	if net == nil {
		return fmt.Errorf("network %q not found in content", networkName)
	}
	fp := networkFingerprint(net)

	obj, err := q.EnsureDeployedObject(ctx, db.EnsureDeployedObjectParams{
		TeamID: teamID, Kind: "network", ObjectName: networkName,
	})
	if err != nil {
		return err
	}
	// The runner has exhausted retries or reported a permanent network error.
	// Keep that failure terminal until the object is explicitly reset;
	// creating a fresh task here would silently reset the retry budget.
	if obj.Status == "deploy_failed" {
		return nil
	}
	_, err = reconcileObject(ctx, q, buildID, obj, fp, "", false, networkDeployPriority, deployTasks)
	return err
}

func reconcileCopy(ctx context.Context, q *db.Queries, repoRoot string, c *loader.Content, envShared *render.EnvShared, envName string, teamID, buildID, contentRevID pgtype.UUID, teamNum int, kind, objectName, asName, networkName string, priority int, deployTasks, materializeActive, holdOnStepChange bool, anchors schedule.Anchors, placed, finishedDeps, failedDeps map[string]bool, gates *teamGates, networkUp, networkFailed map[string]bool) error {
	rctx, err := render.ResolveWith(c, envName, asName, teamNum, envShared)
	if err != nil {
		return fmt.Errorf("resolving: %w", err)
	}

	var disk int
	var ports loader.Ports
	var dependsOn []string
	var ownTags map[string]string
	var steps, sched []loader.Step
	if kind == "host" {
		h := findHost(c, objectName)
		if h == nil {
			return fmt.Errorf("host %q not found in content", objectName)
		}
		disk, ports, dependsOn, ownTags, steps, sched = h.Disk, h.Ports, h.DependsOn, h.Tags, h.Steps, h.Schedule
	} else {
		ct := findContainer(c, objectName)
		if ct == nil {
			return fmt.Errorf("container %q not found in content", objectName)
		}
		disk, ports, dependsOn, ownTags, steps, sched = ct.Disk, ct.Ports, ct.DependsOn, ct.Tags, ct.Steps, ct.Schedule
	}
	// Tags carried onto the runtime row cascade from least to most specific
	// (most specific wins): environment -> network -> the scripts this object
	// runs -> the object's own tags.
	tags := effectiveTags(c, envName, networkName, objectName, steps, sched, ownTags)

	// The fingerprint re-renders every step script, which is invariant for a
	// given (content revision, object): cache it so a settled build doesn't
	// re-render its whole content on every 2s pass. render.Resolve above still
	// runs each pass (rctx is needed below), but it's far cheaper than the
	// RenderScript loop Fingerprint drives. A new content revision (a rebuild)
	// gets a fresh key, so a content change is never masked. See fingerprintCache.
	fpKey := fmt.Sprintf("%s\x00%s\x00%s\x00%d", contentRevID.String(), kind, asName, teamNum)
	fp, cached := fpCache.get(fpKey)
	if !cached {
		fp, err = Fingerprint(repoRoot, c, rctx, disk, ports, dependsOn)
		if err != nil {
			return fmt.Errorf("fingerprinting: %w", err)
		}
		fpCache.put(fpKey, fp)
	}
	// The infrastructure half on its own (what the hoster built), recorded
	// with the deploy so a later change can be told apart as steps-only.
	infraFP, cached := fpCache.get(fpKey + "\x00infra")
	if !cached {
		infraFP, err = InfraFingerprint(repoRoot, rctx, disk, ports, dependsOn)
		if err != nil {
			return fmt.Errorf("fingerprinting infrastructure: %w", err)
		}
		fpCache.put(fpKey+"\x00infra", infraFP)
	}

	obj, err := q.EnsureDeployedObject(ctx, db.EnsureDeployedObjectParams{
		TeamID: teamID, Kind: kind, ObjectName: objectName,
		AsName: db.StrPtr(asName), NetworkName: db.StrPtr(networkName),
	})
	if err != nil {
		return err
	}
	// Carry the authored tags onto the runtime row so tasks can target by tag.
	// Every pass (idempotent), so an edited tag propagates on the next converge.
	tagsJSON := []byte("{}")
	if len(tags) > 0 {
		if b, err := json.Marshal(tags); err == nil {
			tagsJSON = b
		}
	}
	if err := q.SetDeployedObjectTags(ctx, db.SetDeployedObjectTagsParams{ID: obj.ID, Tags: tagsJSON}); err != nil {
		return fmt.Errorf("setting tags: %w", err)
	}
	// Hold a guest's deployment until its network is ready. A failed network may
	// still exist at the hoster with an unusable allocation.
	// Fail its dependent guests without bootstrapping them on that network. This
	// also lets the build settle instead of leaving those guests pending forever.
	if deployTasks && !infraUp(obj.Status) && obj.Status != "destroy_queued" && obj.Status != "destroying" && obj.Status != "destroyed" && networkName != "" && !networkUp[networkName] {
		if networkFailed[networkName] && obj.Status != "deploy_failed" {
			msg := fmt.Sprintf("network %q failed to deploy; resolve its failure before redeploying", networkName)
			_, err := q.MarkDeployedObjectDeployFailed(ctx, db.MarkDeployedObjectDeployFailedParams{ID: obj.ID, LastError: db.StrPtr(msg)})
			if err == nil {
				decide(ctx, q, buildID, obj.ID, "reconcile.deploy_failed", fmt.Sprintf("%s: %s", objectLabel(obj, int32(teamNum)), msg), nil)
			}
			return err
		}
		decideRepeating(ctx, q, "network-wait/"+obj.ID.String(), buildID, obj.ID, "reconcile.waiting", fmt.Sprintf("%s: waiting for network %q to be up before deploying", objectLabel(obj, int32(teamNum)), networkName), nil)
		return nil
	}
	// The box deploys ahead of time: its deploy task is no longer gated on
	// depends_on (that only ever waited for a dependency's instance to exist,
	// which isn't the same as the dependency being configured). Ordering is
	// handled instead by deploying roots-first (see orderedPlacements), and
	// correctness by holding this object's STEP execution below until every
	// dependency has actually finished.
	created, err := reconcileObject(ctx, q, buildID, obj, fp, infraFP, holdOnStepChange, priority, deployTasks)
	if err != nil {
		return err
	}
	if created && len(rctx.Schedule) > 0 {
		// Exactly once per real (re)deploy, not every 2-second reconcile
		// pass -- see reconcileObject's own doc comment on `created`.
		if err := materializeSchedule(ctx, q, buildID, obj, rctx.Schedule, anchors); err != nil {
			return fmt.Errorf("materializing schedule: %w", err)
		}
	}
	// Once the box is up AND every dependency has finished configuring, queue
	// this object's authored steps. obj here is this pass's snapshot (its
	// status and steps_materialized_at are current via EnsureDeployedObject's
	// RETURNING), so this converges over successive passes as dependencies
	// finish.
	if err := materializeStepsIfReady(ctx, q, repoRoot, c, envName, buildID, obj, teamNum, asName, dependsOn, placed, finishedDeps, failedDeps, gates, materializeActive); err != nil {
		return fmt.Errorf("materializing steps: %w", err)
	}
	return nil
}

// reconcileObject is the actual diff, per object: does the desired
// fingerprint match what's already deployed? See fingerprint.go's doc
// comment and migrations/00003's comment on deployed_object for the two
// states this can land in. deployTasks is "Build and deploy are
// separate verbs" made real (see Reconcile's own doc comment on it):
// false means a `planned` build, which still computes and records
// everything above (real fingerprint, real EnsureDeployedObject row --
// "fully inspectable") but must not create a single task that would
// touch a hoster. created is true only when this call is the one that
// newly opened a deploy task -- reconcileCopy uses it to materialize a
// host/container's own schedule: entries exactly once per real
// (re)deploy, not on every reconcile pass over an already-deploying
// object.
//
// Deploy is NOT gated on depends_on: the box is provisioned as soon as its
// turn comes (ordered roots-first by the caller), ahead of its dependencies
// finishing. depends_on now holds only step EXECUTION (materializeStepsIfReady),
// not the infrastructure deploy.
//
// priority is recorded on the deploy task's payload (see deployTaskPayload) so
// the runner leases, and the builder admits, critical-path objects first.
//
// desiredInfraFP is the infrastructure half of desiredFP (InfraFingerprint).
// With holdOnStepChange, an object whose fingerprint differs only in its
// steps -- the infrastructure half still matches what it was deployed with --
// is marked steps-changed and left alone for an operator to restart from a
// step (docs/plans/restart-from-step.md §6), instead of being recreated. An
// object deployed before infra_fingerprint existed has none recorded and is
// recreated as before.
func reconcileObject(ctx context.Context, q *db.Queries, buildID pgtype.UUID, obj db.DeployedObject, desiredFP, desiredInfraFP string, holdOnStepChange bool, priority int, deployTasks bool) (created bool, err error) {
	// A rebuild recorded while the object was busy is carried out as soon as
	// it isn't: destroy what's there (by its reference, or by name if it has
	// none yet), and the runner resets it for a fresh deploy.
	if obj.RebuildRequestedAt.Valid && deployTasks && obj.Status != "destroyed" {
		payload, err := json.Marshal(map[string]any{"remove": false, "external_ref": db.StrOrEmpty(obj.ExternalRef)})
		if err != nil {
			return false, err
		}
		task, err := q.CreateTaskIfNoneOpen(ctx, db.CreateTaskIfNoneOpenParams{BuildID: buildID, DeployedObjectID: obj.ID, Kind: "destroy_" + obj.Kind, Payload: payload})
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil // still busy; the request stays recorded
		}
		if err != nil {
			return false, fmt.Errorf("queuing the recorded rebuild of %s: %w", objectLabel(obj, 0), err)
		}
		if err := q.ClearRebuildRequested(ctx, obj.ID); err != nil {
			return false, err
		}
		if _, err := q.MarkDeployedObjectDestroyQueued(ctx, obj.ID); err != nil {
			return false, err
		}
		decide(ctx, q, buildID, obj.ID, "rebuild.destroy", fmt.Sprintf("%s: the task it was busy with ended; destroy queued for the rebuild recorded %s ago", objectLabel(obj, 0), time.Since(obj.RebuildRequestedAt.Time).Round(time.Second)),
			map[string]any{"task_id": task.ID.String(), "external_ref": db.StrOrEmpty(obj.ExternalRef)})
		return false, nil
	}
	if infraUp(obj.Status) {
		if obj.Fingerprint == desiredFP {
			return false, nil // already correct
		}
		stepsOnly := obj.InfraFingerprint != "" && obj.InfraFingerprint == desiredInfraFP
		if holdOnStepChange && stepsOnly {
			n, err := q.MarkDeployedObjectStepsChanged(ctx, db.MarkDeployedObjectStepsChangedParams{ID: obj.ID, StepsChangedFingerprint: desiredFP})
			if err != nil {
				return false, fmt.Errorf("marking steps changed: %w", err)
			}
			if n > 0 {
				q.CreateEvent(ctx, db.CreateEventParams{BuildID: buildID, Kind: "steps.changed", Payload: []byte("{}"),
					Message: fmt.Sprintf("%s: its steps or scripts changed in the content; held rather than recreated (this build holds hosts on step-only changes) -- restart it from a step to apply the change", db.StrOrEmpty(obj.AsName))})
			}
			return false, nil
		}
		// "rebuild means recreate": destroy first, not remove -- a later
		// pass (once destroyed resets it to pending) will redeploy fresh.
		what := "its infrastructure (image, size, disk, ports or dependencies)"
		if stepsOnly {
			what = "only its steps or scripts (this build recreates on any change)"
		} else if obj.InfraFingerprint == "" {
			what = "its content (deployed before changes could be told apart)"
		}
		requested, err := requestDestroy(ctx, q, buildID, obj, false, deployTasks)
		if err != nil {
			return false, err
		}
		if requested {
			decide(ctx, q, buildID, obj.ID, "reconcile.recreate", fmt.Sprintf("%s: %s changed in the content; destroying it to redeploy fresh", objectLabel(obj, 0), what),
				map[string]any{"deployed_fingerprint": obj.Fingerprint, "desired_fingerprint": desiredFP, "steps_only": stepsOnly})
		} else if deployTasks {
			decideRepeating(ctx, q, "recreate-wait/"+obj.ID.String(), buildID, obj.ID, "reconcile.waiting", fmt.Sprintf("%s: needs recreating but already has an open task; waiting for it", objectLabel(obj, 0)), nil)
		}
		return false, nil
	}
	if obj.Status == "destroyed" {
		return false, nil // a torn-down object stays gone
	}
	if !deployTasks {
		return false, nil // planned: the object row and its fingerprint are already recorded above; stop here
	}
	// Before anything is (re)deployed in this object's place, its last destroy
	// -- a rebuild's, a content change's -- must actually have finished;
	// otherwise the old instance stays at the hoster beside the new one, or
	// (if the deploy can't even be queued behind an open task) the object
	// sits pending with nothing in the log saying why. So: an open destroy is
	// waited for, and said; a transiently failed one is queued again, with
	// the same hoster reference, after a backoff; a permanently failed one
	// stops the object with its error. Only a done destroy (or none) clears
	// the way to deploy.
	if proceed, err := settleLastDestroy(ctx, q, buildID, obj); err != nil || !proceed {
		return false, err
	}
	if obj.Status == "deploy_failed" {
		if proceed, err := settleLastDeploy(ctx, q, buildID, obj); err != nil || !proceed {
			return false, err
		}
	}
	if obj.Status == "destroy_queued" {
		// Waiting to be destroyed, yet no destroy is open or failed
		// (settleLastDestroy would have stopped here): the one it was waiting
		// for went away -- withdrawn, or lost. Queue it again, with the hoster
		// reference the object still holds, rather than wait forever.
		payload, err := json.Marshal(map[string]any{"remove": false, "external_ref": db.StrOrEmpty(obj.ExternalRef)})
		if err != nil {
			return false, err
		}
		task, err := q.CreateTaskIfNoneOpen(ctx, db.CreateTaskIfNoneOpenParams{BuildID: buildID, DeployedObjectID: obj.ID, Kind: "destroy_" + obj.Kind, Payload: payload})
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		if err != nil {
			return false, fmt.Errorf("re-queuing the destroy of %s: %w", objectLabel(obj, 0), err)
		}
		decide(ctx, q, buildID, obj.ID, "reconcile.destroy_retry", fmt.Sprintf("%s: was waiting to be destroyed but had no destroy queued; queued one (hoster ref %s)", objectLabel(obj, 0), db.StrOrEmpty(obj.ExternalRef)),
			map[string]any{"task_id": task.ID.String(), "external_ref": db.StrOrEmpty(obj.ExternalRef)})
		return false, nil
	}
	if obj.Status == "destroying" {
		// No destroy task is in flight (settleLastDestroy would have said), yet
		// the runner's reset to pending never happened: say so once a minute.
		decideRepeating(ctx, q, "destroying/"+obj.ID.String(), buildID, obj.ID, "reconcile.waiting", fmt.Sprintf("%s: marked destroying but no destroy task is open; waiting for the runner to reset it", objectLabel(obj, 0)), nil)
		return false, nil
	}
	// pending, deploying, or deploy_failed: ensure a deploy task exists.
	// CreateTaskIfNoneOpen is the idempotency guard here -- if one's
	// already leased/pending for this object, this is a no-op (and
	// ErrNoRows, its own signal for "nothing inserted" -- ends up back
	// here as created=false).
	payload, err := deployTaskPayload(desiredFP, desiredInfraFP, priority)
	if err != nil {
		return false, err
	}
	task, err := q.CreateTaskIfNoneOpen(ctx, db.CreateTaskIfNoneOpenParams{
		BuildID: buildID, DeployedObjectID: obj.ID, Kind: "deploy_" + obj.Kind, Payload: payload,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	why := "not deployed yet"
	if obj.Status == "deploy_failed" {
		why = "its last deploy failed; trying again"
	}
	decide(ctx, q, buildID, obj.ID, "reconcile.deploy", fmt.Sprintf("%s: %s; deploy task queued (priority %d)", objectLabel(obj, 0), why, priority),
		map[string]any{"task_id": task.ID.String(), "fingerprint": desiredFP, "priority": priority})
	return true, nil
}

// networkDeployPriority ranks a network's deploy above every host and container:
// each of a team's instances waits for its network to exist, so a network
// queued behind another team's hosts would stall its whole team.
const networkDeployPriority = 1 << 20

// deployTaskPayload is a deploy task's payload: the fingerprint it deploys and
// its critical-path priority. The lease queries order by payload->'priority'
// (internal/db/queries/runtime.sql) and the runner hands it to the builder as
// HostSpec/ContainerSpec.Priority. Omitted when 0, as on a destroy task.
func deployTaskPayload(fingerprint, infraFingerprint string, priority int) ([]byte, error) {
	p := map[string]any{"fingerprint": fingerprint, "infra_fingerprint": infraFingerprint}
	if priority > 0 {
		p["priority"] = priority
	}
	return json.Marshal(p)
}

// requestDestroy queues a destroy for obj and reports whether it did (false
// when the object already has an open task, which is then the one to wait
// for, or when the build is only planned).
func requestDestroy(ctx context.Context, q *db.Queries, buildID pgtype.UUID, obj db.DeployedObject, remove bool, deployTasks bool) (bool, error) {
	if !deployTasks {
		return false, nil // a `planned` build never destroys anything real either
	}
	payload, err := json.Marshal(map[string]bool{"remove": remove})
	if err != nil {
		return false, err
	}
	_, err = q.CreateTaskIfNoneOpen(ctx, db.CreateTaskIfNoneOpenParams{
		BuildID: buildID, DeployedObjectID: obj.ID, Kind: "destroy_" + obj.Kind, Payload: payload,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	// Shown as waiting to be destroyed until the runner starts on it.
	if _, err := q.MarkDeployedObjectDestroyQueued(ctx, obj.ID); err != nil {
		return false, err
	}
	return true, nil
}

// placedNames is the set of object and network names that appear in an
// environment's topology -- what depends_on ordering can wait on.
func placedNames(networks map[string]map[string][]loader.Copy) map[string]bool {
	out := make(map[string]bool)
	for netName, objs := range networks {
		out[netName] = true
		for objName := range objs {
			out[objName] = true
		}
	}
	return out
}

// placement is one host/container copy to reconcile, carrying its depends_on
// depth so the whole set can be ordered roots-first.
type placement struct {
	networkName string
	objectName  string
	as          string
	kind        string
	depth       int
	priority    int // how many objects transitively depends_on this one (loader.DependentCounts)
}

// orderedPlacements flattens an environment's topology into every copy it
// places, sorted by depends_on depth (roots first) with a deterministic
// tiebreak. Deploy tasks are created in this order so a dependency's box is
// provisioned before its dependents' -- the tree-priority the box-deploy
// decoupling asks for. The order is identical for every team (the topology is
// too), so callers compute it once.
func orderedPlacements(env *loader.Environment, containerNames map[string]bool, depth, priority map[string]int) []placement {
	var out []placement
	for networkName, objs := range env.Networks {
		for objectName, copies := range objs {
			kind := "host"
			if containerNames[objectName] {
				kind = "container"
			}
			for _, cp := range copies {
				out = append(out, placement{networkName, objectName, cp.As, kind, depth[objectName], priority[objectName]})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].depth != out[j].depth {
			return out[i].depth < out[j].depth
		}
		if out[i].priority != out[j].priority {
			return out[i].priority > out[j].priority
		}
		if out[i].objectName != out[j].objectName {
			return out[i].objectName < out[j].objectName
		}
		return out[i].as < out[j].as
	})
	return out
}

// teamDepReadiness summarizes a team's objects for depends_on gating of step
// execution. finished[name] is true only when EVERY copy of that object has
// fully FINISHED configuring (status "finished"); failed[name] is true when
// ANY copy reached a terminal failure (deploy_failed/build_failed/invalid). An
// object with no row yet is neither finished nor failed. A dependent's steps
// are materialized only once all its dependencies are finished (the box having
// merely come up is no longer enough -- a DC must be a configured DC); a
// dependency that failed lets the dependent fail fast instead of waiting
// forever.
func teamDepReadiness(ctx context.Context, q *db.Queries, teamID pgtype.UUID) (finished, failed, networkUp, networkFailed map[string]bool, err error) {
	objs, err := q.ListDeployedObjectsByTeam(ctx, teamID)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	finished = make(map[string]bool, len(objs))
	failed = make(map[string]bool, len(objs))
	// A host/container's first deploy waits for its network's subnet/bridge to
	// exist (see reconcileCopy): networkUp[name] once the network's infrastructure
	// is up, networkFailed[name] once its deploy terminally failed (so a host
	// stops waiting and fails on its own instead of stranding forever).
	networkUp = make(map[string]bool, len(objs))
	networkFailed = make(map[string]bool, len(objs))
	seen := make(map[string]bool, len(objs))
	for _, o := range objs {
		isFail := o.Status == "deploy_failed" || o.Status == "build_failed" || o.Status == "invalid"
		if o.Kind == "network" {
			networkUp[o.ObjectName] = infraUp(o.Status)
			networkFailed[o.ObjectName] = isFail
			continue
		}
		isFin := o.Status == "finished"
		if !seen[o.ObjectName] {
			seen[o.ObjectName] = true
			finished[o.ObjectName] = isFin
			failed[o.ObjectName] = isFail
		} else {
			finished[o.ObjectName] = finished[o.ObjectName] && isFin
			failed[o.ObjectName] = failed[o.ObjectName] || isFail
		}
	}
	return finished, failed, networkUp, networkFailed, nil
}

// effectiveTags computes the tags stored on a deployed_object, cascading from
// least to most specific so the most specific wins: environment tags, then the
// object's network's tags, then the tags of every script the object runs (a
// script's tags "flow to every host/container running this script"), then the
// object's own host/container tags. This is what makes environment/network
// tags and script tags real -- before, only the object's own tags were stored,
// so tag-targeting couldn't see any of the cascade.
func effectiveTags(c *loader.Content, envName, networkName, objectName string, steps, sched []loader.Step, ownTags map[string]string) map[string]string {
	out := map[string]string{}
	if env := findEnvironment(c, envName); env != nil {
		for k, v := range env.Tags {
			out[k] = v
		}
	}
	if net := findNetwork(c, networkName); net != nil {
		for k, v := range net.Tags {
			out[k] = v
		}
	}
	seenScript := map[string]bool{}
	for _, st := range append(append([]loader.Step{}, steps...), sched...) {
		name, ok := st["script"].(string)
		if !ok || seenScript[name] {
			continue
		}
		seenScript[name] = true
		if s := findScript(c, name); s != nil {
			for k, v := range s.Tags {
				out[k] = v
			}
		}
	}
	for k, v := range ownTags {
		out[k] = v
	}
	return out
}

// --- small content lookups, local to this package (internal/render's
// equivalents are unexported, and orchestrator needs a couple it doesn't
// export: raw Host/Container/Network definitions, not a resolved Context) ---

func findEnvironment(c *loader.Content, name string) *loader.Environment {
	for i := range c.Environments {
		if c.Environments[i].Name == name {
			return &c.Environments[i]
		}
	}
	return nil
}

func findNetwork(c *loader.Content, name string) *loader.Network {
	for i := range c.Networks {
		if c.Networks[i].Name == name {
			return &c.Networks[i]
		}
	}
	return nil
}

func findHost(c *loader.Content, name string) *loader.Host {
	for i := range c.Hosts {
		if c.Hosts[i].Name == name {
			return &c.Hosts[i]
		}
	}
	return nil
}

func findContainer(c *loader.Content, name string) *loader.Container {
	for i := range c.Containers {
		if c.Containers[i].Name == name {
			return &c.Containers[i]
		}
	}
	return nil
}

// settleLastDestroy looks at obj's most recent destroy task and reports
// whether a deploy may go ahead (true: the last destroy is done, or there
// never was one). See reconcileObject. The backoff before re-queuing a
// failed destroy is the teardown's (teardownRequeueBackoff), so a flaky
// hoster is retried the same way whichever path asked for the destroy.
func settleLastDestroy(ctx context.Context, q *db.Queries, buildID pgtype.UUID, obj db.DeployedObject) (bool, error) {
	last, err := q.GetLatestDestroyTaskForObject(ctx, obj.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("reading %s's last destroy task: %w", objectLabel(obj, 0), err)
	}
	age := time.Since(last.UpdatedAt.Time).Round(time.Second)
	switch last.Status {
	case "done":
		return true, nil
	case "pending", "leased":
		// The object's own status (waiting to be destroyed / destroying) says
		// this; only an object that doesn't show it gets the log line.
		if obj.Status == "destroy_queued" || obj.Status == "destroying" {
			return false, nil
		}
		state := last.Status
		if last.Status == "leased" {
			state = fmt.Sprintf("leased by %s until %s", db.StrOrEmpty(last.LeaseOwner), last.LeaseExpiresAt.Time.Format(time.TimeOnly))
		}
		msg := fmt.Sprintf("%s: its destroy task is %s (attempt %d, %s old); not deploying until the old instance is gone", objectLabel(obj, 0), state, last.Attempts, age)
		if e := db.StrOrEmpty(last.LastError); e != "" {
			msg += "; last error: " + e
		}
		decideRepeating(ctx, q, "destroy-open/"+obj.ID.String(), buildID, obj.ID, "reconcile.waiting", msg, map[string]any{"task_id": last.ID.String(), "task_status": last.Status, "attempts": last.Attempts, "last_error": db.StrOrEmpty(last.LastError)})
		return false, nil
	case "failed":
		errText := db.StrOrEmpty(last.LastError)
		if last.Permanent {
			if obj.Status != "deploy_failed" {
				msg := fmt.Sprintf("the old instance could not be destroyed (permanent error): %s -- fix the builder or remove it at the hoster, then rebuild", errText)
				if _, err := q.MarkDeployedObjectDeployFailed(ctx, db.MarkDeployedObjectDeployFailedParams{ID: obj.ID, LastError: db.StrPtr(msg)}); err != nil {
					return false, err
				}
				decide(ctx, q, buildID, obj.ID, "reconcile.deploy_failed", fmt.Sprintf("%s: %s", objectLabel(obj, 0), msg), map[string]any{"task_id": last.ID.String(), "error": errText})
			}
			return false, nil
		}
		wait := teardownRequeueBackoff(int(last.FailedCount))
		if time.Since(last.UpdatedAt.Time) < wait {
			decideRepeating(ctx, q, "destroy-backoff/"+obj.ID.String(), buildID, obj.ID, "reconcile.waiting", fmt.Sprintf("%s: its destroy failed %d time(s) (%s); trying again in %s", objectLabel(obj, 0), last.FailedCount, errText, (wait-time.Since(last.UpdatedAt.Time)).Round(time.Second)), nil)
			return false, nil
		}
		// Same payload as the failed one, so the real hoster reference a
		// rebuild captured (the row's own was cleared) is kept.
		task, err := q.CreateTaskIfNoneOpen(ctx, db.CreateTaskIfNoneOpenParams{BuildID: buildID, DeployedObjectID: obj.ID, Kind: last.Kind, Payload: last.Payload})
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil // something else opened a task meanwhile
		}
		if err != nil {
			return false, fmt.Errorf("re-requesting destroy of %s: %w", objectLabel(obj, 0), err)
		}
		decide(ctx, q, buildID, obj.ID, "reconcile.destroy_retry", fmt.Sprintf("%s: destroy queued again after %d failed attempt(s) (%s); the old instance must go before it's redeployed", objectLabel(obj, 0), last.FailedCount, errText),
			map[string]any{"task_id": task.ID.String(), "failed_destroys": last.FailedCount, "last_error": errText})
		return false, nil
	}
	return true, nil
}

// maxConsecutiveDeployFailures is how many deploys of one object may fail in
// a row before the reconciler stops queueing new ones. Each deploy task has
// its own retries already; past this the failure is the cluster's or the
// content's to fix, and a rebuild (which resets the count) is the operator's
// way to try again.
const maxConsecutiveDeployFailures = 5

// settleLastDeploy decides whether a deploy_failed object gets another
// deploy now. It used to get one on every pass, immediately -- a loop that
// kept a builder slot busy failing the same way (an instance that already
// existed, a cluster that refused) for as long as the build lived. A
// permanent failure stops; otherwise retries back off (the teardown's
// schedule, 30 s doubling to 10 m) and stop after
// maxConsecutiveDeployFailures, each decision recorded.
func settleLastDeploy(ctx context.Context, q *db.Queries, buildID pgtype.UUID, obj db.DeployedObject) (bool, error) {
	last, err := q.GetLatestDeployTaskForObject(ctx, obj.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("reading %s's last deploy task: %w", objectLabel(obj, 0), err)
	}
	if last.Status != "failed" {
		return true, nil
	}
	errText := db.StrOrEmpty(last.LastError)
	switch {
	case last.Permanent:
		decideRepeating(ctx, q, "deploy-permanent/"+obj.ID.String(), buildID, obj.ID, "reconcile.deploy_stopped",
			fmt.Sprintf("%s: not retrying its deploy -- it failed with an error retrying can't fix: %s (fix it, then rebuild the host)", objectLabel(obj, 0), errText), nil)
		return false, nil
	case int(last.FailedCount) >= maxConsecutiveDeployFailures:
		decideRepeating(ctx, q, "deploy-giveup/"+obj.ID.String(), buildID, obj.ID, "reconcile.deploy_stopped",
			fmt.Sprintf("%s: not retrying its deploy -- it has failed %d times in a row (last: %s); rebuild the host to try again", objectLabel(obj, 0), last.FailedCount, errText), nil)
		return false, nil
	}
	wait := teardownRequeueBackoff(int(last.FailedCount))
	if since := time.Since(last.UpdatedAt.Time); since < wait {
		decideRepeating(ctx, q, "deploy-backoff/"+obj.ID.String(), buildID, obj.ID, "reconcile.waiting",
			fmt.Sprintf("%s: its deploy failed %d time(s) in a row (%s); trying again in %s", objectLabel(obj, 0), last.FailedCount, errText, (wait-since).Round(time.Second)), nil)
		return false, nil
	}
	return true, nil
}
