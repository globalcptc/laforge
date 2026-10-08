// Package runner is "a runner takes a task with FOR UPDATE SKIP LOCKED,
// gets a lease, and heartbeats" made real, on top of internal/db's
// generated LeaseTask/HeartbeatTask/CompleteTask/RetryTask/FailTask
// queries (the exact lease-reclaim mechanism proven earlier against
// Postgres). A Runner holds nothing durable of its own -- every fact it needs
// (the task, the deployed_object, the team, the content) is re-read from
// Postgres or the checkout on every task, so killing a runner mid-task
// loses nothing that wasn't already either safely uncommitted or already
// recorded. See internal/chaos for the SIGKILL proofs.
package runner

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/globalcptc/laforge/internal/agentdelivery"
	"github.com/globalcptc/laforge/internal/builder"
	"github.com/globalcptc/laforge/internal/checkout"
	"github.com/globalcptc/laforge/internal/db"
	"github.com/globalcptc/laforge/internal/loader"
	"github.com/globalcptc/laforge/internal/render"
)

// MaxAttempts is the cap on "retries use backoff and a cap": once a
// task's attempts (incremented on every lease, including a reclaim after
// a dead runner) reaches this, a further failure is terminal rather than
// going back to pending.
const MaxAttempts = 3

// retryBackoffBase and retryBackoffCap bound the delay RetryBackoff spaces
// retries by. The base is comfortably longer than the time a torn-down
// network needs to drain its used_by once its instances detach -- the
// concrete failure that motivated real backoff: without it all MaxAttempts
// fire inside one second and the network is left stuck in 'destroying'.
const (
	retryBackoffBase = 15 * time.Second
	retryBackoffCap  = 60 * time.Second
)

// RetryBackoff is how long a failed-but-retryable task is held before it can
// be re-leased, scaled by how many attempts it has already had so a transient
// condition has time to clear. Linear in the attempt count, capped. The base
// comes from r.RetryBackoffBase, or retryBackoffBase when that is zero.
func (r *Runner) RetryBackoff(attempts int32) time.Duration {
	base := r.RetryBackoffBase
	if base <= 0 {
		base = retryBackoffBase
	}
	if attempts < 1 {
		attempts = 1
	}
	d := time.Duration(attempts) * base
	cap := retryBackoffCap
	if base > cap {
		cap = base
	}
	if d > cap {
		d = cap
	}
	return d
}

// BuilderResolver answers "which builder.Builder should this task use" --
// mirrors orchestrator.ContentResolver's own per-build resolution,
// letting one Runner process serve tasks from builds that use different
// builder configs (internal/builderconfig) instead of being fixed to one
// builder for its whole lifetime. See Runner.ResolveBuilderFromDB for
// the real, database-backed implementation cmd/laforge-runner uses.
type BuilderResolver func(ctx context.Context, task db.Task) (builder.Builder, error)

type Runner struct {
	Pool *pgxpool.Pool
	// Builder is used directly when Builders is nil -- the original,
	// still-supported single-builder mode every test in this codebase
	// already uses. Real production runners set Builders (see
	// ResolveBuilderFromDB) so each task resolves its OWN build's own
	// builder_config by name -- the "what about the
	// MicroCloud builder" gap -- instead of every task sharing one fixed
	// builder regardless of which cluster its own configured_build
	// actually names.
	Builder  builder.Builder
	Builders BuilderResolver
	// RepoRoot is used directly when Checkouts is nil -- the original,
	// still-supported single-repository mode. Real production runners
	// set Checkouts once GitHub credentials are configured (see
	// checkout.FromEnv) so each task resolves its OWN repository/commit
	// (see repoRootFor) -- instead of
	// every task sharing one fixed checkout regardless of which
	// repository it actually belongs to.
	RepoRoot          string
	Checkouts         *checkout.Cache
	ID                string
	LeaseDuration     time.Duration
	HeartbeatInterval time.Duration
	// RetryBackoffBase scales how long a failed-but-retryable task is held
	// before it can be re-leased (see RetryBackoff). Zero means the
	// production default (retryBackoffBase); tests that exercise the retry
	// path set it small so they don't wait real seconds.
	RetryBackoffBase time.Duration

	// AfterBuilderCall, if set, is called immediately after a builder
	// call succeeds, before the result is recorded in Postgres. Used only
	// by cmd/laforge-runner's crash-injection flags and internal/chaos:
	// setting it to os.Exit(9) simulates "did the real (idempotent) work,
	// then died before marking the task complete" -- the exact scenario
	// proven earlier against a spike schema;
	// this is the same proof against the real one.
	AfterBuilderCall func()

	// Delivery, when configured (see agentdelivery.Load), makes the runner
	// deliver an agent to every host it deploys: a per-host binary stored
	// for the api to serve, and cloud-init user-data on the instance that
	// downloads and runs it. Nil/disabled means "deploy without agents,"
	// the same as before this existed -- every test that doesn't set it is
	// unaffected.
	Delivery *agentdelivery.Config

	// BuildID, if set, restricts leasing to this one build (LeaseTaskForBuild
	// instead of LeaseTask). Real production runners never set this --
	// see LeaseTaskForBuild's doc comment: this exists purely so tests in
	// different packages, running in parallel against the same shared
	// Postgres database, don't lease and complete each other's tasks.
	BuildID *pgtype.UUID
}

