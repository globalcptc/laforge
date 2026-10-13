package aws

import (
	"testing"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"github.com/globalcptc/laforge/internal/builder"
)

func TestParsePortRange(t *testing.T) {
	cases := []struct {
		in       string
		from, to int32
		ok       bool
	}{
		{"80", 80, 80, true},
		{" 443 ", 443, 443, true},
		{"8000-8100", 8000, 8100, true},
		{"0-65535", 0, 65535, true},
		{"100-50", 0, 0, false}, // inverted
		{"70000", 0, 0, false},  // out of range
		{"-5", 0, 0, false},
		{"abc", 0, 0, false},
		{"", 0, 0, false},
	}
	for _, c := range cases {
		from, to, ok := parsePortRange(c.in)
		if ok != c.ok || (ok && (from != c.from || to != c.to)) {
			t.Errorf("parsePortRange(%q) = %d,%d,%v want %d,%d,%v", c.in, from, to, ok, c.from, c.to, c.ok)
		}
	}
}

// TestPortPermissions is the per-host firewall's core rule generation: one
// permission per port(-range) per protocol, each carrying every source CIDR; a
// host with no ports (deny-all) or no sources yields nothing.
func TestPortPermissions(t *testing.T) {
	sources := []string{"10.0.1.0/24", "10.0.2.0/24"}
	perms := portPermissions(sources, []string{"80", "8000-8100"}, []string{"53"})
	if len(perms) != 3 {
		t.Fatalf("got %d permissions, want 3 (2 tcp + 1 udp)", len(perms))
	}
	// First tcp rule: port 80 from both sources.
	if awssdk.ToString(perms[0].IpProtocol) != "tcp" || awssdk.ToInt32(perms[0].FromPort) != 80 || awssdk.ToInt32(perms[0].ToPort) != 80 {
		t.Errorf("tcp/80 rule malformed: %+v", perms[0])
	}
	if len(perms[0].IpRanges) != 2 {
		t.Errorf("tcp/80 should list both source CIDRs, got %d", len(perms[0].IpRanges))
	}
	// Range rule.
	if awssdk.ToInt32(perms[1].FromPort) != 8000 || awssdk.ToInt32(perms[1].ToPort) != 8100 {
		t.Errorf("range rule = %d-%d, want 8000-8100", awssdk.ToInt32(perms[1].FromPort), awssdk.ToInt32(perms[1].ToPort))
	}
	// Udp rule.
	if awssdk.ToString(perms[2].IpProtocol) != "udp" || awssdk.ToInt32(perms[2].FromPort) != 53 {
		t.Errorf("udp rule malformed: %+v", perms[2])
	}

	// Deny-all: a host with no declared ports gets no ingress.
	if got := portPermissions(sources, nil, nil); got != nil {
		t.Errorf("no ports should yield no permissions, got %d", len(got))
	}
	// No sources (shouldn't happen -- a host is always at least on its own net)
	// yields nothing rather than an all-sources rule.
	if got := portPermissions(nil, []string{"80"}, nil); got != nil {
		t.Errorf("no sources should yield no permissions, got %d", len(got))
	}
	// Unparseable ports are skipped, not errored.
	if got := portPermissions(sources, []string{"bad"}, nil); got != nil {
		t.Errorf("only-bad ports should yield no permissions, got %d", len(got))
	}
}

func TestConfigFromInput(t *testing.T) {
	c := ConfigFromInput(ConfigInput{Region: "us-east-1", ECSCluster: "lab"}, map[string]string{"ubuntu": "ami-1"}, map[string]string{"small": "t3.micro"})
	if c.Region != "us-east-1" || c.Cluster != "lab" {
		t.Errorf("fields not mapped: %+v", c)
	}
	if c.Images["ubuntu"] != "ami-1" || c.Sizes["small"] != "t3.micro" {
		t.Errorf("image/size maps not carried: %+v", c)
	}
	// Empty fields stay empty so New applies env fallbacks.
	if c2 := ConfigFromInput(ConfigInput{}, nil, nil); c2.Cluster != "" || c2.ExecutionRoleARN != "" {
		t.Errorf("empty input should leave fields empty for env fallback, got %+v", c2)
	}
}

// TestExternalEndpoint pins the public-IP builder contract: no port remap, and
// the address is the host's own public IP.
func TestExternalEndpoint(t *testing.T) {
	e := externalEndpoint("i-abc", "tcp", "3389", "203.0.113.7")
	if e.InternalPort != "3389" || e.ExternalPort != "3389" {
		t.Errorf("public-IP builder must not remap ports: %+v", e)
	}
	if e.PublicAddress != "203.0.113.7:3389" {
		t.Errorf("PublicAddress = %q, want 203.0.113.7:3389", e.PublicAddress)
	}
	if e.ExternalRef != "i-abc" || e.Protocol != "tcp" {
		t.Errorf("endpoint fields wrong: %+v", e)
	}
}

