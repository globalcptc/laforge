package orchestrator

import (
	"context"
	"testing"

	"github.com/globalcptc/laforge/internal/db"
)

// TestAdvanceBuildStatus covers the build phase transitions the redesign
// introduced: a build moves deploying -> building (infra up, agents still
// working) -> finished/failed (every object terminal). A build must not
// reach "finished" while any object is still running/building. Found live
// originally as a fully-provisioned build stuck in "deploying" forever.
func TestAdvanceBuildStatus(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	q := db.New(pool)

	// newObj ensures one deployed_object on build and drives it to status.
	// "finished" goes through running first, since SetDeployedObjectFinished
	// only advances an object that is running/building (the same guard the
	// live poll relies on).
	newObj := func(t *testing.T, build db.Build, teamNum int32, name, status string) db.DeployedObject {
		t.Helper()
		team, err := q.EnsureTeam(ctx, db.EnsureTeamParams{BuildID: build.ID, TeamNumber: teamNum})
		if err != nil {
			t.Fatalf("EnsureTeam: %v", err)
		}
		obj, err := q.EnsureDeployedObject(ctx, db.EnsureDeployedObjectParams{
			TeamID: team.ID, Kind: "host", ObjectName: name,
		})
		if err != nil {
			t.Fatalf("EnsureDeployedObject: %v", err)
		}
		switch status {
		case "running":
			if _, err := q.MarkDeployedObjectRunning(ctx, db.MarkDeployedObjectRunningParams{ID: obj.ID}); err != nil {
				t.Fatalf("MarkDeployedObjectRunning: %v", err)
			}
		case "finished":
			if _, err := q.MarkDeployedObjectRunning(ctx, db.MarkDeployedObjectRunningParams{ID: obj.ID}); err != nil {
				t.Fatalf("MarkDeployedObjectRunning: %v", err)
			}
			if err := q.SetDeployedObjectFinished(ctx, obj.ID); err != nil {
				t.Fatalf("SetDeployedObjectFinished: %v", err)
			}
		case "deploy_failed":
			msg := "boom"
			if _, err := q.MarkDeployedObjectDeployFailed(ctx, db.MarkDeployedObjectDeployFailedParams{ID: obj.ID, LastError: &msg}); err != nil {
				t.Fatalf("MarkDeployedObjectDeployFailed: %v", err)
			}
		case "pending":
			// EnsureDeployedObject already leaves it pending.
		}
		return obj
	}

	assertStatus := func(t *testing.T, buildID db.Build, want string) {
		t.Helper()
		got, err := q.GetBuild(ctx, buildID.ID)
		if err != nil {
			t.Fatalf("GetBuild: %v", err)
		}
		if got.Status != want {
			t.Fatalf("build status = %q, want %q", got.Status, want)
		}
	}

	t.Run("all objects finished settles to finished", func(t *testing.T) {
		build := newTestBuild(t, pool, "advance-finished")
		newObj(t, build, 1, "web", "finished")
		newObj(t, build, 1, "db", "finished")
		settled, err := AdvanceBuildStatus(ctx, pool, build.ID)
		if err != nil {
			t.Fatalf("AdvanceBuildStatus: %v", err)
		}
		if !settled {
			t.Fatal("settled = false, want true")
		}
		assertStatus(t, build, "finished")
	})

	t.Run("a failed object settles to failed", func(t *testing.T) {
		build := newTestBuild(t, pool, "advance-failed")
		newObj(t, build, 1, "web", "finished")
		newObj(t, build, 1, "db", "deploy_failed")
		settled, err := AdvanceBuildStatus(ctx, pool, build.ID)
		if err != nil {
			t.Fatalf("AdvanceBuildStatus: %v", err)
		}
		if !settled {
			t.Fatal("settled = false, want true")
		}
		assertStatus(t, build, "failed")
	})

	t.Run("an object still building leaves it building, not settled", func(t *testing.T) {
		build := newTestBuild(t, pool, "advance-building")
		newObj(t, build, 1, "web", "finished")
		newObj(t, build, 1, "db", "running") // infra up, agent still working
		settled, err := AdvanceBuildStatus(ctx, pool, build.ID)
		if err != nil {
			t.Fatalf("AdvanceBuildStatus: %v", err)
		}
		if settled {
			t.Fatal("settled = true, want false (still building)")
		}
		assertStatus(t, build, "building")
	})

	t.Run("an in-progress infra object leaves it deploying", func(t *testing.T) {
		build := newTestBuild(t, pool, "advance-inprogress")
		newObj(t, build, 1, "web", "finished")
		newObj(t, build, 1, "db", "pending")
		settled, err := AdvanceBuildStatus(ctx, pool, build.ID)
		if err != nil {
			t.Fatalf("AdvanceBuildStatus: %v", err)
		}
		if settled {
			t.Fatal("settled = true, want false (still deploying)")
		}
		assertStatus(t, build, "deploying")
	})

	t.Run("a terminal build is left alone", func(t *testing.T) {
		build := newTestBuild(t, pool, "advance-terminal")
		if _, err := q.SetBuildStatus(ctx, db.SetBuildStatusParams{ID: build.ID, Status: "finished"}); err != nil {
			t.Fatalf("SetBuildStatus: %v", err)
		}
		newObj(t, build, 1, "web", "finished")
		settled, err := AdvanceBuildStatus(ctx, pool, build.ID)
		if err != nil {
			t.Fatalf("AdvanceBuildStatus: %v", err)
		}
		if settled {
			t.Fatal("settled = true, want false (already terminal, not our job)")
		}
	})
}
