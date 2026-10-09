package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/globalcptc/laforge/internal/builder"
	"github.com/globalcptc/laforge/internal/builder/fake"
	"github.com/globalcptc/laforge/internal/db"
	"github.com/globalcptc/laforge/internal/loader"
	"github.com/globalcptc/laforge/internal/orchestrator"
	"github.com/globalcptc/laforge/internal/runner"
)

// Exercise the optional MicroCloud hooks through the actual runner, database,
// step materializer, and authenticated API. Hoster HTTP/SQL allocation behavior
// is covered separately in the microcloud package.
type publicAddressTestBuilder struct {
	*fake.Builder
	addresses              map[string]string
	rebuilt, publicDeploys int
}

func (b *publicAddressTestBuilder) HostReference(externalName, _ string) string {
	return externalName + "-mc"
}
func (b *publicAddressTestBuilder) DeployHostWithPublicPorts(ctx context.Context, spec builder.HostSpec, _, _ []string) (string, error) {
	b.publicDeploys++
	spec.ExternalName = b.HostReference(spec.ExternalName, spec.DisplayName)
	ref, err := b.Builder.DeployHost(ctx, spec)
	if err == nil && b.addresses[ref] == "" {
		b.addresses[ref] = fmt.Sprintf("10.250.3.%d", 100+len(b.addresses))
	}
	return ref, err
}
func (b *publicAddressTestBuilder) HostPublicAddress(_ context.Context, ref string) (string, error) {
	return b.addresses[ref], nil
}
func (b *publicAddressTestBuilder) DestroyHostForRebuild(ctx context.Context, team, ref string) error {
	if b.addresses[ref] == "" {
		return fmt.Errorf("rebuild used the wrong instance reference %s", ref)
	}
	b.rebuilt++
	return b.Builder.DestroyHost(ctx, team, ref)
}
func (b *publicAddressTestBuilder) DestroyHostForDeployment(ctx context.Context, team, ref, externalName string, rebuild bool) error {
	if ref != b.HostReference(externalName, "") {
		return fmt.Errorf("incorrect deployment identity %s", externalName)
	}
	if rebuild {
		return b.DestroyHostForRebuild(ctx, team, ref)
	}
	return b.DestroyHost(ctx, team, ref)
}
func (b *publicAddressTestBuilder) DestroyHost(ctx context.Context, team, ref string) error {
	if err := b.Builder.DestroyHost(ctx, team, ref); err != nil {
		return err
	}
	delete(b.addresses, ref)
	return nil
}

