package incus

import (
	"strings"
	"testing"

	"github.com/gen0cide/laforge/configs"
	"github.com/lxc/incus/v6/shared/api"
)

func validConfig() BuilderConfig {
	return BuilderConfig{
		LaForgeServerURL:   "https://laforge.example.com",
		MaxBuildWorkers:    2,
		MaxTeardownWorkers: 2,
		GatewayImage:       "lf-gateway",
		Hosts: []HostConfig{{
			Name:                 "a",
			BaseURL:              "https://a:8443",
			ClientCertPath:       "c.crt",
			ClientKeyPath:        "c.key",
			ServerCertPath:       "s.crt",
			StoragePool:          "default",
			TransitNetwork:       "incusbr0",
			IngressListenAddress: "203.0.113.10",
			IngressPortBase:      10000,
			IngressPortStride:    100,
		}},
		InstanceSizes: map[string]InstanceSize{"small": {CPU: "2", Memory: "4GiB"}},
		Images:        map[string]string{"ubuntu22": "ubuntu22"},
	}
}

func TestSafeName(t *testing.T) {
	got := safeName("lf", "My Env", "team-01", "abcd1234")
	if got != "lf-my-env-team-01-abcd1234" {
		t.Fatalf("safeName = %q", got)
	}

	long := safeName(strings.Repeat("a", 100))
	if len(long) > maxIncusNameLength {
		t.Fatalf("safeName length = %d, want <= %d", len(long), maxIncusNameLength)
	}
	if long != safeName(strings.Repeat("a", 100)) {
		t.Fatal("truncated safeName is not stable")
	}
	if long == safeName(strings.Repeat("a", 99)+"b") {
		t.Fatal("truncated safeName collides for distinct inputs")
	}
}

func TestMacAddressStableAndValid(t *testing.T) {
	a := macAddress("host-1")
	if a != macAddress("host-1") {
		t.Fatal("macAddress is not stable")
	}
	if a == macAddress("host-2") {
		t.Fatal("macAddress collides for distinct seeds")
	}
	if len(a) != 17 || !strings.HasPrefix(a, "10:66:6a:") {
		t.Fatalf("macAddress = %q", a)
	}
}

func TestBridgeAndInterfaceNamesFitLinuxLimit(t *testing.T) {
	bridge := "lf" + hashHex(10, "build", "1", "vdi")
	if len(bridge) > 15 {
		t.Fatalf("bridge name %q is %d chars, limit 15", bridge, len(bridge))
	}
	if iface := gatewayInterface(bridge); len(iface) > 15 || iface == bridge {
		t.Fatalf("gateway interface %q invalid for bridge %q", iface, bridge)
	}
}

func TestNetworkGateway(t *testing.T) {
	got, err := networkGateway("10.1.2.0/24")
	if err != nil || got != "10.1.2.254/24" {
		t.Fatalf("networkGateway = %q, %v", got, err)
	}
	for _, bad := range []string{"10.1.2.0/25", "fd00::/64", "nonsense"} {
		if _, err := networkGateway(bad); err == nil {
			t.Fatalf("networkGateway(%q) succeeded, want error", bad)
		}
	}
}

