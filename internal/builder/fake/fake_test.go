package fake

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/globalcptc/laforge/internal/builder"
	"github.com/globalcptc/laforge/internal/db"
)

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	conn := os.Getenv("LAFORGE_TEST_DATABASE_URL")
	if conn == "" {
		if _, err := os.Stat("/tmp/.s.PGSQL.5432"); err != nil {
			t.Skip("no local Postgres available (set LAFORGE_TEST_DATABASE_URL to point at one)")
		}
		conn = "host=/tmp port=5432 user=lucas dbname=laforge_dev sslmode=disable"
	}
	_, pool, err := db.Open(context.Background(), conn)
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// TestFakeBuilderEnsureSemantics proves the fake builder satisfies the
// contract's central promise through the real Builder interface (not just
// the raw SQL underneath, which internal/db's own tests already cover):
// deploying the same ExternalName twice converges to one resource, and
// destroying is safe to call more than once, including on something that
// was never deployed.
func TestFakeBuilderEnsureSemantics(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	b := New(pool)

	ref := "fake-builder-test-host"
	t.Cleanup(func() {
		pool.Exec(ctx, "DELETE FROM fake_hoster_resource WHERE external_ref = $1", ref)
	})

	ref1, err := b.DeployHost(ctx, builder.HostSpec{ExternalName: ref, OS: "ubuntu22", Size: "small"})
	if err != nil {
		t.Fatalf("DeployHost: %v", err)
	}
	ref2, err := b.DeployHost(ctx, builder.HostSpec{ExternalName: ref, OS: "ubuntu22", Size: "small"})
	if err != nil {
		t.Fatalf("DeployHost (again): %v", err)
	}
	if ref1 != ref2 || ref1 != ref {
		t.Fatalf("DeployHost returned different refs on retry: %q then %q", ref1, ref2)
	}

	resources, err := b.Inspect(ctx)
	if err != nil {
		t.Fatalf("Inspect: %v", err)
	}
	count := 0
	for _, r := range resources {
		if r.ExternalRef == ref {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("Inspect found %d resources for %q after two DeployHost calls, want exactly 1", count, ref)
	}

	if err := b.DestroyHost(ctx, "", ref); err != nil {
		t.Fatalf("DestroyHost: %v", err)
	}
	// Safe to call again, and safe on something never deployed at all.
	if err := b.DestroyHost(ctx, "", ref); err != nil {
		t.Fatalf("DestroyHost (again, already destroyed): %v", err)
	}
	if err := b.DestroyHost(ctx, "", "never-deployed-at-all"); err != nil {
		t.Fatalf("DestroyHost (never existed): %v", err)
	}

	resources, err = b.Inspect(ctx)
	if err != nil {
		t.Fatalf("Inspect (after destroy): %v", err)
	}
	for _, r := range resources {
		if r.ExternalRef == ref {
			t.Fatalf("Inspect still lists %q as live after DestroyHost", ref)
		}
	}
}
