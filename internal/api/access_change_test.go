package api

import (
	"testing"
	"time"

	"github.com/globalcptc/laforge/internal/db"
	"github.com/jackc/pgx/v5/pgtype"
)

func accessRow(team, kind, status string, created time.Time, lastErr string) db.ListRecentTeamAccessTasksRow {
	r := db.ListRecentTeamAccessTasksRow{
		Team: team, Kind: kind, Status: status, Attempts: 1,
		CreatedAt: pgtype.Timestamptz{Time: created, Valid: true},
		UpdatedAt: pgtype.Timestamptz{Time: created.Add(time.Minute), Valid: true},
	}
	if lastErr != "" {
		r.LastError = &lastErr
	}
	return r
}

func TestTeamAccessChanges(t *testing.T) {
	t0 := time.Date(2026, 11, 1, 0, 15, 0, 0, time.UTC)
	teams := []db.Team{
		{TeamNumber: 1, AccessState: "open"},
		{TeamNumber: 2, AccessState: "open"},
		{TeamNumber: 3, AccessState: "closed"},
		{TeamNumber: 4, AccessState: "open"},
		{TeamNumber: 5, AccessState: "closed"},
		{TeamNumber: 6, AccessState: "open"},
	}
	// Newest first, as the query returns them.
	rows := []db.ListRecentTeamAccessTasksRow{
		// Team 1: a second close queued behind one that's running -> running, since the first.
		accessRow("1", "close_access", "pending", t0.Add(90*time.Second), ""),
		accessRow("1", "close_access", "leased", t0, ""),
		// Team 2: only queued.
		accessRow("2", "close_access", "pending", t0, ""),
		// Team 3: an open queued after it closed -> opening, the newest intent wins.
		accessRow("3", "open_access", "pending", t0.Add(time.Minute), ""),
		accessRow("3", "close_access", "done", t0, ""),
		// Team 4: close failed and the team is still open -> failed.
		accessRow("4", "close_access", "failed", t0, "lxd api: boom"),
		// Team 5: close failed earlier but the team is closed now -> nothing to show.
		accessRow("5", "close_access", "failed", t0, "transient"),
		// Team 6: last change done -> nothing in flight.
		accessRow("6", "open_access", "done", t0, ""),
		// A malformed payload is ignored.
		accessRow("x", "close_access", "pending", t0, ""),
	}
	got := teamAccessChanges(rows, teams)

	want := map[int32]accessChange{
		1: {Direction: "closing", State: "running", Since: t0, Attempts: 1},
		2: {Direction: "closing", State: "queued", Since: t0, Attempts: 1},
		3: {Direction: "opening", State: "queued", Since: t0.Add(time.Minute), Attempts: 1},
		4: {Direction: "closing", State: "failed", Since: t0.Add(time.Minute), Attempts: 1, Error: "lxd api: boom"},
	}
	if len(got) != len(want) {
		t.Fatalf("got changes for %d teams, want %d: %+v", len(got), len(want), got)
	}
	for team, w := range want {
		g, ok := got[team]
		if !ok {
			t.Fatalf("team %d: no change reported, want %+v", team, w)
		}
		if *g != w {
			t.Errorf("team %d: got %+v, want %+v", team, *g, w)
		}
	}
}