func TestConfigValidation(t *testing.T) {
	if err := validConfig().Validate(); err != nil {
		t.Fatalf("valid config rejected: %v", err)
	}

	cases := map[string]func(*BuilderConfig){
		"no hosts":          func(c *BuilderConfig) { c.Hosts = nil },
		"duplicate hosts":   func(c *BuilderConfig) { c.Hosts = append(c.Hosts, c.Hosts[0]) },
		"missing gateway":   func(c *BuilderConfig) { c.GatewayImage = "" },
		"missing transit":   func(c *BuilderConfig) { c.Hosts[0].TransitNetwork = "" },
		"bad dns":           func(c *BuilderConfig) { c.DNSServers = []string{"not-an-ip"} },
		"bad listen":        func(c *BuilderConfig) { c.Hosts[0].IngressListenAddress = "nope" },
		"ingress no base":   func(c *BuilderConfig) { c.Hosts[0].IngressPortBase = 0 },
		"ingress no stride": func(c *BuilderConfig) { c.Hosts[0].IngressPortStride = 0 },
		"ingress overflow":  func(c *BuilderConfig) { c.Hosts[0].IngressPortBase = 65500 },
		"no sizes":          func(c *BuilderConfig) { c.InstanceSizes = nil },
		"no images":         func(c *BuilderConfig) { c.Images = nil },
		"zero workers":      func(c *BuilderConfig) { c.MaxBuildWorkers = 0 },
	}
	for name, mutate := range cases {
		config := validConfig()
		mutate(&config)
		if err := config.Validate(); err == nil {
			t.Errorf("%s: Validate succeeded, want error", name)
		}
	}

	config := validConfig()
	config.Hosts[0].IngressListenAddress = ""
	config.Hosts[0].IngressPortBase = 0
	if err := config.Validate(); err != nil {
		t.Errorf("host without ingress rejected: %v", err)
	}
}

func TestPickHostRoundRobin(t *testing.T) {
	order := []string{"a", "b", "c"}
	want := map[int]string{0: "a", 1: "a", 2: "b", 3: "c", 4: "a", 5: "b"}
	for team, host := range want {
		if got := pickHost(order, team); got != host {
			t.Errorf("pickHost(team %d) = %q, want %q", team, got, host)
		}
	}
}

func TestIngressPorts(t *testing.T) {
	host := validConfig().Hosts[0]

	got, err := ingressPorts(host, 3, "10.0.1.5", []string{"443", "22"}, []string{"53"})
	if err != nil {
		t.Fatal(err)
	}
	want := []gatewayIngress{
		{"tcp", 10300, "10.0.1.5", 22},
		{"tcp", 10301, "10.0.1.5", 443},
		{"udp", 10302, "10.0.1.5", 53},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d ports, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("port %d = %+v, want %+v", i, got[i], want[i])
		}
	}

	// Different teams never share a listen port.
	other, _ := ingressPorts(host, 4, "10.0.1.5", []string{"443", "22"}, []string{"53"})
	for _, a := range got {
		for _, b := range other {
			if a.HostPort == b.HostPort {
				t.Fatalf("teams 3 and 4 share host port %d", a.HostPort)
			}
		}
	}

	if _, err := ingressPorts(host, 3, "10.0.1.5", []string{"8000-8010"}, nil); err == nil {
		t.Error("port range accepted, want error")
	}
	if _, err := ingressPorts(host, 700, "10.0.1.5", []string{"22"}, nil); err == nil {
		t.Error("port above 65535 accepted, want error")
	}

	tooMany := make([]string, host.IngressPortStride+1)
	for i := range tooMany {
		tooMany[i] = "1000"
	}
	if _, err := ingressPorts(host, 1, "10.0.1.5", tooMany, nil); err == nil {
		t.Error("more ports than stride accepted, want error")
	}

	host.IngressListenAddress = ""
	if _, err := ingressPorts(host, 1, "10.0.1.5", []string{"22"}, nil); err == nil {
		t.Error("ingress without listen address accepted, want error")
	}
}

func TestGatewayDevices(t *testing.T) {
	host := validConfig().Hosts[0]
	network, err := gatewayNetworkFor("lfabc", "10.0.1.0/24")
	if err != nil {
		t.Fatal(err)
	}
	spec := gatewaySpec{
		Networks: []gatewayNetwork{network},
		Ingress:  []gatewayIngress{{"tcp", 10301, "10.0.1.5", 443}},
	}

	devices := gatewayDevices(spec, host)
	if devices["eth0"]["network"] != "incusbr0" {
		t.Errorf("transit device = %v", devices["eth0"])
	}
	nic := devices["net-lfabc"]
	if nic["network"] != "lfabc" || nic["name"] != network.Interface {
		t.Errorf("network nic = %v", nic)
	}
	proxy := devices["ingress-tcp-10301"]
	if proxy["listen"] != "tcp:203.0.113.10:10301" || proxy["connect"] != "tcp:10.0.1.5:443" {
		t.Errorf("proxy device = %v", proxy)
	}
}

