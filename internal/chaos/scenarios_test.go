package chaos

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/globalcptc/laforge/internal/db"
	"github.com/globalcptc/laforge/internal/orchestrator"
)

// identityOf mirrors internal/runner's own deterministicExternalName
// identity rule exactly (as_name if set, else object_name) -- duplicated
// here rather than exported from internal/runner because this is test
// code computing the SAME value independently, as a check, not reusing
// production logic that could hide a mismatch between the two.
func identityOf(obj db.DeployedObject) string {
	if as := db.StrOrEmpty(obj.AsName); as != "" {
		return as
	}
	return obj.ObjectName
}

const wantObjects = 65 // 5 teams x 13 objects each, matching examples/lm-test -- see internal/orchestrator's own test for the breakdown

// wantFirstPassTasks is how many deploy tasks a single reconcile creates: 55,
// because depends_on ordering holds 2 objects/team (webserver -> database,
// workstation -> domain-controller) until their dependencies deploy.
const wantFirstPassTasks = 55

// convergeRemaining interleaves reconcile + a healthy runner until every object
// is running. A single reconcile can't create every deploy task (depends_on
// ordering holds dependents back), so after the first wave of a chaos scenario
// deploys the independent objects, this finishes the dependents. In-process
// Reconcile is fine here -- the crash scenarios use the orchestrator binary;
// this is just the completion phase.
func convergeRemaining(t *testing.T, pool *pgxpool.Pool, buildID pgtype.UUID, runnerPrefix string) {
	t.Helper()
	ctx := context.Background()
	q := db.New(pool)
	for pass := 0; pass < 10; pass++ {
		objs, err := q.ListDeployedObjectsByBuild(ctx, buildID)
		if err != nil {
			t.Fatalf("ListDeployedObjectsByBuild: %v", err)
		}
		allRunning := len(objs) > 0
		for _, o := range objs {
			if o.Status != "running" {
				allRunning = false
				break
			}
		}
		if allRunning {
			return
		}
		if err := orchestrator.Reconcile(ctx, pool, contentDirAbs, buildID); err != nil {
			t.Fatalf("Reconcile (converge pass %d): %v", pass, err)
		}
		runHealthyRunnerToCompletion(t, buildID.String(), fmt.Sprintf("%s-%d", runnerPrefix, pass), 0)
	}
	t.Fatalf("build did not converge within 10 passes")
}

var leasedRe = regexp.MustCompile(`LEASED (\S+) kind=`)

func parseLeasedTaskID(t *testing.T, logPath string) string {
	t.Helper()
	b, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("reading %s: %v", logPath, err)
	}
	m := leasedRe.FindStringSubmatch(string(b))
	if m == nil {
		t.Fatalf("no LEASED line found in %s:\n%s", logPath, b)
	}
	return m[1]
}

func assertFullConvergence(t *testing.T, pool *pgxpool.Pool, buildID pgtype.UUID, want int) {
	t.Helper()
	ctx := context.Background()
	q := db.New(pool)

	objs, err := q.ListDeployedObjectsByBuild(ctx, buildID)
	if err != nil {
		t.Fatalf("ListDeployedObjectsByBuild: %v", err)
	}
	if len(objs) != want {
		t.Fatalf("len(objs) = %d, want %d", len(objs), want)
	}
	seenRefs := map[string]bool{}
	for _, o := range objs {
		// The runner alone (no orchestrator lifecycle poll in this test)
		// drives a successful deploy to "running" -- infra up, before any
		// agent check-in advances it toward building/finished.
		if o.Status != "running" {
			t.Errorf("object %s (%s) status = %q, want running", o.ObjectName, db.StrOrEmpty(o.AsName), o.Status)
			continue
		}
		ref := db.StrOrEmpty(o.ExternalRef)
		if ref == "" {
			t.Errorf("object %s (%s) has no external_ref after deploy", o.ObjectName, db.StrOrEmpty(o.AsName))
			continue
		}
		if seenRefs[ref] {
			t.Errorf("duplicate external_ref %q across deployed objects -- convergence failed", ref)
		}
		seenRefs[ref] = true
		count, err := q.CountFakeHosterResourcesByRef(ctx, ref)
		if err != nil {
			t.Fatalf("CountFakeHosterResourcesByRef(%s): %v", ref, err)
		}
		if count != 1 {
			t.Errorf("fake_hoster_resource rows for %q = %d, want exactly 1 (chaos must not create duplicates)", ref, count)
		}
	}
	if len(seenRefs) != want {
		t.Fatalf("saw %d distinct external_refs, want %d", len(seenRefs), want)
	}

	tasks, err := q.ListTasksByBuild(ctx, buildID)
	if err != nil {
		t.Fatalf("ListTasksByBuild: %v", err)
	}
	for _, tk := range tasks {
		if tk.Status == "pending" || tk.Status == "leased" {
			t.Errorf("unexpected open task after full convergence: %s %s", tk.Kind, tk.Status)
		}
	}
}