// LeaseOne claims the next available task, if any. Split out from
// executing it so a caller (cmd/laforge-runner's -crash-after-lease flag,
// or a chaos test) can inject a crash *between* leasing and doing any
// work at all -- proving a pure crash, zero work done, still converges
// once the lease expires. Returns (nil, nil) when nothing is leasable.
func (r *Runner) LeaseOne(ctx context.Context) (*db.Task, error) {
	q := db.New(r.Pool)
	var task db.Task
	var err error
	if r.BuildID != nil {
		// Scoped leasing -- see LeaseTaskForBuild's own doc comment: this
		// is for test isolation (several packages' tests share one
		// Postgres database and run in parallel), never set by the real
		// cmd/laforge-runner binary.
		task, err = q.LeaseTaskForBuild(ctx, db.LeaseTaskForBuildParams{
			LeaseOwner: db.StrPtr(r.ID), Column2: db.Interval(r.LeaseDuration), BuildID: *r.BuildID,
		})
	} else {
		task, err = q.LeaseTask(ctx, db.LeaseTaskParams{
			LeaseOwner: db.StrPtr(r.ID), Column2: db.Interval(r.LeaseDuration),
		})
	}
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("leasing a task: %w", err)
	}
	return &task, nil
}

// LeaseAndExecuteOne leases the next available task, if any, executes it
// against r.Builder, and records the result. Returns (false, nil) when
// there was nothing leasable right now, distinct from an error, so a
// caller (the loop in cmd/laforge-runner, or a chaos test driving this
// directly) can tell "queue empty" from "something went wrong."
func (r *Runner) LeaseAndExecuteOne(ctx context.Context) (bool, error) {
	task, err := r.LeaseOne(ctx)
	if err != nil {
		return false, err
	}
	if task == nil {
		return false, nil
	}
	return true, r.ExecuteAndRecord(ctx, *task)
}

// ExecuteAndRecord runs a task this runner already holds the lease for
// (from LeaseOne) and records the outcome -- the second half of
// LeaseAndExecuteOne, exposed separately for the same crash-injection
// reason LeaseOne is.
func (r *Runner) ExecuteAndRecord(ctx context.Context, task db.Task) error {
	q := db.New(r.Pool)
	stopHeartbeat := make(chan struct{})
	go r.heartbeatLoop(ctx, task.ID, stopHeartbeat)
	resolved, execErr := r.execute(ctx, q, task)
	close(stopHeartbeat)

	if resolved {
		// The task row itself is already gone (a "remove" destroy deletes
		// deployed_object, which cascades its task away) -- nothing left
		// to Complete/Retry/Fail. task_id is left NULL on this event since
		// the row it would reference no longer exists.
		r.logEvent(ctx, q, task.BuildID, pgtype.UUID{}, "task.removed", fmt.Sprintf("%s completed; object removed", task.Kind))
		return nil
	}

	if execErr != nil {
		if task.Attempts >= MaxAttempts {
			q.FailTask(ctx, db.FailTaskParams{ID: task.ID, LeaseOwner: db.StrPtr(r.ID), LastError: db.StrPtr(execErr.Error())})
			// Only a failed DEPLOY task marks the object deploy_failed -- a
			// failed destroy leaves the object in 'destroying' for a later
			// teardown/reconcile pass to retry, and must not be mislabeled
			// as an infra deploy failure.
			if task.DeployedObjectID.Valid && strings.HasPrefix(task.Kind, "deploy_") {
				q.MarkDeployedObjectDeployFailed(ctx, db.MarkDeployedObjectDeployFailedParams{ID: task.DeployedObjectID, LastError: db.StrPtr(execErr.Error())})
			}
			r.logEvent(ctx, q, task.BuildID, task.ID, "task.failed", fmt.Sprintf("%s: %v (attempt %d, giving up)", task.Kind, execErr, task.Attempts))
		} else {
			backoff := r.RetryBackoff(task.Attempts)
			q.RetryTask(ctx, db.RetryTaskParams{ID: task.ID, LeaseOwner: db.StrPtr(r.ID), LastError: db.StrPtr(execErr.Error()), Column4: db.Interval(backoff)})
			r.logEvent(ctx, q, task.BuildID, task.ID, "task.retry", fmt.Sprintf("%s: %v (attempt %d, retrying in %s)", task.Kind, execErr, task.Attempts, backoff))
		}
		return execErr
	}

	if _, err := q.CompleteTask(ctx, db.CompleteTaskParams{ID: task.ID, LeaseOwner: db.StrPtr(r.ID)}); err != nil {
		return fmt.Errorf("marking task complete: %w", err)
	}
	r.logEvent(ctx, q, task.BuildID, task.ID, "task.completed", task.Kind)
	return nil
}

func (r *Runner) heartbeatLoop(ctx context.Context, taskID pgtype.UUID, stop <-chan struct{}) {
	q := db.New(r.Pool)
	t := time.NewTicker(r.HeartbeatInterval)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			// Errors (including "0 rows affected" -- the lease was
			// reclaimed out from under us) are deliberately not acted on
			// here: there's no clean way to abort an in-flight builder
			// call from this goroutine, and the task-completion path
			// above already re-checks ownership on every write via
			// `WHERE lease_owner = $2`. A reclaimed runner's eventual
			// Complete/Retry/Fail calls simply affect zero rows and are
			// harmless -- whoever reclaimed the lease owns the outcome.
			q.HeartbeatTask(ctx, db.HeartbeatTaskParams{
				ID: taskID, LeaseOwner: db.StrPtr(r.ID), Column3: db.Interval(r.LeaseDuration),
			})
		case <-stop:
			return
		case <-ctx.Done():
			return
		}
	}
}

