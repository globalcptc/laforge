package microcloud

import (
	"strings"
	"testing"

	"github.com/canonical/lxd/shared/api"
)

func TestSafeName(t *testing.T) {
	t.Parallel()

	got := safeName("LaForge", "Team 01", "DMZ_network")
	if got != "laforge-team-01-dmz-network" {
		t.Fatalf("safeName() = %q", got)
	}
}

func TestSafeNameTruncatesWithStableUniqueSuffix(t *testing.T) {
	t.Parallel()

	first := safeName(strings.Repeat("a", 70), "first")
	second := safeName(strings.Repeat("a", 70), "second")
	if len(first) > maxLXDNameLength {
		t.Fatalf("safeName() length = %d", len(first))
	}
	if first == second {
		t.Fatalf("different long names collapsed to %q", first)
	}
	if first != safeName(strings.Repeat("a", 70), "first") {
		t.Fatalf("safeName() is not stable")
	}
}

func TestNetworkGateway(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cidr    string
		want    string
		wantErr bool
	}{
		{name: "slash 24", cidr: "10.20.30.0/24", want: "10.20.30.254/24"},
		{name: "slash 16", cidr: "10.20.0.0/16", want: "10.20.0.254/16"},
		{name: "small subnet", cidr: "10.20.30.0/25", wantErr: true},
		{name: "IPv6", cidr: "fd00::/64", wantErr: true},
		{name: "invalid", cidr: "not-a-cidr", wantErr: true},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := networkGateway(test.cidr)
			if test.wantErr {
				if err == nil {
					t.Fatalf("networkGateway(%q) returned no error", test.cidr)
				}
				return
			}
			if err != nil {
				t.Fatalf("networkGateway(%q): %v", test.cidr, err)
			}
			if got != test.want {
				t.Fatalf("networkGateway(%q) = %q, want %q", test.cidr, got, test.want)
			}
		})
	}
}

func TestUserData(t *testing.T) {
	t.Parallel()

	linux := userData("ubuntu22", "https://laforge.example/agent", "p'ass")
	if !strings.Contains(linux, "curl -fSL") {
		t.Fatalf("Linux user data does not download the agent")
	}
	if !strings.Contains(linux, `'p'"'"'ass'`) {
		t.Fatalf("Linux user data does not shell-escape the password")
	}

	windows := userData("w2k22", "https://laforge.example/agent", "unused")
	if !strings.Contains(windows, "Invoke-WebRequest") {
		t.Fatalf("Windows user data does not download the agent")
	}
}

func TestConfigValidation(t *testing.T) {
	t.Parallel()

	valid := BuilderConfig{
		LaForgeServerURL:   "https://laforge.example",
		MaxBuildWorkers:    1,
		MaxTeardownWorkers: 1,
		BaseURL:            "https://microcloud.example:8443",
		ClientCertPath:     "client.crt",
		ClientKeyPath:      "client.key",
		ServerCertPath:     "server.crt",
		StoragePool:        "remote",
		UplinkNetwork:      "UPLINK",
		InstanceSizes: map[string]InstanceSize{
			"small": {CPU: "2", Memory: "4GiB"},
		},
		Images: map[string]string{"ubuntu22": "ubuntu22"},
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid config failed validation: %v", err)
	}

	valid.BaseURL = ""
	if err := valid.Validate(); err == nil {
		t.Fatalf("config missing base_url passed validation")
	}
}

func TestValidateExistingInstance(t *testing.T) {
	t.Parallel()

	desired := api.InstancePut{
		Config: map[string]string{
			"limits.cpu":    "2",
			"limits.memory": "4GiB",
		},
		Devices: map[string]map[string]string{
			"eth0": {
				"type":         "nic",
				"network":      "lf-vdi-build",
				"ipv4.address": "10.0.0.10",
			},
		},
	}
	instance := &api.Instance{
		Type:       string(api.InstanceTypeVM),
		Location:   "member-1",
		StatusCode: api.Running,
		Config:     cloneVars(desired.Config),
		Devices: map[string]map[string]string{
			"eth0": cloneVars(desired.Devices["eth0"]),
		},
	}

	if err := validateExistingInstance(instance, desired, "member-1"); err != nil {
		t.Fatalf("matching instance rejected: %v", err)
	}

	instance.Devices["eth0"]["network"] = "wrong"
	if err := validateExistingInstance(instance, desired, "member-1"); err == nil {
		t.Fatalf("instance with wrong network accepted")
	}
}