// TestChaos_RunnerKilledZeroWorkDone is scenario A, against the real thing: a real
// laforge-runner subprocess is SIGKILLed immediately after leasing a
// task, before doing any work. Proves the task is reclaimed and the WHOLE
// 65-object build still reaches full convergence once a healthy runner
// takes over -- not just that one task recovers.
func TestChaos_RunnerKilledZeroWorkDone(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	build := newTestBuild(t, pool, "chaos-zero-work")

	if err := orchestrator.Reconcile(ctx, pool, contentDirAbs, build.ID); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}

	cmdA, logA := runnerCmd(t, build.ID.String(), "runner-A", "-crash-after-lease", "1")
	if err := cmdA.Start(); err != nil {
		t.Fatalf("starting runner-A: %v", err)
	}
	doneA := make(chan error, 1)
	go func() { doneA <- cmdA.Wait() }()

	if !waitForLogLine(t, logA, "LEASED", 5*time.Second) {
		t.Fatal("runner-A never leased a task before crashing")
	}
	select {
	case <-doneA:
	case <-time.After(3 * time.Second):
		cmdA.Process.Kill()
		t.Fatal("runner-A did not crash (exit) in time after leasing")
	}

	taskID := parseLeasedTaskID(t, logA)

	var status string
	var owner *string
	if err := pool.QueryRow(ctx, "SELECT status, lease_owner FROM task WHERE id = $1", taskID).Scan(&status, &owner); err != nil {
		t.Fatalf("querying task after kill: %v", err)
	}
	if status != "leased" || owner == nil || *owner != "runner-A" {
		t.Fatalf("immediately after SIGKILL: status=%s owner=%v, want leased/runner-A", status, owner)
	}

	time.Sleep(4 * time.Second) // lease-seconds=3 in runnerCmd

	var reclaimable int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM task WHERE id=$1 AND status='leased' AND lease_expires_at < now()", taskID).Scan(&reclaimable); err != nil {
		t.Fatalf("checking reclaimability: %v", err)
	}
	if reclaimable != 1 {
		t.Fatalf("task %s is not reclaimable after its lease expired", taskID)
	}

	runHealthyRunnerToCompletion(t, build.ID.String(), "runner-B", 0)

	var finalOwner *string
	pool.QueryRow(ctx, "SELECT lease_owner FROM task WHERE id = $1", taskID).Scan(&finalOwner)
	if finalOwner == nil || *finalOwner != "runner-B" {
		t.Fatalf("expected task %s to end up leased by runner-B, got %v", taskID, finalOwner)
	}
	convergeRemaining(t, pool, build.ID, "runner-B-converge")
	assertFullConvergence(t, pool, build.ID, wantObjects)
}