func (r *Runner) logEvent(ctx context.Context, q *db.Queries, buildID pgtype.UUID, taskID pgtype.UUID, kind, message string) {
	q.CreateEvent(ctx, db.CreateEventParams{
		BuildID: buildID, TaskID: taskID, Kind: kind, Message: message, Payload: []byte("{}"),
	})
}

// execute dispatches by task kind. The bool return is "the task row is
// already gone" -- true only for a destroy that removed its object
// (DeleteDeployedObject cascades the task away too), telling the caller
// not to try completing a row that no longer exists.
func (r *Runner) execute(ctx context.Context, q *db.Queries, task db.Task) (resolved bool, err error) {
	switch task.Kind {
	case "open_access", "close_access":
		return false, r.executeAccess(ctx, q, task)
	case "configure_network_access":
		return false, r.executeConfigureNetworkAccess(ctx, q, task)
	case "configure_external_access":
		return false, r.executeConfigureExternalAccess(ctx, q, task)
	}

	if !task.DeployedObjectID.Valid {
		return false, fmt.Errorf("task %s (kind %q) has no deployed_object_id", task.ID, task.Kind)
	}
	obj, err := q.GetDeployedObject(ctx, task.DeployedObjectID)
	if err != nil {
		return false, fmt.Errorf("loading deployed object: %w", err)
	}
	team, err := q.GetTeam(ctx, obj.TeamID)
	if err != nil {
		return false, fmt.Errorf("loading team: %w", err)
	}
	externalName := deterministicExternalName(task.BuildID, team.TeamNumber, obj)
	displayName := r.displayNameFor(ctx, q, task.BuildID, team.TeamNumber, obj)

	switch {
	case strings.HasPrefix(task.Kind, "deploy_"):
		return false, r.executeDeploy(ctx, q, task, obj, externalName, displayName, team.TeamNumber)
	case strings.HasPrefix(task.Kind, "destroy_"):
		return r.executeDestroy(ctx, q, task, obj, externalName, team.TeamNumber)
	default:
		return false, fmt.Errorf("unknown task kind %q", task.Kind)
	}
}

// deterministicExternalName is "deterministic resource names/tags, so a
// retry after partial success converges instead of duplicating" made
// concrete: computed the same way every time from facts that never change
// for a given deployed_object (which build, which team number, what kind,
// what identity), never stored anywhere and never needing to be -- any
// runner, at any point, recomputes the exact same name.
func deterministicExternalName(buildID pgtype.UUID, teamNumber int32, obj db.DeployedObject) string {
	identity := db.StrOrEmpty(obj.AsName)
	if identity == "" {
		identity = obj.ObjectName
	}
	return fmt.Sprintf("build-%s-team-%d-%s-%s", buildID.String(), teamNumber, obj.Kind, identity)
}

// displayNameFor is the human-readable hint the hoster shows for a host or
// container -- "<branch>-t<team>-<identity>" -- so an instance is
// recognizable at the hoster itself, not just through LaForge (direct
// product feedback: instances named "lf-<hash>" tell an operator nothing).
// Advisory: returns "" for networks (their names are too short-capped to
// carry it) and whenever the branch can't be resolved, in which case the
// builder keeps its deterministic hash name. The builder is responsible
// for sanitizing and length-capping this -- see incus.instanceName.
func (r *Runner) displayNameFor(ctx context.Context, q *db.Queries, buildID pgtype.UUID, teamNumber int32, obj db.DeployedObject) string {
	if obj.Kind == "network" {
		return ""
	}
	label := r.branchLabelFor(ctx, q, buildID)
	if label == "" {
		return ""
	}
	identity := db.StrOrEmpty(obj.AsName)
	if identity == "" {
		identity = obj.ObjectName
	}
	return fmt.Sprintf("%s-t%d-%s", label, teamNumber, identity)
}

// branchLabelFor resolves the configured build's branch for naming, falling
// back to the environment name for a build with no configured build behind
// it (an ad-hoc build), and to "" only when even that can't be read -- at
// which point the builder keeps its hash name rather than inventing a label.
func (r *Runner) branchLabelFor(ctx context.Context, q *db.Queries, buildID pgtype.UUID) string {
	build, err := q.GetBuild(ctx, buildID)
	if err != nil {
		return ""
	}
	if build.ConfiguredBuildID.Valid {
		if cb, err := q.GetConfiguredBuild(ctx, build.ConfiguredBuildID); err == nil && cb.Branch != "" {
			return cb.Branch
		}
	}
	return build.EnvironmentName
}

// networkDisplayName is the readable hoster label for a network --
// "t<team><network>" -- kept hyphen-free and short because a network name
// is capped hard at the hoster (see incus.networkName). Computed the same
// way whether it's the network's own deploy or an instance's NIC reference,
// so both sides agree on the resulting name.
func networkDisplayName(teamNumber int32, networkName string) string {
	return fmt.Sprintf("t%d%s", teamNumber, networkName)
}