func TestToImageInfo(t *testing.T) {
	img := ec2types.Image{
		ImageId:         awssdk.String("ami-0abc123"),
		Name:            awssdk.String("ubuntu-22.04-server"),
		Architecture:    ec2types.ArchitectureValuesX8664,
		PlatformDetails: awssdk.String("Linux/UNIX"),
		Description:     awssdk.String("Canonical Ubuntu 22.04"),
	}
	got := toImageInfo(img)
	if got.Fingerprint != "ami-0abc123" {
		t.Errorf("Fingerprint = %q, want the AMI id", got.Fingerprint)
	}
	if len(got.Aliases) != 1 || got.Aliases[0] != "ubuntu-22.04-server" {
		t.Errorf("Aliases = %v, want the AMI name", got.Aliases)
	}
	if got.Architecture != "x86_64" || got.Type != "virtual-machine" {
		t.Errorf("arch/type = %q/%q", got.Architecture, got.Type)
	}
	if got.Properties.OS != "Linux/UNIX" || got.Properties.Description != "Canonical Ubuntu 22.04" {
		t.Errorf("properties = %+v", got.Properties)
	}
}

func TestVPCCIDRFor(t *testing.T) {
	cases := map[string]string{
		"10.0.1.0/24":       "10.0.0.0/16",
		"10.0.2.0/24":       "10.0.0.0/16",
		"10.0.3.0/24":       "10.0.0.0/16",
		"172.16.5.0/24":     "172.16.0.0/16",
		"10.5.0.0/16":       "10.5.0.0/16",
		"192.168.50.128/25": "192.168.0.0/16",
	}
	for in, want := range cases {
		got, err := vpcCIDRFor(in)
		if err != nil || got != want {
			t.Errorf("vpcCIDRFor(%q) = %q,%v want %q", in, got, err, want)
		}
	}
	// A block larger than /16 can't be an AWS VPC.
	if _, err := vpcCIDRFor("10.0.0.0/8"); err == nil {
		t.Error("vpcCIDRFor(/8) should error (exceeds AWS /16 VPC max)")
	}
	if _, err := vpcCIDRFor("not-a-cidr"); err == nil {
		t.Error("vpcCIDRFor(garbage) should error")
	}
}

func TestAggregateInstanceMetrics(t *testing.T) {
	// Two instances: CPU 40 & 60 -> mean 50; net (in+out) 300+600+... summed and
	// divided by the 300s period; disk ops likewise.
	ids := []string{"i-a", "i-b"}
	latest := map[string]float64{
		"cpu_0": 40, "nin_0": 300, "nout_0": 300, "rops_0": 150, "wops_0": 150,
		"cpu_1": 60, "nin_1": 600, "nout_1": 0, "rops_1": 0, "wops_1": 0,
	}
	cpu, netBps, opsPerS := aggregateInstanceMetrics(ids, latest)
	if cpu == nil || *cpu != 50 {
		t.Errorf("cpu = %v, want 50", cpu)
	}
	// net bytes = (300+300)+(600+0) = 1200; /300 = 4 bytes/s.
	if netBps != 4 {
		t.Errorf("netBps = %v, want 4", netBps)
	}
	// ops = (150+150)+(0+0) = 300; /300 = 1 op/s.
	if opsPerS != 1 {
		t.Errorf("opsPerS = %v, want 1", opsPerS)
	}
	// No CPU datapoints at all -> nil (so it's omitted, not reported as 0%).
	if cpu, _, _ := aggregateInstanceMetrics([]string{"i-x"}, map[string]float64{}); cpu != nil {
		t.Errorf("cpu with no data = %v, want nil", cpu)
	}
}

func TestQuotaGauge(t *testing.T) {
	g := quotaGauge("vpcs", "VPCs (region)", 4, 5)
	if g.Key != "quota:vpcs" || g.Kind != builder.MetricGauge || g.Value != 4 {
		t.Errorf("gauge = %+v", g)
	}
	if g.Max == nil || *g.Max != 5 {
		t.Errorf("gauge max = %v, want 5", g.Max)
	}
}

