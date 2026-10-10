package microcloud

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/globalcptc/laforge/internal/builder"
	"gopkg.in/yaml.v3"
)

type memoryPublicAllocator struct {
	addresses map[string]string
	released  int
}

func (a *memoryPublicAllocator) Reserve(_ context.Context, project, name, _ string, cfg PublicAccessConfig, _ bool) (string, error) {
	if a.addresses == nil {
		a.addresses = map[string]string{}
	}
	if ip := a.addresses[project+name]; ip != "" {
		return ip, nil
	}
	ip := fmt.Sprintf("10.250.3.%d", 100+len(a.addresses))
	a.addresses[project+name] = ip
	return ip, nil
}
func (a *memoryPublicAllocator) Release(_ context.Context, project, name string, _ ...string) error {
	delete(a.addresses, project+name)
	a.released++
	return nil
}

type publicTestServer struct {
	instances   map[string]map[string]interface{}
	acls        map[string]map[string]interface{}
	starts      int
	failDelete  bool
	networkIPv4 string
}

func newPublicTestServer(t *testing.T) (*publicTestServer, *Builder, *memoryPublicAllocator) {
	t.Helper()
	f := &publicTestServer{instances: map[string]map[string]interface{}{}, acls: map[string]map[string]interface{}{}, networkIPv4: "none"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ok := func(data interface{}) {
			json.NewEncoder(w).Encode(map[string]interface{}{"type": "sync", "metadata": data})
		}
		fail := func(status int, msg string) {
			w.WriteHeader(status)
			json.NewEncoder(w).Encode(map[string]interface{}{"type": "error", "error_code": status, "error": msg})
		}
		var body map[string]interface{}
		if r.Body != nil && r.Method != "GET" && r.Method != "DELETE" {
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("body: %v", err)
			}
		}
		path := r.URL.Path
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/1.0/cluster/members":
			// This fixture exercises networking against a non-clustered API.
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]interface{}{"type": "error", "error_code": 404, "error": "not clustered"})
		case strings.HasPrefix(path, "/1.0/networks/") && r.Method == "GET":
			ip := "172.16.0.1/24"
			if strings.HasSuffix(path, "GUEST_GUAC_WAN") {
				ip = f.networkIPv4
			}
			ok(map[string]interface{}{"type": "ovn", "status": "Created", "config": map[string]string{"ipv4.address": ip, "bridge.mtu": "1500"}})
		case path == "/1.0/instances" && r.Method == "POST":
			name := body["name"].(string)
			if f.instances[name] != nil {
				fail(409, "already exists")
				return
			}
			f.instances[name] = body
			ok(map[string]interface{}{})
		case path == "/1.0/network-acls" && r.Method == "POST":
			name := body["name"].(string)
			if f.acls[name] != nil {
				fail(409, "already exists")
				return
			}
			f.acls[name] = body
			ok(map[string]interface{}{})
		case strings.HasPrefix(path, "/1.0/network-acls/"):
			name := strings.TrimPrefix(path, "/1.0/network-acls/")
			if r.Method == "PUT" {
				f.acls[name] = body
			}
			if r.Method == "DELETE" {
				delete(f.acls, name)
			}
			ok(map[string]interface{}{})
		case strings.HasSuffix(path, "/state"):
			if r.Method == "GET" {
				ok(map[string]string{"status": "Running"})
				return
			}
			if body["action"] == "start" {
				f.starts++
			}
			ok(map[string]interface{}{})
		case path == "/1.0/instances/windows-base/snapshots/golden" && r.Method == "GET":
			if r.URL.Query().Get("project") != "templates" {
				t.Errorf("source snapshot read used project %s", r.URL.Query().Get("project"))
			}
			ok(map[string]interface{}{"expanded_config": map[string]string{"security.secureboot": "true"}, "devices": map[string]interface{}{
				"eth-1": map[string]string{"type": "nic", "network": "deleted-network"},
			}})
		case strings.HasPrefix(path, "/1.0/instances/"):
			name := strings.TrimPrefix(path, "/1.0/instances/")
			inst := f.instances[name]
			if inst == nil {
				fail(404, "not found")
				return
			}
			switch r.Method {
			case "GET":
				inst["expanded_devices"] = inst["devices"]
				ok(inst)
			case "DELETE":
				if f.failDelete {
					fail(503, "database unavailable")
					return
				}
				delete(f.instances, name)
				ok(map[string]interface{}{})
			case "PUT":
				f.instances[name] = body
				ok(map[string]interface{}{})
			case "PATCH":
				for _, key := range []string{"config", "devices"} {
					if patch, ok := body[key].(map[string]interface{}); ok {
						target := inst[key].(map[string]interface{})
						for k, v := range patch {
							target[k] = v
						}
					}
				}
				ok(map[string]interface{}{})
			default:
				t.Errorf("unexpected %s %s", r.Method, path)
				fail(500, "unexpected")
			}
		case strings.HasPrefix(path, "/1.0/storage-pools/") && r.Method == "DELETE":
			ok(map[string]interface{}{})
		default:
			t.Errorf("unexpected %s %s", r.Method, path)
			fail(500, "unexpected")
		}
	}))
	t.Cleanup(srv.Close)
	a := &memoryPublicAllocator{}
	b := New(&Client{BaseURL: srv.URL, Project: "competition", HTTPClient: srv.Client(), OperationTimeout: time.Second}, Config{PublicAccess: testPublicConfig(), Images: map[string]ImageRef{"linux": {VM: true, Fingerprint: "linux"}, "win2019": {VM: true, Fingerprint: "windows"}}, Sizes: map[string]SizeSpec{"small": {CPU: "2", Memory: "4GiB"}}, StoragePool: "Cluster_A_Pool"})
	b.PublicAddresses = a
	return f, b, a
}