// networkExternalName computes the SAME deterministic ExternalName the
// network's own deploy task computes (reconcileNetwork in
// internal/orchestrator never sets AsName on a network's deployed_object,
// so its identity is always just its ObjectName) -- see HostSpec.Network's
// doc comment in internal/builder/builder.go for why a host/container
// needs this at all. A synthetic db.DeployedObject is enough: only Kind
// and ObjectName feed deterministicExternalName, and both are known from
// obj.NetworkName without any extra lookup.
func networkExternalName(buildID pgtype.UUID, teamNumber int32, obj db.DeployedObject) string {
	return deterministicExternalName(buildID, teamNumber, db.DeployedObject{
		Kind: "network", ObjectName: db.StrOrEmpty(obj.NetworkName),
	})
}

// copyAddress resolves a host or container's own address -- content's
// CIDR plus its `last_octet` -- via render.Resolve, the exact same
// function internal/orchestrator's own reconcileCopy already calls to
// build a script's template context. Reusing it here (rather than
// re-deriving the network+copy lookup a second time) guarantees a
// script's `{{ .Address }}` and the address a builder actually assigns
// the host can never disagree -- they're the same call. obj.AsName is
// always set for a host/container's deployed_object (see reconcileCopy),
// so this only needs one extra query, for the build's EnvironmentName.
func (r *Runner) copyAddress(ctx context.Context, q *db.Queries, buildID pgtype.UUID, c *loader.Content, obj db.DeployedObject, teamNumber int32) (string, error) {
	build, err := q.GetBuild(ctx, buildID)
	if err != nil {
		return "", fmt.Errorf("loading build: %w", err)
	}
	rctx, err := render.Resolve(c, build.EnvironmentName, db.StrOrEmpty(obj.AsName), int(teamNumber))
	if err != nil {
		return "", fmt.Errorf("resolving address: %w", err)
	}
	return rctx.Address, nil
}

// repoRootFor resolves task's own build to a local checkout path: via
// Checkouts (its own repository/commit) when configured, or the fixed
// r.RepoRoot otherwise -- see the Runner struct's own doc comment on
// these two fields.
func (r *Runner) repoRootFor(ctx context.Context, task db.Task) (string, error) {
	if r.Checkouts != nil {
		return r.Checkouts.ForBuild(ctx, task.BuildID)
	}
	return r.RepoRoot, nil
}

// builderFor resolves task to the builder.Builder it should run against:
// via Builders (its own build's own builder_config) when configured, or
// the fixed r.Builder otherwise -- see the Runner struct's own doc
// comment on these two fields.
func (r *Runner) builderFor(ctx context.Context, task db.Task) (builder.Builder, error) {
	if r.Builders != nil {
		return r.Builders(ctx, task)
	}
	return r.Builder, nil
}

