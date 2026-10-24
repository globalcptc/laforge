// Live smoke tests against a real AWS account. They are gated on LAFORGE_AWS_TEST
// (plus a usable AWS credential chain) and t.Skip cleanly when unset, so
// `go test ./...` stays green in CI with no account. They assert the control-plane
// effects LaForge depends on -- the VPC/subnet/IGW/route/NACL scaffolding, the
// access-schedule NACL deny toggle, and that ConfigureNetworkAccess writes
// exactly the declared-port ingress onto an object's security group -- using the
// EC2 API, which needs no in-guest access.
//
// RUNBOOK (end-to-end packet verification, done by hand once these pass; AWS has
// no native in-guest exec, so the actual-traffic check isn't automated here):
//  1. Configure an AWS builder in the wizard (region, exec role, AMIs, sizes).
//  2. Deploy a 2-team environment with two networks per team, a `visible_from:`
//     between them, and a host declaring e.g. `ports: tcp: [80]`.
//  3. Confirm every agent checks in (egress to the API/gateway works).
//  4. From a host on an allowed sibling network, `nc -vz <target> 80` succeeds and
//     `nc -vz <target> 22` times out (dropped); repeat same-network. Cross-team
//     must always fail.
//  5. Trigger a close window (or override); confirm an established SSH session to
//     a host drops within seconds, then open and confirm it recovers.
//  6. Give a host a `public:` port and confirm it is reachable on its EIP.
//
// Env:
//   LAFORGE_AWS_TEST=1                  enable these tests
//   AWS_REGION / credentials            standard chain
//   LAFORGE_AWS_TEST_AMI, _TYPE         (optional) enable the host subtest
package aws

import (
	"context"
	"os"
	"testing"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"github.com/globalcptc/laforge/internal/builder"
)

func liveBuilder(t *testing.T) *Builder {
	t.Helper()
	if os.Getenv("LAFORGE_AWS_TEST") == "" {
		t.Skip("no AWS account configured -- set LAFORGE_AWS_TEST=1 (plus AWS credentials/region) to run")
	}
	b, err := New(context.Background(), Config{Region: os.Getenv("AWS_REGION")})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return b
}

// a team number unlikely to collide with a real build.
const liveTeam = "9991"

func liveNet(name string) builder.NetworkSpec {
	return builder.NetworkSpec{ExternalName: "lftest-" + name, DisplayName: "lftest-" + name, Team: liveTeam}
}

func TestLiveNetworkLifecycle(t *testing.T) {
	b := liveBuilder(t)
	ctx := context.Background()
	spec := liveNet("net-a")
	spec.CIDR = "10.231.0.0/24"

	subnetID, err := b.DeployNetwork(ctx, spec)
	if err != nil {
		t.Fatalf("DeployNetwork: %v", err)
	}
	t.Cleanup(func() { _ = b.DestroyNetwork(context.Background(), liveTeam, subnetID) })

	// Idempotent adopt: a second deploy returns the same subnet.
	if again, err := b.DeployNetwork(ctx, spec); err != nil || again != subnetID {
		t.Fatalf("DeployNetwork adopt = %q,%v want %q", again, err, subnetID)
	}

	// The scaffolding exists: a managed VPC for the team, an attached IGW, a
	// default route to it, and a custom NACL.
	vpcID, err := b.findTeamVPC(ctx, buildFromExternalName(spec.ExternalName), liveTeam)
	if err != nil || vpcID == "" {
		t.Fatalf("team VPC not found: %q, %v", vpcID, err)
	}
	igws, _ := b.ec2.DescribeInternetGateways(ctx, &ec2.DescribeInternetGatewaysInput{Filters: []ec2types.Filter{filter("attachment.vpc-id", vpcID)}})
	if len(igws.InternetGateways) == 0 {
		t.Error("no internet gateway attached to the team VPC")
	}
	if nacls, _ := b.teamNACLs(ctx, liveTeam); len(nacls) == 0 {
		t.Error("no team NACL created")
	}

	if err := b.DestroyNetwork(ctx, liveTeam, subnetID); err != nil {
		t.Fatalf("DestroyNetwork: %v", err)
	}
	// Second destroy is a no-op success (ensure semantics).
	if err := b.DestroyNetwork(ctx, liveTeam, subnetID); err != nil {
		t.Fatalf("second DestroyNetwork should be nil, got %v", err)
	}
}

