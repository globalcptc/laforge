package microcloud

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/globalcptc/laforge/internal/builder"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// This test is explicitly opt-in. The JSON file describes a trusted cluster,
// exclusive test address range, Linux image and Windows snapshot. It creates
// disposable guests using the actual builder and the actual SQL allocator.
// No token enrollment or trust-store changes are performed.
func TestMicrocloudPublicNICLive(t *testing.T) {
	path := os.Getenv("LAFORGE_MICROCLOUD_PUBLIC_TEST_CONFIG")
	if path == "" {
		t.Skip("set LAFORGE_MICROCLOUD_PUBLIC_TEST_CONFIG for the explicit live smoke test")
	}
	var input struct {
		APIURL, ClientCertPath, ClientKeyPath, ServerCertPath, Project       string
		Run                                                                  string
		Config                                                               Config
		LinuxOS, WindowsOS, LinuxUserDataPath, WindowsUserDataPath, JumpHost string
		Keep, VerifyGuests, Rebuild                                          bool
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(raw, &input); err != nil {
		t.Fatal(err)
	}
	if input.LinuxOS == "" && input.WindowsOS == "" {
		t.Fatal("configure at least one Linux or Windows test guest")
	}
	read := func(path string) []byte {
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	client, err := NewClient(input.APIURL, read(input.ClientCertPath), read(input.ClientKeyPath), read(input.ServerCertPath), input.Project)
	if err != nil {
		t.Fatal(err)
	}
	client.OperationTimeout = 10 * time.Minute
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Minute)
	defer cancel()
	dsn := os.Getenv("LAFORGE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Fatal("live test requires a migrated test database for durable allocations")
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	run := input.Run
	if run == "" {
		run = fmt.Sprintf("public-nic-smoke-%d", time.Now().Unix())
	}
	if !strings.HasPrefix(run, "public-nic-smoke-") {
		t.Fatal("test run must start with public-nic-smoke-")
	}
	var id pgtype.UUID
	if err = pool.QueryRow(ctx, `INSERT INTO builder_config (name,kind) VALUES ($1,'microcloud') ON CONFLICT(name) DO UPDATE SET name=EXCLUDED.name RETURNING id`, run).Scan(&id); err != nil {
		t.Fatal(err)
	}
	b := New(client, input.Config)
	b.PublicAddresses = SQLPublicAddressAllocator{Pool: pool, BuilderID: id}
	netSpec := builder.NetworkSpec{ExternalName: run + "-net", DisplayName: "publictest", Team: run, CIDR: "172.16.231.0/24"}
	netName := networkName(netSpec.DisplayName, netSpec.ExternalName)
	var names []string
	defer func() {
		if input.Keep {
			t.Logf("KEPT test builder %s, network %s, instances %v; clean these exact resources after inspection", id.String(), netName, names)
			return
		}
		cleanup, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		clean := true
		for _, name := range names {
			if err := b.DestroyHost(cleanup, run, name); err != nil {
				t.Errorf("cleanup %s: %v", name, err)
				clean = false
			}
		}
		if clean {
			if err := b.DestroyNetwork(cleanup, run, netName); err != nil {
				t.Errorf("cleanup network: %v", err)
				clean = false
			}
		}
		if clean {
			if _, err := pool.Exec(cleanup, `DELETE FROM builder_config WHERE id=$1`, id); err != nil {
				t.Errorf("cleanup builder: %v", err)
			}
		}
	}()
	t.Logf("Creating primary OVN %s on %s", netName, b.Config.OVNUplinkNetwork)
	if err := b.retryInstanceStep(ctx, netName, "live test ensuring primary network", func() error {
		_, err := b.DeployNetwork(ctx, netSpec)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	used := map[string]bool{}
	type guest struct {
		name, address, marker string
		spec                  builder.HostSpec
	}
	var guests []guest
	publicAddress := func(name string) string {
		t.Helper()
		endpoints, err := b.ConfigureExternalAccess(ctx, run, []builder.ExternalHost{{ExternalRef: name, TCPPorts: []string{"18080"}}})
		if err != nil {
			t.Fatal(err)
		}
		if len(endpoints) != 1 {
			t.Fatalf("%s: expected one public endpoint, got %v", name, endpoints)
		}
		return strings.Split(endpoints[0].PublicAddress, ":")[0]
	}
	for index, osName := range []string{input.LinuxOS, input.WindowsOS} {
		if osName == "" {
			continue
		}
		userPath := input.LinuxUserDataPath
		if index == 1 {
			userPath = input.WindowsUserDataPath
		}
		spec := builder.HostSpec{ExternalName: run + "-" + osName, DisplayName: "publictest-" + osName, Team: run, OS: osName, Size: "test", Network: netSpec.ExternalName, NetworkDisplayName: netSpec.DisplayName, Address: fmt.Sprintf("172.16.231.%d", 10+index), DiskGB: 80, CloudInitUserData: string(read(userPath))}
		name := instanceName(spec.DisplayName, spec.ExternalName)
		names = append(names, name)
		t.Logf("Deploying %s (%s)", name, osName)
		if _, err := b.DeployHostWithPublicPorts(ctx, spec, []string{"18080"}, nil); err != nil {
			t.Fatal(err)
		}
		address := publicAddress(name)
		if used[address] {
			t.Fatal("duplicate public address")
		}
		used[address] = true
		marker := "laforge-public-nic-smoke linux"
		if index == 1 {
			marker = "laforge-public-nic-smoke windows"
		}
		guests = append(guests, guest{name: name, address: address, marker: marker, spec: spec})
		t.Logf("%s public=%s primary=%s", name, address, spec.Address)
	}
	if input.Keep && !input.VerifyGuests && !input.Rebuild {
		return
	} // Inspection runs deliberately separate provisioning from guest diagnostics.
	verifyGuest := func(g guest) {
		t.Helper()
		var last string
		reachable := false
		for attempt := 0; attempt < 30; attempt++ {
			probe, stop := context.WithTimeout(ctx, 15*time.Second)
			out, err := exec.CommandContext(probe, "ssh", "-o", "BatchMode=yes", "-o", "ConnectTimeout=5", input.JumpHost, "curl --fail --silent --connect-timeout 3 --max-time 5 http://"+g.address+":18080/").CombinedOutput()
			stop()
			last = string(out)
			if err == nil && strings.TrimSpace(last) == g.marker {
				t.Logf("Guest %s reachable on %s: %s", g.name, g.address, strings.TrimSpace(last))
				reachable = true
				break
			}
			select {
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			case <-time.After(10 * time.Second):
			}
		}
		if !reachable {
			t.Errorf("guest %s (%s) did not serve its first-boot marker: %s", g.name, g.address, last)
		}
	}
	for _, g := range guests {
		verifyGuest(g)
	}
	if input.Rebuild && !t.Failed() {
		for _, g := range guests {
			t.Logf("Rebuilding %s while retaining public IP %s", g.name, g.address)
			if err := b.DestroyHostForDeployment(ctx, run, g.name, g.spec.ExternalName, true); err != nil {
				t.Fatal(err)
			}
			if _, err := b.DeployHostWithPublicPorts(ctx, g.spec, []string{"18080"}, nil); err != nil {
				t.Fatal(err)
			}
			if address := publicAddress(g.name); address != g.address {
				t.Fatalf("rebuild changed %s public IP: %s -> %s", g.name, g.address, address)
			}
			verifyGuest(g)
		}
	}
}