func (r *Runner) executeDeploy(ctx context.Context, q *db.Queries, task db.Task, obj db.DeployedObject, externalName, displayName string, teamNumber int32) error {
	repoRoot, err := r.repoRootFor(ctx, task)
	if err != nil {
		return fmt.Errorf("resolving content location: %w", err)
	}
	b, err := r.builderFor(ctx, task)
	if err != nil {
		return fmt.Errorf("resolving builder: %w", err)
	}
	c, err := loader.Load(repoRoot)
	if err != nil {
		return fmt.Errorf("loading content: %w", err)
	}
	team := strconv.FormatInt(int64(teamNumber), 10)

	// Real status transition, found missing by direct audit:
	// MarkDeployedObjectDeploying existed since
	// this table's own CHECK constraint named "deploying" as a real state
	// (migrations/00003), and the UI already renders it (StatusBadge,
	// DashboardCharts' own STATUS_ORDER), but nothing ever called it -- a
	// real deploy that takes minutes (a VM's first image clone, per item
	// 1's own live finding) read as "pending" for its entire duration.
	// Unconditional, not guarded by obj.Status: safe to set again on a
	// retried task that's already "deploying," and MarkDeployedObjectRunning/
	// MarkDeployedObjectDeployFailed below always supersede it once the real
	// builder call actually finishes either way.
	if _, err := q.MarkDeployedObjectDeploying(ctx, obj.ID); err != nil {
		return fmt.Errorf("recording deploying state: %w", err)
	}

	var externalRef string
	switch obj.Kind {
	case "network":
		net := findNetwork(c, obj.ObjectName)
		if net == nil {
			return fmt.Errorf("network %q not found in content", obj.ObjectName)
		}
		externalRef, err = b.DeployNetwork(ctx, builder.NetworkSpec{
			ExternalName: externalName, DisplayName: networkDisplayName(teamNumber, obj.ObjectName), Team: team, CIDR: net.CIDR,
		})
	case "host":
		h := findHost(c, obj.ObjectName)
		if h == nil {
			return fmt.Errorf("host %q not found in content", obj.ObjectName)
		}
		addr, addrErr := r.copyAddress(ctx, q, task.BuildID, c, obj, teamNumber)
		if addrErr != nil {
			return addrErr
		}
		del, derr := r.deliverAgent(ctx, q, obj, h.OS)
		if derr != nil {
			return derr
		}
		externalRef, err = b.DeployHost(ctx, builder.HostSpec{
			ExternalName: externalName, DisplayName: displayName, Team: team,
			Network: networkExternalName(task.BuildID, teamNumber, obj), NetworkDisplayName: networkDisplayName(teamNumber, db.StrOrEmpty(obj.NetworkName)), Address: addr,
			OS: h.OS, Size: h.Size, DiskGB: h.Disk, TCPPorts: h.Ports.TCP, UDPPorts: h.Ports.UDP,
			CloudInitUserData: del.UserData, CloudInitViaISO: del.Platform == agentdelivery.Windows,
		})
	case "container":
		ct := findContainer(c, obj.ObjectName)
		if ct == nil {
			return fmt.Errorf("container %q not found in content", obj.ObjectName)
		}
		addr, addrErr := r.copyAddress(ctx, q, task.BuildID, c, obj, teamNumber)
		if addrErr != nil {
			return addrErr
		}
		// A container's agent always targets Linux (it runs inside the OCI
		// container), regardless of the image; PlatformFor defaults there.
		del, derr := r.deliverAgent(ctx, q, obj, "linux")
		if derr != nil {
			return derr
		}
		// The registry credential for this image, if one is configured, so a
		// nesting builder (MicroCloud) can `docker login` before pulling a
		// private image. The builder, not a materialized agent step, owns the
		// pull now -- a container's agent runs INSIDE the app and has no Docker.
		cspec := builder.ContainerSpec{
			ExternalName: externalName, DisplayName: displayName, Team: team,
			Network: networkExternalName(task.BuildID, teamNumber, obj), NetworkDisplayName: networkDisplayName(teamNumber, db.StrOrEmpty(obj.NetworkName)), Address: addr,
			Image: ct.Image, Size: ct.Size, Env: ct.Env, Command: ct.Command,
			TCPPorts: ct.Ports.TCP, UDPPorts: ct.Ports.UDP,
			// The agent binary is planted as the app container's entrypoint on
			// every builder (native OCI: oci.entrypoint; nesting: a bind-mounted
			// --entrypoint). Cloud (Fargate/Zun): downloaded at start from its
			// URL. Each builder takes what it needs; the others are ignored.
			CloudInitUserData: del.UserData, CloudInitViaISO: del.Platform == agentdelivery.Windows,
			AgentBinary: del.Binary, AgentDownloadURL: del.DownloadURL,
		}
		// Native container-log forwarding (environment container_logs): the
		// builder applies this as its platform's log driver on the container. A
		// ComposeHost ignores it -- a compose project's logging is set as the
		// Docker daemon default on its host before `docker compose up` (gateway).
		if raw, lerr := q.GetEnvironmentContainerLogsForObject(ctx, obj.ID); lerr == nil && len(raw) > 0 {
			var cl struct {
				Driver  string            `json:"driver"`
				Options map[string]string `json:"options"`
			}
			if json.Unmarshal(raw, &cl) == nil {
				cspec.LogDriver, cspec.LogOptions = cl.Driver, cl.Options
			}
		}
		if ct.Compose != "" {
			// A Compose project: the builder provides the machine, and the
			// agent on it pulls and starts the project (registry logins
			// included) as the container's first commands.
			cspec.ComposeHost = true
			cspec.DiskGB = ct.Disk
			if cspec.DiskGB == 0 {
				cspec.DiskGB = builder.DefaultComposeDiskGB
			}
			cspec.AgentBinary, cspec.AgentDownloadURL = nil, ""
		} else if cred := r.registryCredFor(ctx, q, ct.Image); cred != nil {
			cspec.RegistryHost, cspec.RegistryUser, cspec.RegistrySecret = cred.RegistryHost, cred.Username, cred.Secret
		}
		externalRef, err = b.DeployContainer(ctx, cspec)
	default:
		return fmt.Errorf("unknown deployed_object kind %q", obj.Kind)
	}
	// Record the hoster ref even on failure: a builder returns the
	// deterministic instance name it attempted (e.g. a Windows host whose
	// config drive failed after the instance was partially created), so
	// teardown can still destroy the orphan instead of leaving it untracked.
	if externalRef != "" {
		if refErr := q.SetDeployedObjectExternalRef(ctx, db.SetDeployedObjectExternalRefParams{ID: obj.ID, ExternalRef: db.StrPtr(externalRef)}); refErr != nil {
			return fmt.Errorf("recording external ref: %w", refErr)
		}
	}
	if err != nil {
		return fmt.Errorf("builder deploy: %w", err)
	}
	if r.AfterBuilderCall != nil {
		r.AfterBuilderCall()
	}

	var payload struct {
		Fingerprint string `json:"fingerprint"`
	}
	json.Unmarshal(task.Payload, &payload)

	if _, err := q.MarkDeployedObjectRunning(ctx, db.MarkDeployedObjectRunningParams{
		ID: obj.ID, ExternalRef: db.StrPtr(externalRef), Fingerprint: payload.Fingerprint,
	}); err != nil {
		return fmt.Errorf("recording running state: %w", err)
	}

	// Authored steps are NOT queued here. The box is now deployed ahead of its
	// dependencies, but its steps must not run until those dependencies have
	// finished configuring -- so step materialization moved to the orchestrator
	// (internal/orchestrator.materializeStepsIfReady), which knows each team's
	// dependency readiness. A just-deployed object sits at "running" with no
	// agent_task rows until the orchestrator queues them.
	return nil
}