func TestRenderGatewayScript(t *testing.T) {
	a, err := gatewayNetworkFor("lfaaa", "10.0.1.0/24")
	if err != nil {
		t.Fatal(err)
	}
	b, err := gatewayNetworkFor("lfbbb", "10.0.2.0/24")
	if err != nil {
		t.Fatal(err)
	}
	spec := gatewaySpec{
		Networks: []gatewayNetwork{b, a},
		Reservations: []gatewayReservation{
			{MAC: "10:66:6a:00:00:02", IP: "10.0.2.5"},
			{MAC: "10:66:6a:00:00:01", IP: "10.0.1.5"},
		},
		DNSServers: []string{"1.1.1.1", "8.8.8.8"},
	}

	script := renderGatewayScript(spec)
	for _, want := range []string{
		"ip addr replace 10.0.1.254/24 dev " + a.Interface,
		"ip addr replace 10.0.2.254/24 dev " + b.Interface,
		`oifname "eth0" masquerade`,
		"dhcp-range=set:" + a.Interface + ",10.0.1.0,static,255.255.255.0,12h",
		"dhcp-option=tag:" + a.Interface + ",option:dns-server,1.1.1.1,8.8.8.8",
		"dhcp-host=10:66:6a:00:00:01,10.0.1.5",
		"dhcp-host=10:66:6a:00:00:02,10.0.2.5",
		"systemctl restart dnsmasq",
	} {
		if !strings.Contains(script, want) {
			t.Errorf("script missing %q\n%s", want, script)
		}
	}

	// Rendering is deterministic regardless of input order.
	spec.Networks = []gatewayNetwork{a, b}
	spec.Reservations = []gatewayReservation{spec.Reservations[1], spec.Reservations[0]}
	if renderGatewayScript(spec) != script {
		t.Error("script depends on input order")
	}
}

func TestUserData(t *testing.T) {
	linux := userData("ubuntu22", "https://laforge/api/download/x", "pa'ss")
	for _, want := range []string{"curl -fSL", `'pa'"'"'ss'`, "-service install"} {
		if !strings.Contains(linux, want) {
			t.Errorf("linux user data missing %q", want)
		}
	}

	windows := userData("w2k19", "https://laforge/api/download/x", "pa'ss")
	for _, want := range []string{"<powershell>", "Invoke-WebRequest", "pa''ss"} {
		if !strings.Contains(windows, want) {
			t.Errorf("windows user data missing %q", want)
		}
	}
	if strings.Contains(windows, "1.1.1.1") {
		t.Error("windows user data still probes a public address")
	}
}

func TestValidateExistingInstance(t *testing.T) {
	desired := api.InstancePut{
		Config: map[string]string{"limits.cpu": "2"},
		Devices: map[string]map[string]string{
			"eth0": {"network": "lfabc"},
		},
	}
	instance := &api.Instance{
		Type: string(api.InstanceTypeVM),
		InstancePut: api.InstancePut{
			Config:  map[string]string{"limits.cpu": "2"},
			Devices: map[string]map[string]string{"eth0": {"network": "lfabc"}},
		},
	}
	if err := validateExistingInstance(instance, desired); err != nil {
		t.Fatalf("matching instance rejected: %v", err)
	}

	instance.Config["limits.cpu"] = "4"
	if err := validateExistingInstance(instance, desired); err == nil {
		t.Error("config drift accepted")
	}
	instance.Config["limits.cpu"] = "2"
	instance.Devices["eth0"]["network"] = "other"
	if err := validateExistingInstance(instance, desired); err == nil {
		t.Error("device drift accepted")
	}
}

func TestExampleConfigsAreValid(t *testing.T) {
	for _, path := range []string{
		"../../configs/incus.json.example",
		"../../configs/incus-multihost.json.example",
	} {
		var config BuilderConfig
		if err := configs.LoadBuilderConfig(path, &config); err != nil {
			t.Errorf("%s: %v", path, err)
			continue
		}
		if err := config.Validate(); err != nil {
			t.Errorf("%s: %v", path, err)
		}
	}
}