func TestPublicNICLinuxWindowsAndLifecycle(t *testing.T) {
	f, b, a := newPublicTestServer(t)
	ctx := context.Background()
	ips := map[string]bool{}
	for _, os := range []string{"linux", "win2019"} {
		spec := builder.HostSpec{ExternalName: "same-team-" + os, Team: "1", OS: os, Size: "small", Network: "primary", Address: "172.16.0.10", CloudInitUserData: "test user data"}
		name, err := b.DeployHostWithPublicPorts(ctx, spec, []string{"22", "3389"}, []string{"53"})
		if err != nil {
			t.Fatal(err)
		}
		inst := f.instances[name]
		if profiles, ok := inst["profiles"].([]interface{}); !ok || len(profiles) != 0 {
			t.Fatalf("public host can inherit unwanted NICs from profiles: %v", inst["profiles"])
		}
		devices := inst["devices"].(map[string]interface{})
		nic := devices[publicNICName].(map[string]interface{})
		if nic["network"] != "GUEST_GUAC_WAN" || nic["ipv4.address"] != nil {
			t.Fatalf("DHCP-disabled device: %v", nic)
		}
		config := inst["config"].(map[string]interface{})
		var plan publicNICPlan
		if err := json.Unmarshal([]byte(config[publicNICKey].(string)), &plan); err != nil {
			t.Fatal(err)
		}
		if ips[plan.Address] {
			t.Fatal("two boxes share a public IP")
		}
		ips[plan.Address] = true
		var network struct {
			Version int `yaml:"version"`
			Config  []struct {
				MAC     string                   `yaml:"mac_address"`
				Subnets []map[string]interface{} `yaml:"subnets"`
			} `yaml:"config"`
		}
		if err := yaml.Unmarshal([]byte(plan.NetworkConfig), &network); err != nil {
			t.Fatal(err)
		}
		if network.Version != 1 || len(network.Config) != 2 || network.Config[0].MAC == network.Config[1].MAC {
			t.Fatalf("guest NIC mapping: %+v", network)
		}
		if network.Config[1].Subnets[0]["address"] != plan.Address || network.Config[1].Subnets[0]["gateway"] != nil || network.Config[0].Subnets[0]["gateway"] != "172.16.0.1" {
			t.Fatalf("guest addressing/routes: %+v", network)
		}
		if os == "win2019" {
			if devices["cidata"].(map[string]interface{})["source"] != "cloud-init:config" {
				t.Fatal("Windows has no native config drive")
			}
		}
		if _, err := b.DeployHostWithPublicPorts(ctx, spec, []string{"22"}, nil); err != nil {
			t.Fatal(err)
		}
		if err := b.removeNIC(ctx, name); err != nil {
			t.Fatal(err)
		}
		for _, d := range []string{"eth0", publicNICName} {
			if f.instances[name]["devices"].(map[string]interface{})[d].(map[string]interface{})["type"] != "none" {
				t.Fatalf("%s still open", d)
			}
		}
		endpoints, err := b.ConfigureExternalAccess(ctx, "1", []builder.ExternalHost{{ExternalRef: name, TCPPorts: []string{"22"}}})
		if err != nil || len(endpoints) != 1 || endpoints[0].PublicAddress != plan.Address+":22" || endpoints[0].ExternalPort != "22" {
			t.Fatalf("endpoints: %v, %v", endpoints, err)
		}
		if f.instances[name]["devices"].(map[string]interface{})[publicNICName].(map[string]interface{})["type"] != "none" {
			t.Fatal("ACL reconcile reopened closed NIC")
		}
		endpoints, err = b.ConfigureExternalAccess(ctx, "1", []builder.ExternalHost{{ExternalRef: name}})
		if err != nil || len(endpoints) != 0 {
			t.Fatal("public removal failed", err)
		}
		if len(f.acls[publicACLName("competition", name)]["ingress"].([]interface{})) != 0 {
			t.Fatal("removed public ports still allowed")
		}
		if err := b.restoreNIC(ctx, name); err != nil {
			t.Fatal(err)
		}
		if f.instances[name]["devices"].(map[string]interface{})[publicNICName].(map[string]interface{})["hwaddr"] != plan.Device["hwaddr"] {
			t.Fatal("MAC changed on reopen")
		}
		f.failDelete = true
		if err := b.DestroyHost(ctx, "1", name); err == nil || a.released != 0 {
			t.Fatal("failed delete freed address")
		}
		f.failDelete = false
	}
	for name := range f.instances {
		if err := b.DestroyHost(ctx, "1", name); err != nil {
			t.Fatal(err)
		}
	}
	if a.released != 2 || len(a.addresses) != 0 {
		t.Fatal("addresses not released after confirmed delete")
	}
}