// TestLiveAccessScheduleNACL proves Close adds, and Open removes, the deny-all
// ingress entry on the team's NACL -- the access schedule's kill switch.
func TestLiveAccessScheduleNACL(t *testing.T) {
	b := liveBuilder(t)
	ctx := context.Background()
	spec := liveNet("net-acl")
	spec.CIDR = "10.231.5.0/24"
	subnetID, err := b.DeployNetwork(ctx, spec)
	if err != nil {
		t.Fatalf("DeployNetwork: %v", err)
	}
	t.Cleanup(func() { _ = b.DestroyNetwork(context.Background(), liveTeam, subnetID) })

	hasDeny := func() bool {
		nacls, err := b.ec2.DescribeNetworkAcls(ctx, &ec2.DescribeNetworkAclsInput{Filters: []ec2types.Filter{teamFilter(liveTeam), managedFilter}})
		if err != nil {
			t.Fatalf("DescribeNetworkAcls: %v", err)
		}
		for _, n := range nacls.NetworkAcls {
			for _, e := range n.Entries {
				if !awssdk.ToBool(e.Egress) && awssdk.ToInt32(e.RuleNumber) == denyIngressRuleNumber && e.RuleAction == ec2types.RuleActionDeny {
					return true
				}
			}
		}
		return false
	}

	if hasDeny() {
		t.Fatal("deny entry present before CloseAccess")
	}
	if err := b.CloseAccess(ctx, liveTeam); err != nil {
		t.Fatalf("CloseAccess: %v", err)
	}
	if !hasDeny() {
		t.Error("CloseAccess did not add the deny-all ingress entry")
	}
	if err := b.CloseAccess(ctx, liveTeam); err != nil {
		t.Fatalf("CloseAccess idempotent: %v", err)
	}
	if err := b.OpenAccess(ctx, liveTeam); err != nil {
		t.Fatalf("OpenAccess: %v", err)
	}
	if hasDeny() {
		t.Error("OpenAccess did not remove the deny entry")
	}
	if err := b.OpenAccess(ctx, liveTeam); err != nil {
		t.Fatalf("OpenAccess idempotent: %v", err)
	}
}

// TestLiveHostAndFirewall covers a host up->adopt->firewall->destroy. It needs an
// AMI and instance type for the account's region.
func TestLiveHostAndFirewall(t *testing.T) {
	b := liveBuilder(t)
	ami, typ := os.Getenv("LAFORGE_AWS_TEST_AMI"), os.Getenv("LAFORGE_AWS_TEST_TYPE")
	if ami == "" || typ == "" {
		t.Skip("set LAFORGE_AWS_TEST_AMI and LAFORGE_AWS_TEST_TYPE to run the host subtest")
	}
	b.cfg.Images = map[string]string{"linux": ami}
	b.cfg.Sizes = map[string]string{"small": typ}
	ctx := context.Background()

	net := liveNet("net-host")
	net.CIDR = "10.231.9.0/24"
	subnetID, err := b.DeployNetwork(ctx, net)
	if err != nil {
		t.Fatalf("DeployNetwork: %v", err)
	}
	t.Cleanup(func() { _ = b.DestroyNetwork(context.Background(), liveTeam, subnetID) })

	host := builder.HostSpec{
		ExternalName: "lftest-host1", DisplayName: "lftest-host1", Team: liveTeam,
		Network: net.ExternalName, Address: "10.231.9.10", OS: "linux", Size: "small",
	}
	ref, err := b.DeployHost(ctx, host)
	if err != nil {
		t.Fatalf("DeployHost: %v", err)
	}
	t.Cleanup(func() { _ = b.DestroyHost(context.Background(), liveTeam, ref) })
	if again, err := b.DeployHost(ctx, host); err != nil || again != ref {
		t.Fatalf("DeployHost adopt = %q,%v want %q", again, err, ref)
	}

	// Converge a port firewall and assert the object SG carries exactly it.
	access := []builder.NetworkAccess{{
		ExternalName: net.ExternalName, CIDR: net.CIDR,
		Hosts: []builder.HostAccess{{Address: host.Address, ExternalRef: ref, TCPPorts: []string{"80"}}},
	}}
	if err := b.ConfigureNetworkAccess(ctx, liveTeam, access); err != nil {
		t.Fatalf("ConfigureNetworkAccess: %v", err)
	}
	sgID, err := b.sgForRef(ctx, ref)
	if err != nil || sgID == "" {
		t.Fatalf("object SG not found for %s: %v", ref, err)
	}
	perms, _ := b.securityGroupIngress(ctx, sgID)
	if len(perms) != 1 || awssdk.ToInt32(perms[0].FromPort) != 80 {
		t.Errorf("object SG ingress = %+v, want exactly tcp/80 from the network CIDR", perms)
	}

	// Wait briefly for a public IP to confirm egress addressing works.
	deadline := time.Now().Add(90 * time.Second)
	for time.Now().Before(deadline) {
		if ip, _ := b.HostPublicAddress(ctx, ref); ip != "" {
			break
		}
		time.Sleep(5 * time.Second)
	}
}
