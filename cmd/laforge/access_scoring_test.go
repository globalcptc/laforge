package main

import (
	"encoding/json"
	"os"
	"testing"
)

func TestScoringEntriesShape(t *testing.T) {
	row := func(team int32, host, obj, port string) accessExportRow {
		return accessExportRow{Team: team, Host: host, DeployedObjectID: obj, Kind: "host", Protocol: "tcp", InternalPort: port,
			PublicAddress: "51.159.110.159:4" + obj, Username: "Administrator", Password: "pw-" + obj, PasswordApplied: true}
	}
	rows := []accessExportRow{
		row(1, "vdi-win-01", "11", "3389"),
		row(1, "OutOfScope", "19", "3389"),
		row(1, "jump", "12", "3389"), row(1, "jump", "12", "22"),
		{Team: 1, Host: "web", DeployedObjectID: "13", Kind: "container", PublicAddress: "x:1"},
		row(2, "OutOfScope", "29", "3389"),
	}
	got := scoringEntries(rows, "cptc12", "us-west", 10, []string{"outofscope"}, "  Hello\n")
	if len(got) != 1 {
		t.Fatalf("got %d entries, want 1 (team 2 had only the excluded host)", len(got))
	}
	e := got[0]
	if e.TeamNumber != 11 || e.Season != "cptc12" || e.Competition != "us-west" || e.Instructions != "Hello" {
		t.Errorf("entry header = %+v", e)
	}
	var names []string
	for _, h := range e.Hosts {
		names = append(names, h.Host)
	}
	want := []string{"vdi-win-01", "jump (tcp/3389)", "jump (tcp/22)"}
	if len(names) != len(want) {
		t.Fatalf("hosts = %v, want %v", names, want)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Errorf("hosts = %v, want %v", names, want)
		}
	}
	if e.Hosts[0].URL != "51.159.110.159:411" || e.Hosts[0].Password != "pw-11" {
		t.Errorf("first host = %+v", e.Hosts[0])
	}
	// Dump for the cross-check against the scoring app's own schema.
	if p := os.Getenv("LAFORGE_SCORING_SAMPLE_OUT"); p != "" {
		b, _ := json.MarshalIndent(got, "", "  ")
		os.WriteFile(p, b, 0o600)
	}
}