// deliverAgent, when delivery is configured, builds this object's per-object
// agent binary and the cloud-init user-data that fetches and runs it, stores the
// binary for the api to serve (keyed by a fresh one-time token), and returns the
// whole delivery. A host uses del.UserData (cloud-init); a native OCI container
// uses del.Binary directly (pushed in, agent as entrypoint -- see the incus
// builder). When delivery isn't configured it returns a zero Delivery -- the
// object deploys without an agent, exactly as before. Delivery must not fail a
// deploy that would otherwise succeed by more than it has to: a genuine
// build/patch error is returned (it means the runner is misconfigured).
func (r *Runner) deliverAgent(ctx context.Context, q *db.Queries, obj db.DeployedObject, osName string) (agentdelivery.Delivery, error) {
	if !r.Delivery.Enabled() {
		return agentdelivery.Delivery{}, nil
	}
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return agentdelivery.Delivery{}, fmt.Errorf("generating agent download token: %w", err)
	}
	token := hex.EncodeToString(tokenBytes)
	// agent-debug (environment YAML) bakes a local debug log into the binary;
	// off (the default, and on any lookup miss) means the agent is silent on the
	// box and only reports to the servers. A missing row -- no environment yet,
	// or a chain gap -- is simply "off", never a deploy failure.
	debug, err := q.GetEnvironmentAgentDebugForObject(ctx, obj.ID)
	if err != nil {
		debug = false
	}
	del, err := r.Delivery.Build(obj.ID.String(), token, osName, debug)
	if err != nil {
		return agentdelivery.Delivery{}, fmt.Errorf("building agent delivery: %w", err)
	}
	if err := q.CreateAgentArtifact(ctx, db.CreateAgentArtifactParams{
		DeployedObjectID: obj.ID, Platform: string(del.Platform), Token: token, AgentBinary: del.Binary,
	}); err != nil {
		return agentdelivery.Delivery{}, fmt.Errorf("storing agent binary: %w", err)
	}
	return del, nil
}

func (r *Runner) executeDestroy(ctx context.Context, q *db.Queries, task db.Task, obj db.DeployedObject, externalName string, teamNumber int32) (resolved bool, err error) {
	b, err := r.builderFor(ctx, task)
	if err != nil {
		return false, fmt.Errorf("resolving builder: %w", err)
	}
	ref := db.StrOrEmpty(obj.ExternalRef)
	if ref == "" {
		// Defensive: a destroy was requested for something never actually
		// deployed (e.g. removed from content before its deploy task ever
		// ran). Still call through with the deterministic name -- the
		// builder's destroy is required to be a no-op-safe "ensure" even
		// for something it never created, exactly like this.
		ref = externalName
	}
	team := strconv.FormatInt(int64(teamNumber), 10)

	// Same real status transition executeDeploy's own MarkDeployedObjectDeploying
	// call adds, for the destroy half -- see its own doc comment.
	if _, err := q.MarkDeployedObjectDestroying(ctx, obj.ID); err != nil {
		return false, fmt.Errorf("recording destroying state: %w", err)
	}

	switch obj.Kind {
	case "network":
		err = b.DestroyNetwork(ctx, team, ref)
	case "host":
		err = b.DestroyHost(ctx, team, ref)
	case "container":
		err = b.DestroyContainer(ctx, team, ref)
	default:
		return false, fmt.Errorf("unknown deployed_object kind %q", obj.Kind)
	}
	if err != nil {
		return false, fmt.Errorf("builder destroy: %w", err)
	}
	if r.AfterBuilderCall != nil {
		r.AfterBuilderCall()
	}

	var payload struct {
		Remove bool `json:"remove"`
		// Terminal is internal/orchestrator.Teardown's own signal: this
		// destroy is a deliberate build teardown, not a content-driven
		// removal (Remove) or a fingerprint-change rebuild (neither set)
		// -- the object stays destroyed, not deleted and not reset for a
		// redeploy that a torn-down build will never ask for again.
		Terminal bool `json:"terminal"`
	}
	json.Unmarshal(task.Payload, &payload)

	if payload.Remove {
		if err := q.DeleteDeployedObject(ctx, obj.ID); err != nil {
			return false, fmt.Errorf("deleting removed object: %w", err)
		}
		return true, nil
	}
	if payload.Terminal {
		if _, err := q.MarkDeployedObjectDestroyed(ctx, obj.ID); err != nil {
			return false, fmt.Errorf("recording destroyed state: %w", err)
		}
		return false, nil
	}
	if _, err := q.ResetDeployedObjectForRedeploy(ctx, obj.ID); err != nil {
		return false, fmt.Errorf("resetting object for redeploy: %w", err)
	}
	// Clear this object's materialized steps so the redeploy re-materializes
	// them. materializeSteps skips when any agent_task already exists
	// (NextStepIndexForHost != 0), so without this a rebuilt instance would come
	// up but never re-run its scripts/users/services/validators. Applies to both
	// a content-change redeploy (fingerprint diff) and a forced rebuild
	// (orchestrator.RebuildObjects); validator_result rows cascade.
	if err := q.DeleteAgentTasksForObject(ctx, obj.ID); err != nil {
		return false, fmt.Errorf("clearing steps for redeploy: %w", err)
	}
	return false, nil
}