func TestPolicySourceARN(t *testing.T) {
	cases := map[string]string{
		// user / role ARNs pass through unchanged
		"arn:aws:iam::677302527522:user/laforge-runner": "arn:aws:iam::677302527522:user/laforge-runner",
		"arn:aws:iam::677302527522:role/laforge-runner": "arn:aws:iam::677302527522:role/laforge-runner",
		// an assumed-role session ARN is rewritten to its role ARN
		"arn:aws:sts::677302527522:assumed-role/LaForgeRunner/i-0abc": "arn:aws:iam::677302527522:role/LaForgeRunner",
		// unexpected principals (root, garbage) -> skip
		"arn:aws:iam::677302527522:root": "",
		"nonsense":                       "",
		"":                               "",
	}
	for in, want := range cases {
		if got := policySourceARN(in); got != want {
			t.Errorf("policySourceARN(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestBuildFromExternalName(t *testing.T) {
	cases := map[string]string{
		"build-b81339b3-7d8a-40f3-b5f8-9ff8bf2b2f7e-team-1-network-lab": "b81339b3-7d8a-40f3-b5f8-9ff8bf2b2f7e",
		"build-abc-team-10-host-win01":                                  "abc",
		"lftest-net-a":                                                  "", // not a deterministic build name
		"build-team-1-network-lab":                                      "", // empty uuid segment
		"":                                                              "",
		"team-1-vpc":                                                    "",
	}
	for in, want := range cases {
		if got := buildFromExternalName(in); got != want {
			t.Errorf("buildFromExternalName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestTaskFamilyFromARN(t *testing.T) {
	cases := map[string]string{
		"arn:aws:ecs:us-east-1:123456789012:task-definition/laforge-web01:3": "laforge-web01",
		"arn:aws:ecs:us-east-1:123456789012:task-definition/laforge-web01":   "laforge-web01", // no revision
		"laforge-db01:12": "laforge-db01", // bare family:revision
		"laforge-db01":    "laforge-db01",
		"":                "",
	}
	for in, want := range cases {
		if got := taskFamilyFromARN(in); got != want {
			t.Errorf("taskFamilyFromARN(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestValidateVPCCIDR(t *testing.T) {
	// Accepted: IPv4 blocks from /16 (the VPC max) down to /28 (the min), kept
	// as declared rather than rounded to a /16.
	for _, ok := range []string{"10.0.0.0/16", "10.0.0.0/20", "192.168.0.0/24", "172.16.0.0/28"} {
		if err := validateVPCCIDR(ok); err != nil {
			t.Errorf("validateVPCCIDR(%q) = %v, want nil", ok, err)
		}
	}
	// Rejected: larger than /16, smaller than /28, non-IPv4, garbage.
	for _, bad := range []string{"10.0.0.0/8", "10.0.0.0/30", "::/64", "nonsense"} {
		if err := validateVPCCIDR(bad); err == nil {
			t.Errorf("validateVPCCIDR(%q) = nil, want an error", bad)
		}
	}
}

func TestCIDRWithin(t *testing.T) {
	if !cidrWithin("10.0.1.0/24", "10.0.0.0/16") {
		t.Error("10.0.1.0/24 should be within 10.0.0.0/16")
	}
	if !cidrWithin("10.0.0.0/16", "10.0.0.0/16") {
		t.Error("a /16 is within itself")
	}
	if cidrWithin("10.1.0.0/24", "10.0.0.0/16") {
		t.Error("10.1.0.0/24 is NOT within 10.0.0.0/16")
	}
	if cidrWithin("10.0.0.0/16", "10.0.1.0/24") {
		t.Error("a /16 is not within a /24 (wrong direction)")
	}
}

func TestAWSReservedAddress(t *testing.T) {
	cases := []struct {
		cidr, ip string
		reserved bool
	}{
		{"10.0.3.0/24", "10.0.3.0", true},  // network
		{"10.0.3.0/24", "10.0.3.1", true},  // VPC router
		{"10.0.3.0/24", "10.0.3.2", true},  // Amazon DNS (the one that failed in the field)
		{"10.0.3.0/24", "10.0.3.3", true},  // AWS "future use"
		{"10.0.3.0/24", "10.0.3.255", true}, // broadcast
		{"10.0.3.0/24", "10.0.3.4", false},  // first usable
		{"10.0.3.0/24", "10.0.3.100", false},
		{"10.0.0.0/16", "10.0.0.3", true},   // first four, bigger block
		{"10.0.0.0/16", "10.0.255.255", true}, // broadcast of a /16
		{"10.0.0.0/16", "10.0.0.4", false},
		{"10.0.3.0/24", "10.0.4.2", false},  // not in this subnet
	}
	for _, c := range cases {
		got, reason := awsReservedAddress(c.cidr, c.ip)
		if got != c.reserved {
			t.Errorf("awsReservedAddress(%q, %q) = %v (%q), want %v", c.cidr, c.ip, got, reason, c.reserved)
		}
	}
}
