// Package chaos is the required proof: "Chaos suite (must pass
// before the first real build): SIGKILL orchestrator/runner during a
// multi-team fake build; expect zero duplicate resources and full
// convergence. Run it again with runners killed on a timer, standing in
// for preemption." Every test here spawns the REAL compiled
// cmd/laforge-runner and cmd/laforge-orchestrator binaries as real OS
// subprocesses and sends real SIGKILL (os.Process.Kill()) -- not a
// simulated crash, not a cancelled goroutine. This is the same pattern
// proved by hand earlier, automated and
// run against the real schema and the real orchestrator/runner/fake
// builder.
package chaos

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/globalcptc/laforge/internal/db"
)

var (
	runnerBin       string
	orchestratorBin string
	repoRootAbs     string
	contentDirAbs   string
	testDBURL       string
)

func TestMain(m *testing.M) {
	code := func() int {
		wd, err := os.Getwd() // .../laforge/internal/chaos
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		repoRootAbs = filepath.Join(wd, "..", "..")
		contentDirAbs = filepath.Join(repoRootAbs, "examples", "lm-test")

		testDBURL = os.Getenv("LAFORGE_TEST_DATABASE_URL")
		if testDBURL == "" {
			if _, err := os.Stat("/tmp/.s.PGSQL.5432"); err != nil {
				fmt.Println("no local Postgres available, skipping chaos suite (set LAFORGE_TEST_DATABASE_URL)")
				return 0
			}
			testDBURL = "host=/tmp port=5432 user=lucas dbname=laforge_dev sslmode=disable"
		}

		tmp, err := os.MkdirTemp("", "laforge-chaos-bin-*")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		defer os.RemoveAll(tmp)

		runnerBin = filepath.Join(tmp, "laforge-runner")
		orchestratorBin = filepath.Join(tmp, "laforge-orchestrator")

		if err := buildBinary(runnerBin, "./cmd/laforge-runner"); err != nil {
			fmt.Fprintln(os.Stderr, "building laforge-runner:", err)
			return 1
		}
		if err := buildBinary(orchestratorBin, "./cmd/laforge-orchestrator"); err != nil {
			fmt.Fprintln(os.Stderr, "building laforge-orchestrator:", err)
			return 1
		}

		return m.Run()
	}()
	os.Exit(code)
}