func TestPublicNICSnapshotDoesNotInheritTemplateProfiles(t *testing.T) {
	f, b, _ := newPublicTestServer(t)
	b.Config.Images["win2019"] = ImageRef{VM: true, Source: SourceSnapshot, SourceProject: "templates", Instance: "windows-base", Snapshot: "golden"}
	name, err := b.DeployHostWithPublicPorts(context.Background(), builder.HostSpec{ExternalName: "snapshot-public", OS: "win2019", Size: "small", Network: "primary", Address: "172.16.0.10"}, []string{"3389"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	inst := f.instances[name]
	config := inst["config"].(map[string]interface{})
	if config["security.secureboot"] != "true" {
		t.Fatalf("snapshot lost its profile's firmware mode: %v", config["security.secureboot"])
	}
	profiles, ok := inst["profiles"].([]interface{})
	if !ok || len(profiles) != 0 {
		t.Fatalf("snapshot inherited profiles: %v", inst["profiles"])
	}
	devices := inst["devices"].(map[string]interface{})
	if len(devices) != 4 || devices["cidata"] == nil || devices["eth0"] == nil || devices[publicNICName] == nil || devices["root"] == nil {
		t.Fatalf("snapshot devices: %v", devices)
	}
}

func TestPublicNICInvalidNetworkSettingsDoNotReserveAnAddress(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*publicTestServer, *Builder)
	}{
		{"MTU exceeds network", func(_ *publicTestServer, b *Builder) { b.Config.PublicAccess.MTU = 1600 }},
		{"subnet differs", func(f *publicTestServer, _ *Builder) { f.networkIPv4 = "192.0.2.1/24" }},
		{"gateway inside pool", func(f *publicTestServer, _ *Builder) { f.networkIPv4 = "10.250.3.100/16" }},
		{"Windows without VM", func(_ *publicTestServer, b *Builder) { b.Config.Images["win2019"] = ImageRef{Fingerprint: "invalid"} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, b, a := newPublicTestServer(t)
			tc.change(f, b)
			_, err := b.DeployHostWithPublicPorts(context.Background(), builder.HostSpec{ExternalName: "preflight", OS: "win2019", Size: "small", Network: "primary", Address: "172.16.0.10"}, []string{"3389"}, nil)
			if err == nil || len(a.addresses) != 0 || len(f.instances) != 0 {
				t.Fatalf("invalid config reserved/created a box: err=%v addresses=%v", err, a.addresses)
			}
		})
	}
}

