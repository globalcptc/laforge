// Pre-deploy readiness check. AWS is the one hoster where an account quota
// silently breaks a deploy mid-flight -- the region VPC limit (one per team), the
// Elastic IP limit (one per public: host), the running-instance vCPU limit. This
// computes what a build will need from a builder.Demand and compares it to the
// account's real limits + current usage, up front, so the operator sees "this
// build needs 8 VPCs but the region allows 5" before deploying rather than a 400
// on the 6th CreateVpc. Advisory only -- it warns, it never blocks. The sibling
// of the IAM pre-flight check (preflight.go). See docs/plans/builder-monitoring.md.
package aws

import (
	"context"
	"fmt"
	"sort"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"github.com/globalcptc/laforge/internal/builder"
)

// Compile-time proof the builder answers the pre-deploy readiness probe.
var _ builder.ReadinessChecker = (*Builder)(nil)

// Account quota codes (same service/code pairs Service Quotas uses). VPCs and
// EIPs match the gauges in stats.go; the vCPU quota is the Standard-family
// running-on-demand limit a fleet of hosts hits first.
const (
	quotaVPCs  = "L-F678F1CE" // vpc:   VPCs per region
	quotaEIPs  = "L-0263D0A3" // ec2:   Elastic IPs per region
	quotaVCPUs = "L-1216C47A" // ec2:   Running On-Demand Standard (A,C,D,H,I,M,R,T,Z) instances, in vCPUs
)

func (b *Builder) CheckReadiness(ctx context.Context, d builder.Demand) (builder.Readiness, error) {
	var r builder.Readiness

	// Resolve each requested size to its instance type and that type's vCPUs, so
	// the host fleet's vCPU demand can be compared to the region limit. An unknown
	// size is a configuration gap the operator should see here.
	vcpuByType, typeWarnings := b.vcpusForSizes(ctx, d.InstanceCounts)
	r.Warnings = append(r.Warnings, typeWarnings...)
	wantVCPUs := 0
	for size, n := range d.InstanceCounts {
		if t, ok := b.cfg.Sizes[size]; ok {
			wantVCPUs += n * vcpuByType[t]
		}
	}

	// Current usage of each quota (best-effort per quota: a failed read warns and
	// that dimension is simply not checked, like the stats gauges).
	usage := awsReadinessUsage{}
	if vpcs, err := b.ec2.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{}); err != nil {
		r.Warnings = append(r.Warnings, "couldn't read current VPC count: "+errSummary(err))
	} else {
		usage.vpcs = len(vpcs.Vpcs)
	}
	if addrs, err := b.ec2.DescribeAddresses(ctx, &ec2.DescribeAddressesInput{}); err != nil {
		r.Warnings = append(r.Warnings, "couldn't read current Elastic IP count: "+errSummary(err))
	} else {
		usage.eips = len(addrs.Addresses)
	}
	if running, err := b.runningManagedVCPUs(ctx, vcpuByType); err != nil {
		r.Warnings = append(r.Warnings, "couldn't read current vCPU usage: "+errSummary(err))
	} else {
		usage.vcpus = running
	}

	limits := awsReadinessLimits{}
	limits.vpcs, _ = b.quotaLimit(ctx, "vpc", quotaVPCs)
	limits.eips, _ = b.quotaLimit(ctx, "ec2", quotaEIPs)
	limits.vcpus, _ = b.quotaLimit(ctx, "ec2", quotaVCPUs)

	r.Warnings = append(r.Warnings, evalReadiness(d.Teams, d.PublicHosts, wantVCPUs, usage, limits)...)
	return r, nil
}

type awsReadinessUsage struct {
	vpcs, eips, vcpus int
}

// awsReadinessLimits holds each quota's limit; 0 means "unknown" (the Service
// Quotas read failed) and that dimension is skipped so a missing
// servicequotas:GetServiceQuota permission never produces a false warning.
type awsReadinessLimits struct {
	vpcs, eips, vcpus float64
}

