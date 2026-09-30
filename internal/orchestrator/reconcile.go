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
func Reconcile(ctx context.Context, pool *pgxpool.Pool, repoRoot string, buildID pgtype.UUID) error {
	q := db.New(pool)

	build, err := q.GetBuild(ctx, buildID)
	if err != nil {
		return fmt.Errorf("loading build: %w", err)
	}

	c, err := loader.Load(repoRoot)
	if err != nil {
		return fmt.Errorf("loading content: %w", err)
	}
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
	anchors := anchorsFromEnvironment(env)
	// Object + network names placed in this environment's topology -- the set
	// depends_on ordering can actually wait on (a dependency not placed here
	// can't be ordered on, so it's ignored rather than wedging the build).
	placed := placedNames(env.Networks)

	for teamNum := 1; teamNum <= env.Teams; teamNum++ {
		team, err := q.EnsureTeam(ctx, db.EnsureTeamParams{BuildID: buildID, TeamNumber: int32(teamNum)})
		if err != nil {
			return fmt.Errorf("team %d: %w", teamNum, err)
		}

		// This team's dependency readiness as of the start of this pass: an
		// object name is "ready" only when every one of its copies is up.
		readyDeps, err := teamDepReadiness(ctx, q, team.ID)
		if err != nil {
			return fmt.Errorf("team %d readiness: %w", teamNum, err)
		}

		desired := make(map[string]bool)

		for networkName := range env.Networks {
			desired["network/"+networkName] = true
			if err := reconcileNetwork(ctx, q, c, team.ID, buildID, networkName, deployTasks); err != nil {
				return fmt.Errorf("team %d, network %q: %w", teamNum, networkName, err)
			}
		}

		for networkName, objs := range env.Networks {
			for objectName, copies := range objs {
				kind := "host"
				if containerNames[objectName] {
					kind = "container"
				}
				for _, cp := range copies {
					desired[kind+"/"+cp.As] = true
					if err := reconcileCopy(ctx, q, repoRoot, c, env.Name, team.ID, buildID, teamNum, kind, objectName, cp.As, networkName, deployTasks, anchors, placed, readyDeps); err != nil {
						return fmt.Errorf("team %d, %s %q: %w", teamNum, kind, cp.As, err)
					}
				}
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

	return nil
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
		case "pending", "deploying":
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
	// Infra still being created, or nothing actually stood up (an
	// all-teardown or empty build) -- not a deploy to advance yet.
	if infraPending > 0 || live == 0 {
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
		if err := requestDestroy(ctx, q, buildID, obj, true, deployTasks); err != nil {
			return fmt.Errorf("removing %s %q: %w", obj.Kind, identity, err)
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
	_, err = reconcileObject(ctx, q, buildID, obj, fp, deployTasks, true) // networks have no depends_on
	return err
}

func reconcileCopy(ctx context.Context, q *db.Queries, repoRoot string, c *loader.Content, envName string, teamID, buildID pgtype.UUID, teamNum int, kind, objectName, asName, networkName string, deployTasks bool, anchors schedule.Anchors, placed, readyDeps map[string]bool) error {
	rctx, err := render.Resolve(c, envName, asName, teamNum)
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
		ports, dependsOn, ownTags, steps, sched = ct.Ports, ct.DependsOn, ct.Tags, ct.Steps, ct.Schedule
	}
	// Tags carried onto the runtime row cascade from least to most specific
	// (most specific wins): environment -> network -> the scripts this object
	// runs -> the object's own tags.
	tags := effectiveTags(c, envName, networkName, objectName, steps, sched, ownTags)

	fp, err := Fingerprint(repoRoot, c, rctx, disk, ports, dependsOn)
	if err != nil {
		return fmt.Errorf("fingerprinting: %w", err)
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
	// depends_on ordering: this copy is ready to deploy only when every
	// dependency that is actually placed in this environment is up for this
	// team. A dependency not placed here is ignored (it can't be ordered on),
	// and readyDeps is this team's converged state as of the pass's start, so
	// ordering emerges over successive 2-second passes.
	depsReady := true
	for _, d := range dependsOn {
		if placed[d] && !readyDeps[d] {
			depsReady = false
			break
		}
	}
	created, err := reconcileObject(ctx, q, buildID, obj, fp, deployTasks, depsReady)
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
func reconcileObject(ctx context.Context, q *db.Queries, buildID pgtype.UUID, obj db.DeployedObject, desiredFP string, deployTasks, depsReady bool) (created bool, err error) {
	if infraUp(obj.Status) {
		if obj.Fingerprint == desiredFP {
			return false, nil // already correct
		}
		// "rebuild means recreate": destroy first, not remove -- a later
		// pass (once destroyed resets it to pending) will redeploy fresh.
		return false, requestDestroy(ctx, q, buildID, obj, false, deployTasks)
	}
	if obj.Status == "destroying" || obj.Status == "destroyed" {
		// let the in-flight (or already-completed-but-not-yet-reset)
		// destroy finish; nothing new to request this pass.
		return false, nil
	}
	if !deployTasks {
		return false, nil // planned: the object row and its fingerprint are already recorded above; stop here
	}
	if !depsReady {
		// depends_on ordering: hold this object's deploy task until every
		// dependency is up. The object row + fingerprint are already recorded
		// above (so it stays inspectable); a later pass creates the task once
		// dependencies converge. checkDependsOnCycles rules out a deadlock.
		return false, nil
	}
	// pending, deploying, or deploy_failed: ensure a deploy task exists.
	// CreateTaskIfNoneOpen is the idempotency guard here -- if one's
	// already leased/pending for this object, this is a no-op (and
	// ErrNoRows, its own signal for "nothing inserted" -- ends up back
	// here as created=false).
	payload, err := json.Marshal(map[string]string{"fingerprint": desiredFP})
	if err != nil {
		return false, err
	}
	_, err = q.CreateTaskIfNoneOpen(ctx, db.CreateTaskIfNoneOpenParams{
		BuildID: buildID, DeployedObjectID: obj.ID, Kind: "deploy_" + obj.Kind, Payload: payload,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func requestDestroy(ctx context.Context, q *db.Queries, buildID pgtype.UUID, obj db.DeployedObject, remove bool, deployTasks bool) error {
	if !deployTasks {
		return nil // a `planned` build never destroys anything real either
	}
	payload, err := json.Marshal(map[string]bool{"remove": remove})
	if err != nil {
		return err
	}
	_, err = q.CreateTaskIfNoneOpen(ctx, db.CreateTaskIfNoneOpenParams{
		BuildID: buildID, DeployedObjectID: obj.ID, Kind: "destroy_" + obj.Kind, Payload: payload,
	})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	return nil
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

// teamDepReadiness maps each object name to whether ALL of its copies for this
// team are up (infraUp). An object with no row yet, or any copy not up, is not
// ready. Used to hold a dependent's deploy until its dependencies converge.
func teamDepReadiness(ctx context.Context, q *db.Queries, teamID pgtype.UUID) (map[string]bool, error) {
	objs, err := q.ListDeployedObjectsByTeam(ctx, teamID)
	if err != nil {
		return nil, err
	}
	ready := make(map[string]bool, len(objs))
	seen := make(map[string]bool, len(objs))
	for _, o := range objs {
		up := infraUp(o.Status)
		if !seen[o.ObjectName] {
			seen[o.ObjectName] = true
			ready[o.ObjectName] = up
		} else {
			ready[o.ObjectName] = ready[o.ObjectName] && up
		}
	}
	return ready, nil
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