func TestPublicNICOwnershipGuardsTeardown(t *testing.T) {
	f, b, a := newPublicTestServer(t)
	ctx := context.Background()
	spec := builder.HostSpec{ExternalName: "build-original-host", OS: "linux", Size: "small", Network: "primary", Address: "172.16.0.10"}
	name, err := b.DeployHostWithPublicPorts(ctx, spec, []string{"22"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := b.DestroyHostForDeployment(ctx, "1", name, "other-build-colliding-name", false); err == nil {
		t.Fatal("deleted another deployment's public box")
	}
	if f.instances[name] == nil || len(a.addresses) != 1 || a.released != 0 {
		t.Fatal("ownership failure mutated another build")
	}
	if err := b.DestroyHostForDeployment(ctx, "1", name, spec.ExternalName, true); err != nil {
		t.Fatal(err)
	}
	if f.instances[name] != nil || len(a.addresses) != 1 {
		t.Fatal("rebuild did not keep reservation")
	}
	if err := b.DestroyHostForDeployment(ctx, "1", name, spec.ExternalName, false); err != nil {
		t.Fatal(err)
	}
	if len(a.addresses) != 0 {
		t.Fatal("confirmed absent box retained allocation after teardown")
	}
}

func TestPublicNICGuestRoutesDNSAndInheritedMTU(t *testing.T) {
	f, b, _ := newPublicTestServer(t)
	b.Config.PublicAccess.MTU = 0
	b.Config.PublicAccess.Gateway = "10.250.0.1"
	b.Config.PublicAccess.Routes = []PublicRoute{{To: "192.0.2.0/24", Via: "10.250.0.2"}}
	name, err := b.DeployHostWithPublicPorts(context.Background(), builder.HostSpec{ExternalName: "routes", OS: "win2019", Size: "small", Network: "primary", Address: "172.16.0.10"}, []string{"3389"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	var guest struct {
		Config []struct {
			MTU     int `yaml:"mtu"`
			Subnets []struct {
				Gateway string                                       `yaml:"gateway"`
				DNS     []string                                     `yaml:"dns_nameservers"`
				Routes  []struct{ Network, Netmask, Gateway string } `yaml:"routes"`
			} `yaml:"subnets"`
		} `yaml:"config"`
	}
	raw := f.instances[name]["config"].(map[string]interface{})["cloud-init.network-config"].(string)
	if err := yaml.Unmarshal([]byte(raw), &guest); err != nil {
		t.Fatal(err)
	}
	if len(guest.Config) != 2 {
		t.Fatalf("NICs: %+v", guest)
	}
	pub := guest.Config[1]
	if guest.Config[0].Subnets[0].Gateway != "" || pub.MTU != 1500 || pub.Subnets[0].Gateway != "10.250.0.1" || len(pub.Subnets[0].DNS) != 1 || pub.Subnets[0].DNS[0] != "10.250.0.1" {
		t.Fatalf("routing/DNS/MTU: %+v", guest)
	}
	routes := pub.Subnets[0].Routes
	if len(routes) != 1 || routes[0].Network != "192.0.2.0" || routes[0].Netmask != "255.255.255.0" || routes[0].Gateway != "10.250.0.2" {
		t.Fatalf("Cloudbase-Init subnet routes: %+v", routes)
	}
}

func TestPublicNICDHCPReservationAndPrivateHosts(t *testing.T) {
	f, b, a := newPublicTestServer(t)
	f.networkIPv4 = "10.250.0.1/16"
	spec := builder.HostSpec{ExternalName: "public", Team: "1", OS: "linux", Size: "small", Network: "primary", Address: "172.16.0.10"}
	name, err := b.DeployHostWithPublicPorts(context.Background(), spec, []string{"22"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if f.instances[name]["devices"].(map[string]interface{})[publicNICName].(map[string]interface{})["ipv4.address"] != "10.250.3.100" {
		t.Fatal("missing DHCP reservation")
	}
	spec.ExternalName = "private"
	name, err = b.DeployHostWithPublicPorts(context.Background(), spec, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := f.instances[name]["devices"].(map[string]interface{})[publicNICName]; ok || len(a.addresses) != 1 {
		t.Fatal("private host allocated public NIC")
	}
	if _, err := b.DeployHostWithPublicPorts(context.Background(), spec, []string{"22"}, nil); err == nil {
		t.Fatal("silently enabled first-boot config on an existing private host")
	}
}

func TestPublicNICRebuildRetainsAddressAndClosedRetry(t *testing.T) {
	f, b, a := newPublicTestServer(t)
	ctx := context.Background()
	spec := builder.HostSpec{ExternalName: "rebuild-me", Team: "1", OS: "linux", Size: "small", Network: "primary", Address: "172.16.0.10"}
	name, err := b.DeployHostWithPublicPorts(ctx, spec, []string{"22"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	address, err := b.HostPublicAddress(ctx, name)
	if err != nil {
		t.Fatal(err)
	}
	if err := b.removeNIC(ctx, name); err != nil {
		t.Fatal(err)
	}
	if _, err := b.DeployHostWithPublicPorts(ctx, spec, []string{"22"}, nil); err != nil {
		t.Fatal(err)
	}
	if f.instances[name]["devices"].(map[string]interface{})[publicNICName].(map[string]interface{})["type"] != "none" {
		t.Fatal("retry reopened a closed NIC")
	}
	if err := b.ensureHostPortACL(ctx, "172.16.0.0/24", nil, builder.HostAccess{ExternalRef: name, TCPPorts: []string{"22"}}); err != nil {
		t.Fatal(err)
	}
	if f.instances[name]["devices"].(map[string]interface{})["eth0"].(map[string]interface{})["type"] != "none" {
		t.Fatal("network ACL update reopened primary NIC")
	}
	if err := b.DestroyHostForRebuild(ctx, "1", name); err != nil {
		t.Fatal(err)
	}
	if a.released != 0 || len(a.addresses) != 1 {
		t.Fatal("rebuild freed public IP")
	}
	another := spec
	another.ExternalName = "other-deploy-during-rebuild"
	another.Address = "172.16.0.11"
	other, err := b.DeployHostWithPublicPorts(ctx, another, []string{"22"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	otherIP, _ := b.HostPublicAddress(ctx, other)
	if address == otherIP {
		t.Fatal("concurrent deployment took rebuilding box's IP")
	}
	if _, err := b.DeployHostWithPublicPorts(ctx, spec, []string{"22"}, nil); err != nil {
		t.Fatal(err)
	}
	after, err := b.HostPublicAddress(ctx, name)
	if err != nil || after != address {
		t.Fatalf("rebuild changed address %s -> %s (%v)", address, after, err)
	}
	if err := b.DestroyHost(ctx, "1", name); err != nil {
		t.Fatal(err)
	}
	if a.released != 1 {
		t.Fatal("permanent teardown retained address")
	}
}