// executeAccess runs the real OpenAccess/CloseAccess builder call
// (proven live against a real Incus daemon) and then --
// a real fix found by direct audit -- records what it actually achieved:
// SetTeamAccessState's own doc comment already described this exact
// wiring ("written once the actual OpenAccess/CloseAccess builder call
// completes... never optimistically before"), but nothing ever called
// it. Before this, an admin's Close click genuinely closed the network
// at the hoster, but team.access_state (what BuildAccess.tsx and
// BuildOverview.tsx read to show an operator whether it worked) stayed
// "open" forever -- no way to confirm a close actually took effect.
func (r *Runner) executeAccess(ctx context.Context, q *db.Queries, task db.Task) error {
	b, err := r.builderFor(ctx, task)
	if err != nil {
		return fmt.Errorf("resolving builder: %w", err)
	}
	var payload struct {
		Team string `json:"team"`
	}
	json.Unmarshal(task.Payload, &payload)

	if task.Kind == "open_access" {
		if err := b.OpenAccess(ctx, payload.Team); err != nil {
			return err
		}
	} else {
		if err := b.CloseAccess(ctx, payload.Team); err != nil {
			return err
		}
	}

	teamNum, err := strconv.ParseInt(payload.Team, 10, 32)
	if err != nil {
		return fmt.Errorf("access task's own team payload %q isn't a real team number: %w", payload.Team, err)
	}
	team, err := q.GetTeamByNumber(ctx, db.GetTeamByNumberParams{BuildID: task.BuildID, TeamNumber: int32(teamNum)})
	if err != nil {
		return fmt.Errorf("loading team %s: %w", payload.Team, err)
	}
	state, eventKind := "closed", "access.closed"
	if task.Kind == "open_access" {
		state, eventKind = "open", "access.opened"
	}
	if _, err := q.SetTeamAccessState(ctx, db.SetTeamAccessStateParams{ID: team.ID, AccessState: state}); err != nil {
		return fmt.Errorf("recording access state: %w", err)
	}
	r.logEvent(ctx, q, task.BuildID, task.ID, eventKind, "team "+payload.Team)
	return nil
}

// executeConfigureNetworkAccess builds the team's full set of NetworkAccess
// specs from content -- every one of the team's deployed networks, with its CIDR
// and its visible_from resolved to the same deterministic ExternalNames the
// networks were deployed under -- and hands them to the builder to enforce (OVN
// peering + ACLs on Incus/MicroCloud). The orchestrator only enqueues this once
// the team's networks are all up (see orchestrator.ReconcileNetworkAccess), so
// every network the peering mesh needs already exists.
func (r *Runner) executeConfigureNetworkAccess(ctx context.Context, q *db.Queries, task db.Task) error {
	b, err := r.builderFor(ctx, task)
	if err != nil {
		return fmt.Errorf("resolving builder: %w", err)
	}
	var payload struct {
		Team string `json:"team"`
	}
	json.Unmarshal(task.Payload, &payload)
	teamNum, err := strconv.ParseInt(payload.Team, 10, 32)
	if err != nil {
		return fmt.Errorf("configure_network_access team payload %q isn't a real team number: %w", payload.Team, err)
	}
	team, err := q.GetTeamByNumber(ctx, db.GetTeamByNumberParams{BuildID: task.BuildID, TeamNumber: int32(teamNum)})
	if err != nil {
		return fmt.Errorf("loading team %s: %w", payload.Team, err)
	}
	objs, err := q.ListDeployedObjectsByTeam(ctx, team.ID)
	if err != nil {
		return fmt.Errorf("listing team %s objects: %w", payload.Team, err)
	}
	repoRoot, err := r.repoRootFor(ctx, task)
	if err != nil {
		return fmt.Errorf("resolving checkout: %w", err)
	}
	c, err := loader.Load(repoRoot)
	if err != nil {
		return fmt.Errorf("loading content: %w", err)
	}

	build, err := q.GetBuild(ctx, task.BuildID)
	if err != nil {
		return fmt.Errorf("loading build for team %s: %w", payload.Team, err)
	}

	// Per-network host firewall: each host/container on a network, with its IP
	// (resolved the same way a deploy computes it) and its declared ports, so
	// the builder can enforce `ports:` as an ingress allowlist. A host with no
	// ports contributes an entry with empty lists -> reachable on nothing.
	hostsByNet := make(map[string][]builder.HostAccess)
	for _, o := range objs {
		if o.Kind != "host" && o.Kind != "container" {
			continue
		}
		netName := db.StrOrEmpty(o.NetworkName)
		if netName == "" {
			continue
		}
		var tcp, udp []string
		if h := findHost(c, o.ObjectName); h != nil {
			tcp, udp = h.Ports.TCP, h.Ports.UDP
		} else if ct := findContainer(c, o.ObjectName); ct != nil {
			tcp, udp = ct.Ports.TCP, ct.Ports.UDP
		} else {
			continue // removed from content since deploy
		}
		rctx, err := render.Resolve(c, build.EnvironmentName, db.StrOrEmpty(o.AsName), int(teamNum))
		if err != nil {
			return fmt.Errorf("resolving address for %s: %w", db.StrOrEmpty(o.AsName), err)
		}
		hostsByNet[netName] = append(hostsByNet[netName], builder.HostAccess{
			Address: rctx.Address, ExternalRef: db.StrOrEmpty(o.ExternalRef), TCPPorts: tcp, UDPPorts: udp,
		})
	}

	var access []builder.NetworkAccess
	for _, o := range objs {
		if o.Kind != "network" {
			continue
		}
		net := findNetwork(c, o.ObjectName)
		if net == nil {
			continue // removed from content since deploy; skip
		}
		var visibleFrom []string
		for _, name := range net.VisibleFrom {
			visibleFrom = append(visibleFrom, networkExternalNameByName(task.BuildID, int32(teamNum), name))
		}
		access = append(access, builder.NetworkAccess{
			ExternalName: networkExternalNameByName(task.BuildID, int32(teamNum), o.ObjectName),
			DisplayName:  networkDisplayName(int32(teamNum), o.ObjectName),
			CIDR:         net.CIDR,
			VisibleFrom:  visibleFrom,
			Hosts:        hostsByNet[o.ObjectName],
		})
	}
	if err := b.ConfigureNetworkAccess(ctx, payload.Team, access); err != nil {
		return fmt.Errorf("configuring network access for team %s: %w", payload.Team, err)
	}
	r.logEvent(ctx, q, task.BuildID, task.ID, "network_access.configured", fmt.Sprintf("team %s: %d network(s)", payload.Team, len(access)))
	return nil
}

