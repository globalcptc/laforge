package orchestrator

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/globalcptc/laforge/internal/db"
)

// TestCheckAgentCheckins covers the never-checked-in watchdog: per-OS
// deadlines, one flag + event per deploy, console capture and the optional
// forced reboot through the builder, and a rebuilt host's stale session not
// counting as a check-in.
func TestCheckAgentCheckins(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	q := db.New(pool)
	build := newTestBuildWithFakeBuilder(t, pool, "checkin-watchdog")
	team, err := q.EnsureTeam(ctx, db.EnsureTeamParams{BuildID: build.ID, TeamNumber: 1})
	if err != nil {
		t.Fatalf("EnsureTeam: %v", err)
	}
	// host deploys an object with the given agent platform, deployed `ago`.
	host := func(name, platform string, ago time.Duration) pgtype.UUID {
		t.Helper()
		obj, err := q.EnsureDeployedObject(ctx, db.EnsureDeployedObjectParams{TeamID: team.ID, Kind: "host", ObjectName: name})
		if err != nil {
			t.Fatalf("EnsureDeployedObject: %v", err)
		}
		if err := q.SetDeployedObjectAgentPlatform(ctx, db.SetDeployedObjectAgentPlatformParams{ID: obj.ID, AgentPlatform: platform}); err != nil {
			t.Fatal(err)
		}
		if _, err := q.MarkDeployedObjectRunning(ctx, db.MarkDeployedObjectRunningParams{ID: obj.ID, ExternalRef: db.StrPtr("ref-" + name)}); err != nil {
			t.Fatal(err)
		}
		setDeployedAt(t, pool, obj.ID, time.Now().Add(-ago))
		return obj.ID
	}
	const linux, windows = "x86_64-unknown-linux-musl", "x86_64-pc-windows-gnu"
	stuckLinux := host("kali", linux, 20*time.Minute)  // past 15m
	bootingWin := host("vdi", windows, 20*time.Minute) // under 30m: still booting
	stuckWin := host("dc", windows, 45*time.Minute)    // past 30m
	freshLinux := host("web", linux, 2*time.Minute)    // well within
	checkedIn := host("db", linux, 40*time.Minute)     // overdue but its agent is in
	rebuilt := host("rebuilt", linux, 20*time.Minute)  // has only a pre-redeploy session
	noAgent := host("appliance", "", 2*time.Hour)      // no agent delivered: never expected
	if _, err := q.UpsertAgentSession(ctx, db.UpsertAgentSessionParams{DeployedObjectID: checkedIn, CertFingerprint: "x"}); err != nil {
		t.Fatal(err)
	}
	if _, err := q.UpsertAgentSession(ctx, db.UpsertAgentSessionParams{DeployedObjectID: rebuilt, CertFingerprint: "x"}); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "UPDATE agent_session SET last_heartbeat_at = now() - interval '1 hour' WHERE deployed_object_id = $1", rebuilt); err != nil {
		t.Fatal(err)
	}

	w := DefaultCheckinWatchdog
	w.Reboot = true
	if err := CheckAgentCheckins(ctx, pool, build.ID, w); err != nil {
		t.Fatalf("CheckAgentCheckins: %v", err)
	}
	// A second pass must not flag (or reboot) anything again.
	if err := CheckAgentCheckins(ctx, pool, build.ID, w); err != nil {
		t.Fatalf("CheckAgentCheckins (second pass): %v", err)
	}

	wantFlagged := map[pgtype.UUID]bool{stuckLinux: true, stuckWin: true, rebuilt: true}
	for _, id := range []pgtype.UUID{stuckLinux, bootingWin, stuckWin, freshLinux, checkedIn, rebuilt, noAgent} {
		obj, err := q.GetDeployedObject(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		if obj.AgentCheckinOverdueAt.Valid != wantFlagged[id] {
			t.Errorf("%s: overdue flagged = %v, want %v", obj.ObjectName, obj.AgentCheckinOverdueAt.Valid, wantFlagged[id])
		}
		var events int
		var raw string
		if err := pool.QueryRow(ctx, "SELECT count(*), COALESCE(max(payload::text), '') FROM event WHERE deployed_object_id = $1 AND kind = 'agent_checkin_overdue'", id).Scan(&events, &raw); err != nil {
			t.Fatalf("reading events: %v", err)
		}
		want := 0
		if wantFlagged[id] {
			want = 1
		}
		if events != want {
			t.Errorf("%s: agent_checkin_overdue events = %d, want %d", obj.ObjectName, events, want)
		}
		if want == 1 {
			var payload map[string]any
			if err := json.Unmarshal([]byte(raw), &payload); err != nil {
				t.Fatalf("%s: payload: %v", obj.ObjectName, err)
			}
			if payload["console_log"] != "fake console log for ref-"+obj.ObjectName+"\n" {
				t.Errorf("%s: console_log = %v", obj.ObjectName, payload["console_log"])
			}
			if payload["rebooted"] != true {
				t.Errorf("%s: rebooted = %v, want true", obj.ObjectName, payload["rebooted"])
			}
		}
	}

	// A redeploy clears the flag and restarts the clock.
	if _, err := q.MarkDeployedObjectRunning(ctx, db.MarkDeployedObjectRunningParams{ID: stuckLinux, ExternalRef: db.StrPtr("ref-kali-2")}); err != nil {
		t.Fatal(err)
	}
	if obj, _ := q.GetDeployedObject(ctx, stuckLinux); obj.AgentCheckinOverdueAt.Valid || !obj.DeployedAt.Valid || time.Since(obj.DeployedAt.Time) > time.Minute {
		t.Fatalf("after redeploy: overdue=%v deployed_at=%v, want cleared + now", obj.AgentCheckinOverdueAt, obj.DeployedAt)
	}
}

// TestCheckAgentCheckinsDisabledDeadline: a zero deadline turns the check off
// for that OS.
func TestCheckAgentCheckinsDisabledDeadline(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	q := db.New(pool)
	build := newTestBuild(t, pool, "checkin-watchdog-off")
	team, _ := q.EnsureTeam(ctx, db.EnsureTeamParams{BuildID: build.ID, TeamNumber: 1})
	obj, _ := q.EnsureDeployedObject(ctx, db.EnsureDeployedObjectParams{TeamID: team.ID, Kind: "host", ObjectName: "win"})
	q.SetDeployedObjectAgentPlatform(ctx, db.SetDeployedObjectAgentPlatformParams{ID: obj.ID, AgentPlatform: "x86_64-pc-windows-gnu"})
	q.MarkDeployedObjectRunning(ctx, db.MarkDeployedObjectRunningParams{ID: obj.ID, ExternalRef: db.StrPtr("r")})
	setDeployedAt(t, pool, obj.ID, time.Now().Add(-5*time.Hour))

	if err := CheckAgentCheckins(ctx, pool, build.ID, CheckinWatchdog{Linux: time.Minute}); err != nil {
		t.Fatal(err)
	}
	if got, _ := q.GetDeployedObject(ctx, obj.ID); got.AgentCheckinOverdueAt.Valid {
		t.Fatal("a Windows host was flagged with the Windows deadline disabled")
	}
}

func setDeployedAt(t *testing.T, pool *pgxpool.Pool, id pgtype.UUID, at time.Time) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), "UPDATE deployed_object SET deployed_at = $2 WHERE id = $1", id, at); err != nil {
		t.Fatal(err)
	}
}