func buildBinary(out, pkg string) error {
	cmd := exec.Command("go", "build", "-o", out, pkg)
	cmd.Dir = repoRootAbs
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	if testDBURL == "" {
		t.Skip("no local Postgres available")
	}
	_, pool, err := db.Open(context.Background(), testDBURL)
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func newTestBuild(t *testing.T, pool *pgxpool.Pool, name string) db.Build {
	t.Helper()
	ctx := context.Background()
	q := db.New(pool)
	repo, err := q.CreateRepository(ctx, db.CreateRepositoryParams{GithubOwner: "laforge-chaos", GithubRepo: name})
	if err != nil {
		t.Fatalf("CreateRepository: %v", err)
	}
	t.Cleanup(func() { pool.Exec(context.Background(), "DELETE FROM repository WHERE id = $1", repo.ID) })
	rev, err := q.CreateContentRevision(ctx, db.CreateContentRevisionParams{RepositoryID: repo.ID, CommitSha: "chaos-test-sha"})
	if err != nil {
		t.Fatalf("CreateContentRevision: %v", err)
	}
	build, err := q.CreateBuild(ctx, db.CreateBuildParams{ContentRevisionID: rev.ID, EnvironmentName: "lm-test"})
	if err != nil {
		t.Fatalf("CreateBuild: %v", err)
	}
	// This suite exercises real deploy-task creation and convergence
	// under SIGKILL, which only happens for a build in "deploying" -- see
	// internal/orchestrator.Reconcile's deployTasks gate:
	// a `planned` build now correctly
	// creates no tasks at all, so a build left at the schema default
	// here would make every scenario below a no-op, not a chaos test.
	if _, err := q.SetBuildStatus(ctx, db.SetBuildStatusParams{ID: build.ID, Status: "deploying"}); err != nil {
		t.Fatalf("SetBuildStatus: %v", err)
	}
	build.Status = "deploying"
	// fake_hoster_resource has no foreign key back to build (deliberately
	// -- see migrations/00003: it outlives any single build, modeling a
	// real hoster's own persistence), so nothing about deleting the
	// repository above cleans up whatever this test deployed into it.
	// external_ref is always "build-<this build's id>-...", so this
	// scopes cleanup to exactly this test's own rows.
	t.Cleanup(func() {
		pool.Exec(context.Background(), "DELETE FROM fake_hoster_resource WHERE external_ref LIKE $1", "build-"+build.ID.String()+"-%")
	})
	return build
}

// runnerCmd builds an *exec.Cmd for a real laforge-runner subprocess,
// streaming its output to a file so assertions can grep real process
// output, not just query Postgres.
// runnerCmd builds a real laforge-runner subprocess scoped to buildID via
// -build-id -- test isolation only, see runner.Runner.BuildID's doc
// comment: several packages' tests share one Postgres database and run in
// parallel, so an unscoped runner here could lease and complete a
// different test's task.
func runnerCmd(t *testing.T, buildID, id string, extraArgs ...string) (*exec.Cmd, string) {
	t.Helper()
	logPath := filepath.Join(t.TempDir(), "runner-"+id+".log")
	logFile, err := os.Create(logPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { logFile.Close() })

	args := append([]string{
		"-db-url", testDBURL, "-repo", contentDirAbs, "-id", id, "-build-id", buildID,
		"-lease-seconds", "3", "-heartbeat-seconds", "1", "-poll-interval", "50ms",
	}, extraArgs...)
	cmd := exec.Command(runnerBin, args...)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	return cmd, logPath
}

func waitForLogLine(t *testing.T, logPath, substr string, timeout time.Duration) bool {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		b, _ := os.ReadFile(logPath)
		if bytesContains(b, substr) {
			return true
		}
		time.Sleep(20 * time.Millisecond)
	}
	return false
}

func bytesContains(b []byte, substr string) bool {
	return len(b) > 0 && (func() bool {
		s := string(b)
		for i := 0; i+len(substr) <= len(s); i++ {
			if s[i:i+len(substr)] == substr {
				return true
			}
		}
		return false
	})()
}

// runHealthyRunnerToCompletion runs a single, un-crashing laforge-runner
// subprocess, scoped to buildID, until that build's own queue is empty,
// then stops it.
func runHealthyRunnerToCompletion(t *testing.T, buildID, id string, workDelayMS int) {
	t.Helper()
	cmd, logPath := runnerCmd(t, buildID, id, "-work-delay-ms", fmt.Sprintf("%d", workDelayMS))
	if err := cmd.Start(); err != nil {
		t.Fatalf("starting runner %s: %v", id, err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	// laforge-runner with no -max-tasks loops forever polling an empty
	// queue, so it has to be stopped once the build is actually done --
	// polling Postgres directly for "no open tasks left" rather than
	// guessing a fixed sleep.
	waitForConvergenceThenKill(t, buildID, id, cmd, done, logPath)
}

func waitForConvergenceThenKill(t *testing.T, buildID, id string, cmd *exec.Cmd, done chan error, logPath string) {
	t.Helper()
	pool := testPool(t)
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case err := <-done:
			if err != nil {
				t.Fatalf("runner %s exited with error: %v (log: %s)", id, err, logPath)
			}
			return
		default:
		}
		open, err := pool.Query(context.Background(), "SELECT count(*) FROM task WHERE build_id = $1 AND status IN ('pending','leased')", buildID)
		if err == nil {
			var n int
			if open.Next() {
				open.Scan(&n)
			}
			open.Close()
			if n == 0 {
				cmd.Process.Kill()
				<-done
				return
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	cmd.Process.Kill()
	<-done
	t.Fatalf("runner %s never converged the queue within the deadline (log: %s)", id, logPath)
}