// executeConfigureExternalAccess realizes a team's content `public:` ports: it
// gathers every deployed host/container whose topology copy declares public
// ports, hands them to the builder to make externally reachable (a public IP
// per host on AWS; a port-NAT on a shared uplink IP on Incus/MicroCloud), and
// records the external endpoints the builder assigns in external_access for the
// UI/CLI to surface. `public:` is declared per topology COPY, so it's keyed by
// the copy's as-name here, not the host definition.
func (r *Runner) executeConfigureExternalAccess(ctx context.Context, q *db.Queries, task db.Task) error {
	b, err := r.builderFor(ctx, task)
	if err != nil {
		return fmt.Errorf("resolving builder: %w", err)
	}
	var payload struct {
		Team string `json:"team"`
	}
	json.Unmarshal(task.Payload, &payload)
	teamNum, err := strconv.ParseInt(payload.Team, 10, 32)
	if err != nil {
		return fmt.Errorf("configure_external_access team payload %q isn't a real team number: %w", payload.Team, err)
	}
	team, err := q.GetTeamByNumber(ctx, db.GetTeamByNumberParams{BuildID: task.BuildID, TeamNumber: int32(teamNum)})
	if err != nil {
		return fmt.Errorf("loading team %s: %w", payload.Team, err)
	}
	objs, err := q.ListDeployedObjectsByTeam(ctx, team.ID)
	if err != nil {
		return fmt.Errorf("listing team %s objects: %w", payload.Team, err)
	}
	repoRoot, err := r.repoRootFor(ctx, task)
	if err != nil {
		return fmt.Errorf("resolving checkout: %w", err)
	}
	c, err := loader.Load(repoRoot)
	if err != nil {
		return fmt.Errorf("loading content: %w", err)
	}
	build, err := q.GetBuild(ctx, task.BuildID)
	if err != nil {
		return fmt.Errorf("loading build for team %s: %w", payload.Team, err)
	}
	refToObjID := map[string]pgtype.UUID{}
	var hosts []builder.ExternalHost
	var touched []pgtype.UUID
	for _, o := range objs {
		if o.Kind != "host" && o.Kind != "container" {
			continue
		}
		// `public:` lives on the host/container definition, so read it from there
		// by object name (not the per-copy placement).
		var pub *loader.Ports
		if h := findHost(c, o.ObjectName); h != nil {
			pub = h.Public
		} else if ct := findContainer(c, o.ObjectName); ct != nil {
			pub = ct.Public
		}
		if pub == nil || (len(pub.TCP) == 0 && len(pub.UDP) == 0) {
			continue
		}
		ref := db.StrOrEmpty(o.ExternalRef)
		if ref == "" {
			continue // not deployed at the hoster yet
		}
		rctx, err := render.Resolve(c, build.EnvironmentName, db.StrOrEmpty(o.AsName), int(teamNum))
		if err != nil {
			return fmt.Errorf("resolving address for %s: %w", db.StrOrEmpty(o.AsName), err)
		}
		hosts = append(hosts, builder.ExternalHost{ExternalRef: ref, Address: rctx.Address, TCPPorts: pub.TCP, UDPPorts: pub.UDP})
		refToObjID[ref] = o.ID
		touched = append(touched, o.ID)
	}
	if len(hosts) == 0 {
		return nil // nothing public in this team
	}

	endpoints, err := b.ConfigureExternalAccess(ctx, payload.Team, hosts)
	if err != nil {
		return fmt.Errorf("configuring external access for team %s: %w", payload.Team, err)
	}

	// Replace every touched object's recorded endpoints, then write what the
	// builder returned -- so a removed public port leaves no stale row behind.
	for _, id := range touched {
		if err := q.DeleteExternalAccessForObject(ctx, id); err != nil {
			return fmt.Errorf("clearing external access rows: %w", err)
		}
	}
	for _, e := range endpoints {
		id, ok := refToObjID[e.ExternalRef]
		if !ok {
			continue
		}
		if err := q.UpsertExternalAccess(ctx, db.UpsertExternalAccessParams{
			DeployedObjectID: id, Protocol: e.Protocol, InternalPort: e.InternalPort,
			PublicAddress: e.PublicAddress, ExternalPort: e.ExternalPort,
		}); err != nil {
			return fmt.Errorf("recording external access endpoint: %w", err)
		}
	}
	r.logEvent(ctx, q, task.BuildID, task.ID, "external_access.configured", fmt.Sprintf("team %s: %d endpoint(s)", payload.Team, len(endpoints)))
	return nil
}

// networkExternalNameByName is networkExternalName for a network addressed by its
// content name (rather than via a host's obj.NetworkName) -- the same
// deterministic ExternalName the network's own deploy used, needed to name both a
// network and each of its visible_from siblings for ConfigureNetworkAccess.
func networkExternalNameByName(buildID pgtype.UUID, teamNumber int32, name string) string {
	return deterministicExternalName(buildID, teamNumber, db.DeployedObject{Kind: "network", ObjectName: name})
}

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