// evalReadiness is the pure comparison: for each dimension with a known limit,
// warn when current usage plus this build's demand would exceed it. Kept pure so
// it's unit-testable without any AWS call.
func evalReadiness(wantVPCs, wantEIPs, wantVCPUs int, usage awsReadinessUsage, limits awsReadinessLimits) []string {
	var w []string
	add := func(limit float64, have, want int, unit, noun string) {
		if limit <= 0 || want <= 0 {
			return // limit unknown, or this build needs none of this resource
		}
		if float64(have+want) > limit {
			w = append(w, fmt.Sprintf(
				"this build needs %d more %s (%s %d in use, region limit %d); it will exceed the quota mid-deploy -- raise it in Service Quotas or reduce the build",
				want, noun, unitInUse(unit), have, int(limit)))
		}
	}
	add(limits.vpcs, usage.vpcs, wantVPCs, "vpc", "VPCs (one per team)")
	add(limits.eips, usage.eips, wantEIPs, "eip", "Elastic IPs (one per public host)")
	add(limits.vcpus, usage.vcpus, wantVCPUs, "vcpu", "instance vCPUs")
	return w
}

func unitInUse(unit string) string {
	switch unit {
	case "vcpu":
		return "vCPUs already"
	default:
		return "with"
	}
}

// vcpusForSizes resolves the distinct instance types behind the requested sizes
// and looks up each type's default vCPU count in ONE DescribeInstanceTypes call.
// Returns type -> vCPUs, plus a warning per size that maps to no configured
// instance type (so a typo in the size map surfaces here, not at RunInstances).
func (b *Builder) vcpusForSizes(ctx context.Context, counts map[string]int) (map[string]int, []string) {
	types := map[string]bool{}
	var warnings []string
	for size := range counts {
		t, ok := b.cfg.Sizes[size]
		if !ok || t == "" {
			warnings = append(warnings, fmt.Sprintf("size %q has no AWS instance type configured, so its vCPU demand can't be checked", size))
			continue
		}
		types[t] = true
	}
	return b.describeVCPUs(ctx, keys(types)), warnings
}

// runningManagedVCPUs sums the vCPUs of the builder's own running instances --
// the LaForge footprint already counting against the region vCPU quota. Other
// (non-LaForge) instances also consume it, which the warning wording accounts
// for; this at least catches a build that would blow past the limit on top of
// LaForge's existing fleet.
func (b *Builder) runningManagedVCPUs(ctx context.Context, known map[string]int) (int, error) {
	insts, err := b.ec2.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		Filters: []ec2types.Filter{managedFilter, filter("instance-state-name", "running")},
	})
	if err != nil {
		return 0, err
	}
	// A running instance may be a type not in the size map (e.g. an older build's
	// size that was since changed); fetch any such types too so the sum is honest.
	need := map[string]bool{}
	for _, res := range insts.Reservations {
		for _, i := range res.Instances {
			t := string(i.InstanceType)
			if _, ok := known[t]; !ok {
				need[t] = true
			}
		}
	}
	if extra := b.describeVCPUs(ctx, keys(need)); len(extra) > 0 {
		for t, v := range extra {
			known[t] = v
		}
	}
	total := 0
	for _, res := range insts.Reservations {
		for _, i := range res.Instances {
			total += known[string(i.InstanceType)]
		}
	}
	return total, nil
}

// describeVCPUs looks up the default vCPU count for each instance type in one
// call. Best-effort: a failed/absent lookup just leaves that type out of the map
// (its demand then counts as 0 rather than failing the whole check).
func (b *Builder) describeVCPUs(ctx context.Context, types []string) map[string]int {
	out := map[string]int{}
	if len(types) == 0 {
		return out
	}
	its := make([]ec2types.InstanceType, len(types))
	for i, t := range types {
		its[i] = ec2types.InstanceType(t)
	}
	resp, err := b.ec2.DescribeInstanceTypes(ctx, &ec2.DescribeInstanceTypesInput{InstanceTypes: its})
	if err != nil {
		return out
	}
	for _, it := range resp.InstanceTypes {
		if it.VCpuInfo != nil && it.VCpuInfo.DefaultVCpus != nil {
			out[string(it.InstanceType)] = int(awssdk.ToInt32(it.VCpuInfo.DefaultVCpus))
		}
	}
	return out
}

func keys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
