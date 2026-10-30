package orchestrator

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/globalcptc/laforge/internal/db"
)

// TestFingerprintNotSharedAcrossEnvironments is the CPTC12 SoCal/Central
// regression: two builds of different environments on ONE content revision (one
// commit defines both). The same host and team render differently in each (here
// a var), so each build must get its own fingerprints. When the fingerprint
// cache was keyed without the environment, the second build to reconcile read
// the first one's values, saw every host as changed, and recreated all of them.
func TestFingerprintNotSharedAcrossEnvironments(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	q := db.New(pool)

	// One content tree holding two environments that differ only in a var.
	root := t.TempDir()
	if err := os.CopyFS(root, os.DirFS("../../examples/lm-test")); err != nil {
		t.Fatal(err)
	}
	env, err := os.ReadFile(filepath.Join(root, "lm-test.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	other := strings.Replace(strings.Replace(string(env), "name: lm-test", "name: lm-test-b", 1), "company: Allports", "company: OtherCo", 1)
	if err := os.WriteFile(filepath.Join(root, "lm-test-b.yaml"), []byte(other), 0o644); err != nil {
		t.Fatal(err)
	}

	a := newTestBuild(t, pool, "fp-env-a")
	b, err := q.CreateBuild(ctx, db.CreateBuildParams{ContentRevisionID: a.ContentRevisionID, EnvironmentName: "lm-test-b"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := q.SetBuildStatus(ctx, db.SetBuildStatusParams{ID: b.ID, Status: "deploying"}); err != nil {
		t.Fatal(err)
	}

	// fps: each host's desired fingerprint, as its deploy task carries it.
	fps := func(buildID pgtype.UUID) map[string]string {
		rows, err := pool.Query(ctx, `SELECT tm.team_number::text || '/' || o.as_name, t.payload->>'fingerprint' FROM task t
			JOIN deployed_object o ON o.id = t.deployed_object_id
			JOIN team tm ON tm.id = o.team_id
			WHERE t.build_id = $1 AND t.kind = 'deploy_host'`, buildID)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		m := map[string]string{}
		for rows.Next() {
			var name, fp string
			if err := rows.Scan(&name, &fp); err != nil {
				t.Fatal(err)
			}
			m[name] = fp
		}
		return m
	}

	// reconcile runs a build until its hosts queue: hosts wait for their
	// networks, so bring those up (no runner here) and pass again.
	reconcile := func(id pgtype.UUID) {
		if err := Reconcile(ctx, pool, root, id, nil); err != nil {
			t.Fatalf("Reconcile: %v", err)
		}
		if _, err := pool.Exec(ctx, `UPDATE deployed_object SET status = 'running', external_ref = 'net-'||id::text
			WHERE kind = 'network' AND team_id IN (SELECT id FROM team WHERE build_id = $1)`, id); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `UPDATE task SET status = 'done' WHERE build_id = $1 AND kind = 'deploy_network'`, id); err != nil {
			t.Fatal(err)
		}
		if err := Reconcile(ctx, pool, root, id, nil); err != nil {
			t.Fatalf("Reconcile: %v", err)
		}
	}

	// b reconciled with a cold cache: its true fingerprints.
	fpCache = &fingerprintCache{m: map[string]string{}, max: 20000}
	reconcile(b.ID)
	want := fps(b.ID)

	// Now a first, then b again on the cache a warmed.
	fpCache = &fingerprintCache{m: map[string]string{}, max: 20000}
	reconcile(a.ID)
	// Start b over and reconcile it on the cache a just warmed.
	if _, err := pool.Exec(ctx, "DELETE FROM task WHERE build_id = $1", b.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "DELETE FROM team WHERE build_id = $1", b.ID); err != nil {
		t.Fatal(err)
	}
	reconcile(b.ID)
	got := fps(b.ID)
	differs := 0
	for name, fp := range fps(a.ID) {
		if want[name] != fp {
			differs++
		}
	}
	if differs == 0 {
		t.Fatal("fixture: no host fingerprint differs between the environments")
	}
	if len(want) == 0 {
		t.Fatal("fixture: b queued no host deploys")
	}
	for name, fp := range want {
		if got[name] != fp {
			t.Errorf("%s: fingerprint %s after a's reconcile warmed the cache, want %s (its own environment's)", name, got[name], fp)
		}
	}
}
