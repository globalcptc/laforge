// AWS Service Quotas gauges for the builder page. Each entry pairs a Service
// Quotas code (the limit, applied value) with a way to count current usage, so
// the page shows usage-vs-limit for the account limits that gate a LaForge
// deploy. The limits that actually bite are the ones LaForge consumes per team
// (a VPC, subnets, an internet gateway, ENIs, security groups, Elastic IPs) and
// the running-instance vCPU cap. Extend quotaGaugeSpecs to track another quota;
// a wrong code or missing permission degrades to a warning, not a broken page.
//
// Scope note: Client VPN concurrent-connection quotas are intentionally not here
// -- LaForge's VPN is WireGuard on a guest, not an AWS Client VPN endpoint, so
// there's no usage to report. Per-VPC / per-ENI quotas (subnets, security groups)
// are reported as the max across the relevant parent, since that's the value that
// trips the per-parent limit.
package aws

import (
	"context"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// quotaGaugeSpec is one usage-vs-limit gauge: the Service Quotas code for the
// limit plus a usage counter. service is "ec2" or "vpc".
type quotaGaugeSpec struct {
	key     string
	label   string
	service string
	code    string
	usage   func(ctx context.Context) (float64, error)
}

func (b *Builder) quotaGaugeSpecs() []quotaGaugeSpec {
	return []quotaGaugeSpec{
		{"vpcs", "VPCs (region)", "vpc", "L-F678F1CE", b.countVPCs},
		{"subnets", "Subnets (max per VPC)", "vpc", "L-407747CB", b.countSubnetsMaxPerVPC},
		{"enis", "Network interfaces (region)", "vpc", "L-DF5E4CA3", b.countNetworkInterfaces},
		{"igws", "Internet gateways (region)", "vpc", "L-A4707A72", b.countInternetGateways},
		{"sgs_per_eni", "Security groups (max per NIC)", "vpc", "L-2AFB9258", b.countSecurityGroupsMaxPerENI},
		{"eips", "Elastic IPs (region)", "ec2", "L-0263D0A3", b.countElasticIPs},
		{"vcpus", "Running on-demand vCPUs (standard)", "ec2", "L-1216C47A", b.countRunningStandardVCPUs},
	}
}

func (b *Builder) countVPCs(ctx context.Context) (float64, error) {
	out, err := b.ec2.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{})
	if err != nil {
		return 0, err
	}
	return float64(len(out.Vpcs)), nil
}

func (b *Builder) countElasticIPs(ctx context.Context) (float64, error) {
	out, err := b.ec2.DescribeAddresses(ctx, &ec2.DescribeAddressesInput{})
	if err != nil {
		return 0, err
	}
	return float64(len(out.Addresses)), nil
}

func (b *Builder) countInternetGateways(ctx context.Context) (float64, error) {
	out, err := b.ec2.DescribeInternetGateways(ctx, &ec2.DescribeInternetGatewaysInput{})
	if err != nil {
		return 0, err
	}
	return float64(len(out.InternetGateways)), nil
}

// countSubnetsMaxPerVPC returns the largest subnet count in any one VPC -- the
// value the per-VPC subnet quota limits (there is no region-wide subnet quota).
func (b *Builder) countSubnetsMaxPerVPC(ctx context.Context) (float64, error) {
	byVPC := map[string]int{}
	var next *string
	for {
		out, err := b.ec2.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{NextToken: next})
		if err != nil {
			return 0, err
		}
		for _, s := range out.Subnets {
			byVPC[awssdk.ToString(s.VpcId)]++
		}
		if out.NextToken == nil {
			break
		}
		next = out.NextToken
	}
	return float64(maxInt(byVPC)), nil
}

func (b *Builder) countNetworkInterfaces(ctx context.Context) (float64, error) {
	n := 0
	var next *string
	for {
		out, err := b.ec2.DescribeNetworkInterfaces(ctx, &ec2.DescribeNetworkInterfacesInput{NextToken: next})
		if err != nil {
			return 0, err
		}
		n += len(out.NetworkInterfaces)
		if out.NextToken == nil {
			break
		}
		next = out.NextToken
	}
	return float64(n), nil
}

// countSecurityGroupsMaxPerENI returns the largest number of security groups
// attached to any one network interface -- what the per-NIC SG quota limits
// (LaForge attaches a per-object SG, so a multi-homed box can approach it).
func (b *Builder) countSecurityGroupsMaxPerENI(ctx context.Context) (float64, error) {
	most := 0
	var next *string
	for {
		out, err := b.ec2.DescribeNetworkInterfaces(ctx, &ec2.DescribeNetworkInterfacesInput{NextToken: next})
		if err != nil {
			return 0, err
		}
		for _, ni := range out.NetworkInterfaces {
			if len(ni.Groups) > most {
				most = len(ni.Groups)
			}
		}
		if out.NextToken == nil {
			break
		}
		next = out.NextToken
	}
	return float64(most), nil
}

// countRunningStandardVCPUs sums the vCPUs of all running On-Demand Standard
// (A, C, D, H, I, M, R, T, Z) instances account-wide -- the usage the standard
// running-instance vCPU quota caps. Instance-type vCPU counts come from the same
// lookup the pre-deploy readiness check uses.
func (b *Builder) countRunningStandardVCPUs(ctx context.Context) (float64, error) {
	var types map[string]bool = map[string]bool{}
	var running []ec2types.InstanceType
	var next *string
	for {
		out, err := b.ec2.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
			Filters:   []ec2types.Filter{filter("instance-state-name", "running")},
			NextToken: next,
		})
		if err != nil {
			return 0, err
		}
		for _, r := range out.Reservations {
			for _, i := range r.Instances {
				if isStandardInstanceFamily(string(i.InstanceType)) {
					running = append(running, i.InstanceType)
					types[string(i.InstanceType)] = true
				}
			}
		}
		if out.NextToken == nil {
			break
		}
		next = out.NextToken
	}
	vcpu := b.describeVCPUs(ctx, keys(types))
	total := 0
	for _, t := range running {
		total += vcpu[string(t)]
	}
	return float64(total), nil
}

// isStandardInstanceFamily reports whether an instance type belongs to the
// Standard family set the running-on-demand vCPU quota counts. Keyed on the
// leading letter; the accelerated/HPC/high-memory families (g, p, f, inf, trn,
// u, hpc, ...) have their own separate quotas and are excluded. hpc* also starts
// with 'h' but is rare in LaForge content; this is an advisory gauge, not a gate.
func isStandardInstanceFamily(instanceType string) bool {
	if instanceType == "" {
		return false
	}
	switch instanceType[0] {
	case 'a', 'c', 'd', 'h', 'i', 'm', 'r', 't', 'z':
		return true
	}
	return false
}

func maxInt(m map[string]int) int {
	most := 0
	for _, v := range m {
		if v > most {
			most = v
		}
	}
	return most
}