func TestPublicAddressRunnerTemplatesAPIAndRebuild(t *testing.T) {
	env := setupAPITest(t)
	ctx := context.Background()
	repo := t.TempDir()
	if err := os.CopyFS(repo, os.DirFS("../../examples/m7-two-team")); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(repo, "scripts"), 0755); err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string]string{
		"hosts/m7-webserver.yaml":     "host:\n  name: m7-webserver\n  os: alpine\n  size: small\n  disk: 5\n  ports: {tcp: [\"80\"]}\n  public: {tcp: [\"80\"]}\n  steps:\n    - script: public-address\n",
		"scripts/public-address.yaml": "script:\n  name: public-address\n  language: bash\n  source: public-address.sh\n  timeout: 60\n",
		"scripts/public-address.sh":   "echo '{{ .host.public_address }}'\n",
	} {
		if err := os.WriteFile(filepath.Join(repo, name), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	env.server.RepoRoot = repo
	rev, err := env.q.CreateContentRevision(ctx, db.CreateContentRevisionParams{RepositoryID: env.repo.ID, CommitSha: "public-address-test"})
	if err != nil {
		t.Fatal(err)
	}
	build, err := env.q.CreateBuild(ctx, db.CreateBuildParams{ContentRevisionID: rev.ID, EnvironmentName: "m7-two-team"})
	if err != nil {
		t.Fatal(err)
	}
	build, err = env.q.SetBuildStatus(ctx, db.SetBuildStatusParams{ID: build.ID, Status: "deploying"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		env.server.Pool.Exec(ctx, `DELETE FROM fake_hoster_resource WHERE external_ref LIKE $1`, "build-"+build.ID.String()+"-%")
	})
	b := &publicAddressTestBuilder{Builder: fake.New(env.server.Pool), addresses: map[string]string{}}
	r := &runner.Runner{Pool: env.server.Pool, Builder: b, RepoRoot: repo, ID: "public-address", BuildID: &build.ID, LeaseDuration: time.Minute, HeartbeatInterval: 5 * time.Second}
	drain := func() {
		for i := 0; i < 30; i++ {
			did, err := r.LeaseAndExecuteOne(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if !did {
				return
			}
		}
		t.Fatal("task queue did not drain")
	}
	reconcile := func() {
		if err := orchestrator.Reconcile(ctx, env.server.Pool, repo, build.ID); err != nil {
			t.Fatal(err)
		}
	}
	reconcile()
	drain()
	reconcile() // Materializes scripts only after the public IP is recorded.
	objs, err := env.q.ListDeployedObjectsByBuild(ctx, build.ID)
	if err != nil {
		t.Fatal(err)
	}
	var target db.DeployedObject
	for _, o := range objs {
		if o.Kind == "host" {
			target = o
			break
		}
	}
	if target.PublicAddress == "" || b.publicDeploys != 2 {
		t.Fatalf("missing public allocation: %+v", target)
	}
	steps, err := env.q.ListAgentTasksByHost(ctx, target.ID)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, step := range steps {
		if strings.Contains(string(step.Payload), target.PublicAddress) {
			found = true
		}
	}
	if !found {
		t.Fatal("materialized script did not receive public IP")
	}
	client := signedInClient(t, env, "read-only-viewer")
	base := env.httpURL + "/builds/" + build.ID.String() + "/objects"
	get := func(path string, result interface{}) {
		resp, err := client.Get(path)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("GET %s: %d", path, resp.StatusCode)
		}
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			t.Fatal(err)
		}
	}
	var listed []db.DeployedObject
	get(base, &listed)
	found = false
	for _, o := range listed {
		if o.ID == target.ID && o.PublicAddress == target.PublicAddress {
			found = true
		}
	}
	if !found {
		t.Fatal("objects API omitted public IP")
	}
	var config objectConfig
	get(base+"/"+target.ID.String()+"/config", &config)
	if config.PublicAddress != target.PublicAddress {
		t.Fatalf("config API public IP: %q", config.PublicAddress)
	}
	var rendered []renderedStep
	get(base+"/"+target.ID.String()+"/render", &rendered)
	if len(rendered) != 1 || !strings.Contains(rendered[0].Rendered, target.PublicAddress) {
		t.Fatalf("API template render: %+v", rendered)
	}
	var events []db.Event
	get(base+"/"+target.ID.String()+"/events", &events)
	found = false
	for _, e := range events {
		if e.Kind == "public_ip.assigned" && strings.Contains(e.Message, target.PublicAddress) {
			found = true
		}
	}
	if !found {
		t.Fatal("assignment absent from object event history")
	}

	c, err := loader.Load(repo)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := orchestrator.Rebuild(ctx, env.server.Pool, build, c, orchestrator.AdHocTarget{IDs: []string{target.ID.String()}}, false, false); err != nil {
		t.Fatal(err)
	}
	reset, err := env.q.GetDeployedObject(ctx, target.ID)
	if err != nil || reset.PublicAddress != target.PublicAddress {
		t.Fatalf("rebuild lost public IP: %v %s", err, reset.PublicAddress)
	}
	drain() // Destroy uses the MicroCloud naming hook even after reset cleared ref.
	reconcile()
	drain()
	rebuilt, err := env.q.GetDeployedObject(ctx, target.ID)
	if err != nil || rebuilt.PublicAddress != target.PublicAddress || b.rebuilt != 1 {
		t.Fatalf("rebuild changed IP or skipped retention: %+v %v count=%d", rebuilt, err, b.rebuilt)
	}
	if err := orchestrator.Teardown(ctx, env.server.Pool, build.ID); err != nil {
		t.Fatal(err)
	}
	drain()
	if len(b.addresses) != 0 {
		t.Fatalf("destroyed build retained IPs: %v", b.addresses)
	}
	dead, err := env.q.GetDeployedObject(ctx, target.ID)
	if err != nil || dead.PublicAddress != "" || dead.Status != "destroyed" {
		t.Fatalf("destroyed object still reports public IP: %+v %v", dead, err)
	}
}