// TestChaos_RunnerKilledAfterRealWorkBeforeRecording is scenario B: the
// runner calls the (idempotent) builder -- real work happens, a real
// fake_hoster_resource row is created -- and is SIGKILLed before it can
// record success. Proves the retry converges: the builder is called
// again, but exactly one resource exists at the end, not two.
func TestChaos_RunnerKilledAfterRealWorkBeforeRecording(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	q := db.New(pool)
	build := newTestBuild(t, pool, "chaos-partial-success")

	if err := orchestrator.Reconcile(ctx, pool, contentDirAbs, build.ID); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}

	cmdA, logA := runnerCmd(t, build.ID.String(), "runner-A", "-crash-after-builder-call", "1")
	if err := cmdA.Start(); err != nil {
		t.Fatalf("starting runner-A: %v", err)
	}
	doneA := make(chan error, 1)
	go func() { doneA <- cmdA.Wait() }()

	if !waitForLogLine(t, logA, "BUILDER-CALL-DONE", 5*time.Second) {
		t.Fatal("runner-A never completed a builder call before crashing")
	}
	select {
	case <-doneA:
	case <-time.After(3 * time.Second):
		cmdA.Process.Kill()
		t.Fatal("runner-A did not crash (exit) in time after its builder call")
	}

	taskID := parseLeasedTaskID(t, logA)
	var taskUUID pgtype.UUID
	if err := pool.QueryRow(ctx, "SELECT id FROM task WHERE id = $1", taskID).Scan(&taskUUID); err != nil {
		t.Fatalf("re-reading task id: %v", err)
	}
	task, err := q.GetTask(ctx, taskUUID)
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}
	if task.Status != "leased" {
		t.Fatalf("task status = %q immediately after the crash, want leased (never got to CompleteTask)", task.Status)
	}
	obj, err := q.GetDeployedObject(ctx, task.DeployedObjectID)
	if err != nil {
		t.Fatalf("GetDeployedObject: %v", err)
	}
	if obj.Status == "running" {
		t.Fatalf("object %s already shows running -- MarkDeployedObjectRunning must not have run before the crash point", obj.ObjectName)
	}

	// The real, durable proof that work actually happened: a
	// fake_hoster_resource row exists, ensure_count=1, even though the
	// runner that created it is dead and the task/object don't know
	// about it yet.
	team, err := q.GetTeam(ctx, obj.TeamID)
	if err != nil {
		t.Fatalf("GetTeam: %v", err)
	}
	ref := fmt.Sprintf("build-%s-team-%d-%s-%s", build.ID.String(), team.TeamNumber, obj.Kind, identityOf(obj))
	res, err := q.GetFakeHosterResource(ctx, ref)
	if err != nil {
		t.Fatalf("GetFakeHosterResource(%s): %v (the builder call should have persisted this despite the crash)", ref, err)
	}
	if res.EnsureCount != 1 {
		t.Fatalf("EnsureCount = %d, want 1 (called once, by runner-A, before it crashed)", res.EnsureCount)
	}

	time.Sleep(4 * time.Second) // past lease-seconds=3

	runHealthyRunnerToCompletion(t, build.ID.String(), "runner-B", 0)

	resAfter, err := q.GetFakeHosterResource(ctx, ref)
	if err != nil {
		t.Fatalf("GetFakeHosterResource(%s) after retry: %v", ref, err)
	}
	if resAfter.EnsureCount != 2 {
		t.Fatalf("EnsureCount after retry = %d, want 2 (runner-A's call, then runner-B's retry) -- proves the retry actually re-ran the work", resAfter.EnsureCount)
	}
	countRows, err := q.CountFakeHosterResourcesByRef(ctx, ref)
	if err != nil {
		t.Fatalf("CountFakeHosterResourcesByRef: %v", err)
	}
	if countRows != 1 {
		t.Fatalf("fake_hoster_resource rows for %q = %d, want exactly 1 despite two ensure calls -- this is the whole point", ref, countRows)
	}

	convergeRemaining(t, pool, build.ID, "runner-real-work-converge")
	assertFullConvergence(t, pool, build.ID, wantObjects)
}

// TestChaos_OrchestratorKilledMidReconcile SIGKILLs a real
// laforge-orchestrator subprocess with no coordination point at all --
// killed as soon as the OS schedules it, which is a genuinely
// uncontrolled, realistic kill (it may land before Reconcile starts, mid
// content-load, mid one team's writes, or after -- that's the point: it
// doesn't matter where). A second, healthy orchestrator run afterward
// must still converge to exactly the right desired state, with no
// duplicate deployed_object or task rows from the interrupted first
// attempt.
func TestChaos_OrchestratorKilledMidReconcile(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	q := db.New(pool)
	build := newTestBuild(t, pool, "chaos-orchestrator-kill")

	logPath := t.TempDir() + "/orchestrator-A.log"
	logFile, err := os.Create(logPath)
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(orchestratorBin, "-db-url", testDBURL, "-repo", contentDirAbs, "-once")
	cmd.Stdout, cmd.Stderr = logFile, logFile
	if err := cmd.Start(); err != nil {
		t.Fatalf("starting orchestrator: %v", err)
	}
	// Deliberately no wait, no synchronization: kill it the instant the
	// OS has scheduled it at all. Real process start + DB connect +
	// loading the whole content repo (loader.Load walks every file) is
	// enough real work that this reliably lands somewhere inside that
	// window, not provably before or after it.
	cmd.Process.Kill()
	cmd.Wait()
	logFile.Close()

	// Whatever partial state resulted, a fresh orchestrator pass must
	// converge cleanly -- no duplicates from the interrupted attempt.
	cmd2 := exec.Command(orchestratorBin, "-db-url", testDBURL, "-repo", contentDirAbs, "-once")
	out2, err := cmd2.CombinedOutput()
	if err != nil {
		t.Fatalf("second orchestrator run failed: %v\n%s", err, out2)
	}

	objs, err := q.ListDeployedObjectsByBuild(ctx, build.ID)
	if err != nil {
		t.Fatalf("ListDeployedObjectsByBuild: %v", err)
	}
	if len(objs) != wantObjects {
		t.Fatalf("after kill + fresh reconcile: %d deployed_object rows, want %d (a killed-and-restarted reconcile must not duplicate rows)", len(objs), wantObjects)
	}
	teams, err := q.ListTeamsByBuild(ctx, build.ID)
	if err != nil {
		t.Fatalf("ListTeamsByBuild: %v", err)
	}
	if len(teams) != 5 {
		t.Fatalf("teams = %d, want 5 (no duplicate team rows either)", len(teams))
	}
	tasks, err := q.ListTasksByBuild(ctx, build.ID)
	if err != nil {
		t.Fatalf("ListTasksByBuild: %v", err)
	}
	if len(tasks) != wantFirstPassTasks {
		t.Fatalf("tasks = %d, want %d (one open task per ready object; depends_on holds 2/team, no duplicates from the interrupted first pass)", len(tasks), wantFirstPassTasks)
	}

	// Prove the end state is genuinely usable, not just "no duplicate
	// rows": a healthy runner can complete the whole build from here
	// (converging the depends_on-gated objects over further passes).
	runHealthyRunnerToCompletion(t, build.ID.String(), "runner-after-orchestrator-kill", 0)
	convergeRemaining(t, pool, build.ID, "runner-after-orchestrator-kill-converge")
	assertFullConvergence(t, pool, build.ID, wantObjects)
}

