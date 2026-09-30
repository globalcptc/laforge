package orchestrator

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/globalcptc/laforge/internal/db"
)

// TestDispatchDueScheduledTaskCreatesRealAgentTask is the real,
// automated version of the live proof run manually against the actual
// docker-compose stack during development: a due scheduled_task row
// dispatches through the exact same CreateAgentTask machinery immediate
// ad-hoc dispatch uses, gets a real agent_task row a live agent's
// get-task would actually pick up, and (since this one isn't fires_once)
// reschedules itself rather than disappearing.
func TestDispatchDueScheduledTaskCreatesRealAgentTask(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	q := db.New(pool)
	build := newTestBuild(t, pool, "schedule-dispatch-test")

	team, err := q.EnsureTeam(ctx, db.EnsureTeamParams{BuildID: build.ID, TeamNumber: 1})
	if err != nil {
		t.Fatalf("EnsureTeam: %v", err)
	}
	obj, err := q.EnsureDeployedObject(ctx, db.EnsureDeployedObjectParams{
		TeamID: team.ID, Kind: "host", ObjectName: "webserver", Fingerprint: "test",
	})
	if err != nil {
		t.Fatalf("EnsureDeployedObject: %v", err)
	}

	targetJSON := []byte(`{"ids":["` + obj.ID.String() + `"]}`)
	st, err := q.CreateAdHocScheduledTask(ctx, db.CreateAdHocScheduledTaskParams{
		BuildID:    build.ID,
		Target:     targetJSON,
		WhenExpr:   "Every 30 minutes",
		Command:    "execute",
		Payload:    []byte(`{"command":"/bin/echo","args":["dispatch-test"]}`),
		FiresOnce:  false,
		NextFireAt: pgtype.Timestamptz{Time: time.Now().Add(-time.Second), Valid: true},
	})
	if err != nil {
		t.Fatalf("CreateAdHocScheduledTask: %v", err)
	}

	if err := DispatchDueScheduledTasks(ctx, pool, FixedRepoRoot("../../examples/lm-test"), 100); err != nil {
		t.Fatalf("DispatchDueScheduledTasks: %v", err)
	}

	tasks, err := q.ListAgentTasksByHost(ctx, obj.ID)
	if err != nil {
		t.Fatalf("ListAgentTasksByHost: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("agent_task rows for %s = %d, want 1", obj.ID, len(tasks))
	}
	if tasks[0].Command != "execute" {
		t.Errorf("agent_task.Command = %q, want \"execute\"", tasks[0].Command)
	}

	updated, err := q.GetScheduledTask(ctx, st.ID)
	if err != nil {
		t.Fatalf("GetScheduledTask: %v", err)
	}
	if updated.Status != "pending" {
		t.Errorf("scheduled_task.Status = %q, want \"pending\" (recurring, not fires-once)", updated.Status)
	}
	if !updated.NextFireAt.Valid || !updated.NextFireAt.Time.After(time.Now().Add(29*time.Minute)) {
		t.Errorf("scheduled_task.NextFireAt = %v, want ~30 minutes from now", updated.NextFireAt)
	}
}

// TestDispatchFiresOnceScheduledTaskMarksFired is the same proof for a
// fires-once (competition-start-anchored) entry: it fires exactly once,
// then genuinely stops -- status flips to "fired", not just an empty
// next_fire_at that a bug could reschedule from.
func TestDispatchFiresOnceScheduledTaskMarksFired(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	q := db.New(pool)
	build := newTestBuild(t, pool, "schedule-dispatch-once-test")

	team, err := q.EnsureTeam(ctx, db.EnsureTeamParams{BuildID: build.ID, TeamNumber: 1})
	if err != nil {
		t.Fatalf("EnsureTeam: %v", err)
	}
	obj, err := q.EnsureDeployedObject(ctx, db.EnsureDeployedObjectParams{
		TeamID: team.ID, Kind: "host", ObjectName: "webserver", Fingerprint: "test",
	})
	if err != nil {
		t.Fatalf("EnsureDeployedObject: %v", err)
	}

	targetJSON := []byte(`{"ids":["` + obj.ID.String() + `"]}`)
	st, err := q.CreateAdHocScheduledTask(ctx, db.CreateAdHocScheduledTaskParams{
		BuildID:    build.ID,
		Target:     targetJSON,
		WhenExpr:   "45 minutes after competition start",
		Command:    "reboot",
		Payload:    []byte(`{}`),
		Anchor:     "competition_start",
		FiresOnce:  true,
		NextFireAt: pgtype.Timestamptz{Time: time.Now().Add(-time.Second), Valid: true},
	})
	if err != nil {
		t.Fatalf("CreateAdHocScheduledTask: %v", err)
	}

	if err := DispatchDueScheduledTasks(ctx, pool, FixedRepoRoot("../../examples/lm-test"), 100); err != nil {
		t.Fatalf("DispatchDueScheduledTasks: %v", err)
	}

	updated, err := q.GetScheduledTask(ctx, st.ID)
	if err != nil {
		t.Fatalf("GetScheduledTask: %v", err)
	}
	if updated.Status != "fired" {
		t.Errorf("scheduled_task.Status = %q, want \"fired\"", updated.Status)
	}
	if updated.NextFireAt.Valid {
		t.Errorf("scheduled_task.NextFireAt = %v, want null (fires-once, done for good)", updated.NextFireAt)
	}

	// Running dispatch again must NOT create a second agent_task --
	// "fired" rows are excluded from ListDueScheduledTasks entirely.
	if err := DispatchDueScheduledTasks(ctx, pool, FixedRepoRoot("../../examples/lm-test"), 100); err != nil {
		t.Fatalf("DispatchDueScheduledTasks (second pass): %v", err)
	}
	tasks, err := q.ListAgentTasksByHost(ctx, obj.ID)
	if err != nil {
		t.Fatalf("ListAgentTasksByHost: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("agent_task rows after two dispatch passes = %d, want exactly 1 (fires-once must not re-fire)", len(tasks))
	}
}

// TestMatchAdHocTargetsByIDIgnoresOtherFilters proves the real bug fix
// tasks.go's own doc comment describes (IDs is a complete match on its
// own, since hostnames repeat across teams) still holds after the
// matching logic moved here.
func TestMatchAdHocTargetsByIDIgnoresOtherFilters(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	q := db.New(pool)
	build := newTestBuild(t, pool, "match-adhoc-test")

	team1, _ := q.EnsureTeam(ctx, db.EnsureTeamParams{BuildID: build.ID, TeamNumber: 1})
	team2, _ := q.EnsureTeam(ctx, db.EnsureTeamParams{BuildID: build.ID, TeamNumber: 2})
	obj1, _ := q.EnsureDeployedObject(ctx, db.EnsureDeployedObjectParams{TeamID: team1.ID, Kind: "host", ObjectName: "web01", Fingerprint: "t"})
	_, _ = q.EnsureDeployedObject(ctx, db.EnsureDeployedObjectParams{TeamID: team2.ID, Kind: "host", ObjectName: "web01", Fingerprint: "t"})

	matched, err := MatchAdHocTargets(ctx, q, build.ID, AdHocTarget{IDs: []string{obj1.ID.String()}, Search: "web01"})
	if err != nil {
		t.Fatalf("MatchAdHocTargets: %v", err)
	}
	if len(matched) != 1 || matched[0].ID != obj1.ID {
		t.Fatalf("matched = %+v, want exactly team 1's web01 -- IDs must be the complete match, not narrowed further by Search", matched)
	}
}

// TestMatchAdHocTargetsByTagAndNetwork proves the "operate on them" half of
// tags: a host/container's persisted tags and
// network are real target filters, AND-ed with the rest.
func TestMatchAdHocTargetsByTagAndNetwork(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	q := db.New(pool)
	build := newTestBuild(t, pool, "match-adhoc-tags")

	team1, _ := q.EnsureTeam(ctx, db.EnsureTeamParams{BuildID: build.ID, TeamNumber: 1})
	web, _ := q.EnsureDeployedObject(ctx, db.EnsureDeployedObjectParams{TeamID: team1.ID, Kind: "host", ObjectName: "web01", NetworkName: db.StrPtr("dmz"), Fingerprint: "t"})
	dbHost, _ := q.EnsureDeployedObject(ctx, db.EnsureDeployedObjectParams{TeamID: team1.ID, Kind: "host", ObjectName: "db01", NetworkName: db.StrPtr("internal"), Fingerprint: "t"})
	if err := q.SetDeployedObjectTags(ctx, db.SetDeployedObjectTagsParams{ID: web.ID, Tags: []byte(`{"role":"web","scored":"true"}`)}); err != nil {
		t.Fatalf("SetDeployedObjectTags(web): %v", err)
	}
	if err := q.SetDeployedObjectTags(ctx, db.SetDeployedObjectTagsParams{ID: dbHost.ID, Tags: []byte(`{"role":"db"}`)}); err != nil {
		t.Fatalf("SetDeployedObjectTags(db): %v", err)
	}

	only := func(target AdHocTarget, want pgtype.UUID, label string) {
		t.Helper()
		matched, err := MatchAdHocTargets(ctx, q, build.ID, target)
		if err != nil {
			t.Fatalf("%s: MatchAdHocTargets: %v", label, err)
		}
		if len(matched) != 1 || matched[0].ID != want {
			t.Fatalf("%s: matched %d objects, want exactly one (%s)", label, len(matched), label)
		}
	}

	only(AdHocTarget{Tags: map[string]string{"role": "web"}}, web.ID, "tag role=web")
	only(AdHocTarget{Tags: map[string]string{"scored": ""}}, web.ID, "tag presence scored")
	only(AdHocTarget{Tags: map[string]string{"role": "web", "scored": "true"}}, web.ID, "two tags AND")
	only(AdHocTarget{Network: "internal"}, dbHost.ID, "network internal")

	// A tag no object carries matches nothing.
	none, err := MatchAdHocTargets(ctx, q, build.ID, AdHocTarget{Tags: map[string]string{"role": "cache"}})
	if err != nil {
		t.Fatalf("MatchAdHocTargets(no match): %v", err)
	}
	if len(none) != 0 {
		t.Fatalf("tag role=cache matched %d objects, want 0", len(none))
	}
}
