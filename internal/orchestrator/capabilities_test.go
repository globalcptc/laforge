package orchestrator

import (
	"context"
	"encoding/json"
	"github.com/globalcptc/laforge/internal/agentpki"
	"github.com/globalcptc/laforge/internal/builder/incus"
	"github.com/globalcptc/laforge/internal/db"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/globalcptc/laforge/internal/loader"
)

func lmTestEnv(t *testing.T) (*loader.Content, *loader.Environment) {
	t.Helper()
	c, err := loader.Load("../../examples/lm-test")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(c.Errors) != 0 {
		t.Fatalf("example repo should load clean: %+v", c.Errors)
	}
	env := findEnvironment(c, "lm-test")
	if env == nil {
		t.Fatal("lm-test environment not found")
	}
	return c, env
}

func containerNamesOf(c *loader.Content) map[string]bool {
	m := make(map[string]bool, len(c.Containers))
	for _, ct := range c.Containers {
		m[ct.Name] = true
	}
	return m
}

// There is no capability gating anymore -- content is builder-agnostic -- so the
// only pre-build check is image-map completeness: every os a host uses must be
// mapped in the builder config.

func TestCheckEnvironmentImagesRejectsMissingImageMapping(t *testing.T) {
	c, env := lmTestEnv(t)
	// The exact "windows-server-2022 not listed" scenario:
	// an images map missing an os this environment actually uses.
	images := map[string]bool{"ubuntu22": true} // lm-test's hosts use more than this
	err := checkEnvironmentImages(env, c, containerNamesOf(c), images)
	if err == nil {
		t.Fatal("expected an error for an os/image the builder has no image configured for")
	}
	if !strings.Contains(err.Error(), "no image configured for") {
		t.Errorf("error should name the missing-image reason, got: %v", err)
	}
}

func TestCheckEnvironmentImagesSkipsWhenNilImages(t *testing.T) {
	// nil images means "this builder kind doesn't expose an inspectable image
	// map" (e.g. the fake builder) -- not "definitely wrong."
	c, env := lmTestEnv(t)
	if err := checkEnvironmentImages(env, c, containerNamesOf(c), nil); err != nil {
		t.Errorf("expected no error when images is nil, got: %v", err)
	}
}

func TestPlacementOverflowFailsBeforeDeploymentTasks(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	q := db.New(pool)
	build := newTestBuildWithFakeBuilder(t, pool, "bridge-capacity")
	ca, err := agentpki.GenerateCA("placement-test")
	if err != nil {
		t.Fatal(err)
	}
	cert, key := filepath.Join(t.TempDir(), "client.crt"), filepath.Join(t.TempDir(), "client.key")
	if err := os.WriteFile(cert, ca.CertPEM, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(key, ca.KeyPEM, 0600); err != nil {
		t.Fatal(err)
	}
	hosts, _ := json.Marshal([]incus.HostConfig{{APIURL: "https://unused.invalid:8443", ClientCertPath: cert, ClientKeyPath: key, ServerCertPEM: string(ca.CertPEM), NetworkType: "bridge"}})
	if _, err := pool.Exec(ctx, "UPDATE builder_config SET kind='incus', incus_hosts=$1, incus_one_team_per_host=true WHERE name='bridge-capacity-builder'", hosts); err != nil {
		t.Fatal(err)
	}
	err = Reconcile(ctx, pool, "../../examples/lm-test", build.ID)
	if err == nil || !strings.Contains(err.Error(), "one team per host") {
		t.Fatalf("want placement error, got %v", err)
	}
	tasks, err := q.ListTasksByBuild(ctx, build.ID)
	if err != nil || len(tasks) != 0 {
		t.Fatalf("impossible layout queued deployment work: %d tasks, %v", len(tasks), err)
	}
}