// TestChaos_AllRunnersKilledMidBuild is "nothing durable on a runner: kill
// every runner mid-build, start fresh ones, and the build completes with
// all output, logs, and artifacts intact." Three real runner subprocesses,
// each with real per-call latency (so several tasks are genuinely
// in-flight, not just queued), are SIGKILLed simultaneously partway
// through a 65-object build. Two fresh runners then finish it,
// concurrently with each other -- proving SKIP LOCKED keeps them from
// double-claiming anything even under real OS-level concurrency, not just
// in a single process's goroutines.
func TestChaos_AllRunnersKilledMidBuild(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	q := db.New(pool)
	build := newTestBuild(t, pool, "chaos-all-killed")

	if err := orchestrator.Reconcile(ctx, pool, contentDirAbs, build.ID); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}

	var firstWave []*exec.Cmd
	for i := 1; i <= 3; i++ {
		cmd, _ := runnerCmd(t, build.ID.String(), fmt.Sprintf("runner-wave1-%d", i), "-work-delay-ms", "150")
		if err := cmd.Start(); err != nil {
			t.Fatalf("starting first-wave runner %d: %v", i, err)
		}
		firstWave = append(firstWave, cmd)
	}

	time.Sleep(500 * time.Millisecond) // let real, genuinely in-flight work happen

	for i, cmd := range firstWave {
		if err := cmd.Process.Kill(); err != nil {
			t.Logf("killing first-wave runner %d: %v", i+1, err)
		}
	}
	for _, cmd := range firstWave {
		cmd.Wait()
	}

	// Whatever got done, got done for real -- assert no duplicates exist
	// among it before any recovery happens, same as the single-runner
	// scenarios above.
	objsMidway, err := q.ListDeployedObjectsByBuild(ctx, build.ID)
	if err != nil {
		t.Fatalf("ListDeployedObjectsByBuild (midway): %v", err)
	}
	seen := map[string]bool{}
	deployedSoFar := 0
	for _, o := range objsMidway {
		if o.Status != "running" {
			continue
		}
		deployedSoFar++
		ref := db.StrOrEmpty(o.ExternalRef)
		if seen[ref] {
			t.Fatalf("duplicate external_ref %q already present after killing the first wave", ref)
		}
		seen[ref] = true
	}
	t.Logf("first wave (3 runners, killed after 500ms): %d/%d objects deployed before the kill", deployedSoFar, wantObjects)

	// Two fresh runners, concurrently, finish the rest.
	var secondWave []*exec.Cmd
	for i := 1; i <= 2; i++ {
		cmd, _ := runnerCmd(t, build.ID.String(), fmt.Sprintf("runner-wave2-%d", i))
		if err := cmd.Start(); err != nil {
			t.Fatalf("starting second-wave runner %d: %v", i, err)
		}
		secondWave = append(secondWave, cmd)
	}
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		var n int
		if err := pool.QueryRow(ctx, "SELECT count(*) FROM task WHERE build_id = $1 AND status IN ('pending','leased')", build.ID).Scan(&n); err == nil && n == 0 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	for _, cmd := range secondWave {
		cmd.Process.Kill()
		cmd.Wait()
	}

	// The waves deployed the independent objects; finish the depends_on-gated
	// dependents (which get tasks only after their dependencies are up).
	convergeRemaining(t, pool, build.ID, "runner-all-killed-converge")
	assertFullConvergence(t, pool, build.ID, wantObjects)

	// "all output, logs, and artifacts intact": the event journal
	// (written to Postgres on every task outcome, never buffered only in
	// a runner's own memory) has real entries despite three runners
	// having been killed outright.
	events, err := q.ListEventsByBuild(ctx, build.ID)
	if err != nil {
		t.Fatalf("ListEventsByBuild: %v", err)
	}
	if len(events) == 0 {
		t.Fatal("expected event journal entries to survive despite runners being killed -- got none")
	}
	t.Logf("event journal has %d entries after the full chaos run", len(events))
}
