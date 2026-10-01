// Package aws is a DRAFT builder.Builder backed by real EC2/VPC calls via
// aws-sdk-go-v2. It is written to the same contract every other builder
// satisfies (internal/builder.Builder) and follows the same conventions --
// deterministic ExternalName-based "ensure" semantics, team-tagged
// resources, power/access as infrastructure operations -- but it has NOT
// been exercised against a real AWS account: there is no test infrastructure
// for it yet. Treat every method here as a reviewed starting point, not a
// proven path. Places that need a real account decision (a NAT/IGW for
// genuine egress, Windows password retrieval, Route 53 for DNS) are marked
// TODO(untested) rather than guessed at.
//
// Mapping from LaForge concepts to AWS:
//   - network   -> a VPC + one subnet with the network's CIDR
//   - host      -> an EC2 instance in that subnet
//   - container -> not yet implemented here; containers on AWS are ECS/Fargate
//     (a different shape from EC2). DeployContainer returns an error until that's
//     built -- there is no capability opt-out, so this is a DRAFT gap, not a
//     declared "can't"
//   - team access -> a per-team security group whose ingress rules are
//     added on OpenAccess and revoked on CloseAccess
//
// Every resource is tagged laforgeExternalNameTag=<ExternalName> and
// laforgeTeamTag=<team>, which is how "ensure" (adopt-or-create) and Inspect
// both find what already exists.
package aws

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
	"strings"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/ecs"

	"github.com/globalcptc/laforge/internal/builder"
)

const (
	laforgeExternalNameTag = "laforge:external-name"
	laforgeTeamTag         = "laforge:team"
	laforgeManagedTag      = "laforge:managed" // "true" on everything we create, for Inspect's filter
)

// Config is what the resolver hands the builder. Credentials and region come
// from the standard AWS chain (env, shared config, instance profile) via
// LoadDefaultConfig, so nothing secret is stored here; Region overrides the
// chain's region when set. Images maps a LaForge `os` name to an AMI id;
// Sizes maps a LaForge `size` name to an EC2 instance type.
type Config struct {
	Region string
	Images map[string]string // os name -> AMI id
	Sizes  map[string]string // size name -> instance type (e.g. "t3.medium")

	// Fargate (container) settings, read from the environment in New when unset,
	// since containers on AWS run as ECS/Fargate tasks (see fargate.go). All are
	// account-specific infrastructure, hence env-configured like the credentials.
	Cluster          string // ECS cluster (AWS_ECS_CLUSTER, default "default")
	ExecutionRoleARN string // task execution role (AWS_ECS_EXECUTION_ROLE_ARN) -- Fargate needs it to pull images and write logs
	LogGroup         string // CloudWatch log group for task logs (AWS_ECS_LOG_GROUP), optional
	AgentInitImage   string // public image the agent-download init container runs (AWS_ECS_AGENT_INIT_IMAGE, default "alpine:3.20")
}

// Builder is the draft EC2/VPC + ECS/Fargate builder.
type Builder struct {
	ec2 *ec2.Client
	ecs *ecs.Client
	cfg Config
}

// New constructs the builder from the ambient AWS credential/region chain.
func New(ctx context.Context, cfg Config) (*Builder, error) {
	opts := []func(*awsconfig.LoadOptions) error{}
	if cfg.Region != "" {
		opts = append(opts, awsconfig.WithRegion(cfg.Region))
	}
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("loading AWS config: %w", err)
	}
	if cfg.Cluster == "" {
		cfg.Cluster = envOr("AWS_ECS_CLUSTER", "default")
	}
	if cfg.ExecutionRoleARN == "" {
		cfg.ExecutionRoleARN = os.Getenv("AWS_ECS_EXECUTION_ROLE_ARN")
	}
	if cfg.LogGroup == "" {
		cfg.LogGroup = os.Getenv("AWS_ECS_LOG_GROUP")
	}
	if cfg.AgentInitImage == "" {
		cfg.AgentInitImage = envOr("AWS_ECS_AGENT_INIT_IMAGE", "alpine:3.20")
	}
	return &Builder{ec2: ec2.NewFromConfig(awsCfg), ecs: ecs.NewFromConfig(awsCfg), cfg: cfg}, nil
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// tagSpec builds the standard LaForge tag set for a resource type.
func (b *Builder) tagSpec(rt ec2types.ResourceType, externalName, team, displayName string) ec2types.TagSpecification {
	tags := []ec2types.Tag{
		{Key: awssdk.String(laforgeManagedTag), Value: awssdk.String("true")},
		{Key: awssdk.String(laforgeExternalNameTag), Value: awssdk.String(externalName)},
		{Key: awssdk.String(laforgeTeamTag), Value: awssdk.String(team)},
	}
	if displayName != "" {
		tags = append(tags, ec2types.Tag{Key: awssdk.String("Name"), Value: awssdk.String(displayName)})
	}
	return ec2types.TagSpecification{ResourceType: rt, Tags: tags}
}

func nameFilter(externalName string) ec2types.Filter {
	return ec2types.Filter{Name: awssdk.String("tag:" + laforgeExternalNameTag), Values: []string{externalName}}
}

// --- Networks -------------------------------------------------------------

func (b *Builder) DeployNetwork(ctx context.Context, spec builder.NetworkSpec) (string, error) {
	// Ensure: a VPC already tagged with this ExternalName is adopted.
	if id, err := b.findVPC(ctx, spec.ExternalName); err != nil {
		return "", err
	} else if id != "" {
		return id, nil
	}
	if spec.CIDR == "" {
		return "", fmt.Errorf("network %q has no CIDR -- AWS requires one for the VPC", spec.ExternalName)
	}
	vpc, err := b.ec2.CreateVpc(ctx, &ec2.CreateVpcInput{
		CidrBlock:         awssdk.String(spec.CIDR),
		TagSpecifications: []ec2types.TagSpecification{b.tagSpec(ec2types.ResourceTypeVpc, spec.ExternalName, spec.Team, spec.DisplayName)},
	})
	if err != nil {
		return "", fmt.Errorf("creating VPC for %q: %w", spec.ExternalName, err)
	}
	vpcID := awssdk.ToString(vpc.Vpc.VpcId)
	// One subnet spanning the whole CIDR -- enough for a flat team network;
	// multi-subnet topologies aren't something LaForge content expresses.
	if _, err := b.ec2.CreateSubnet(ctx, &ec2.CreateSubnetInput{
		VpcId:             awssdk.String(vpcID),
		CidrBlock:         awssdk.String(spec.CIDR),
		TagSpecifications: []ec2types.TagSpecification{b.tagSpec(ec2types.ResourceTypeSubnet, spec.ExternalName, spec.Team, spec.DisplayName)},
	}); err != nil {
		return "", fmt.Errorf("creating subnet for %q: %w", spec.ExternalName, err)
	}
	// TODO(untested): for genuine external ingress/egress this VPC also needs
	// an internet gateway, a route table, and (for private hosts) a NAT --
	// deferred until there's an account to validate the routing against.
	return vpcID, nil
}

func (b *Builder) DestroyNetwork(ctx context.Context, team, externalRef string) error {
	// externalRef is the VPC id. Delete its subnets first, then the VPC.
	subnets, err := b.ec2.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{
		Filters: []ec2types.Filter{{Name: awssdk.String("vpc-id"), Values: []string{externalRef}}},
	})
	if err == nil {
		for _, s := range subnets.Subnets {
			_, _ = b.ec2.DeleteSubnet(ctx, &ec2.DeleteSubnetInput{SubnetId: s.SubnetId})
		}
	}
	if _, err := b.ec2.DeleteVpc(ctx, &ec2.DeleteVpcInput{VpcId: awssdk.String(externalRef)}); err != nil {
		if isNotFound(err) {
			return nil // ensure-destroy: already gone is success
		}
		return fmt.Errorf("deleting VPC %s: %w", externalRef, err)
	}
	return nil
}

func (b *Builder) findVPC(ctx context.Context, externalName string) (string, error) {
	out, err := b.ec2.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{Filters: []ec2types.Filter{nameFilter(externalName)}})
	if err != nil {
		return "", fmt.Errorf("looking up VPC %q: %w", externalName, err)
	}
	if len(out.Vpcs) > 0 {
		return awssdk.ToString(out.Vpcs[0].VpcId), nil
	}
	return "", nil
}

func (b *Builder) subnetForNetwork(ctx context.Context, networkExternalName string) (string, error) {
	out, err := b.ec2.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{Filters: []ec2types.Filter{nameFilter(networkExternalName)}})
	if err != nil {
		return "", fmt.Errorf("looking up subnet for network %q: %w", networkExternalName, err)
	}
	if len(out.Subnets) == 0 {
		return "", fmt.Errorf("no subnet found for network %q -- its DeployNetwork may not have run yet", networkExternalName)
	}
	return awssdk.ToString(out.Subnets[0].SubnetId), nil
}

// --- Hosts ----------------------------------------------------------------

func (b *Builder) DeployHost(ctx context.Context, spec builder.HostSpec) (string, error) {
	if id, err := b.findInstance(ctx, spec.ExternalName); err != nil {
		return "", err
	} else if id != "" {
		return id, nil // ensure: adopt an already-created instance
	}

	ami, ok := b.cfg.Images[spec.OS]
	if !ok {
		return "", fmt.Errorf("no AMI mapping for os %q (configure the builder's image map)", spec.OS)
	}
	instanceType, ok := b.cfg.Sizes[spec.Size]
	if !ok {
		return "", fmt.Errorf("no instance-type mapping for size %q (configure the builder's size map)", spec.Size)
	}
	subnetID, err := b.subnetForNetwork(ctx, spec.Network)
	if err != nil {
		return "", err
	}
	sgID, err := b.ensureTeamSecurityGroup(ctx, spec.Team, subnetID)
	if err != nil {
		return "", err
	}

	in := &ec2.RunInstancesInput{
		ImageId:          awssdk.String(ami),
		InstanceType:     ec2types.InstanceType(instanceType),
		MinCount:         awssdk.Int32(1),
		MaxCount:         awssdk.Int32(1),
		SubnetId:         awssdk.String(subnetID),
		SecurityGroupIds: []string{sgID},
		TagSpecifications: []ec2types.TagSpecification{
			b.tagSpec(ec2types.ResourceTypeInstance, spec.ExternalName, spec.Team, spec.DisplayName),
		},
	}
	if spec.Address != "" {
		in.PrivateIpAddress = awssdk.String(spec.Address)
	}
	if spec.CloudInitUserData != "" {
		// EC2 user-data must be base64; cloud-init on Linux and cloudbase-init
		// on Windows both consume it the same way the Incus builder's
		// cloud-init.user-data key does.
		in.UserData = awssdk.String(base64.StdEncoding.EncodeToString([]byte(spec.CloudInitUserData)))
	}
	out, err := b.ec2.RunInstances(ctx, in)
	if err != nil {
		return "", fmt.Errorf("launching instance for %q: %w", spec.ExternalName, err)
	}
	if len(out.Instances) == 0 {
		return "", fmt.Errorf("RunInstances for %q returned no instance", spec.ExternalName)
	}
	return awssdk.ToString(out.Instances[0].InstanceId), nil
}

// DeployContainer / DestroyContainer live in fargate.go (containers on AWS are
// ECS/Fargate tasks, a different shape from EC2 VMs).

func (b *Builder) DestroyHost(ctx context.Context, team, externalRef string) error {
	_, err := b.ec2.TerminateInstances(ctx, &ec2.TerminateInstancesInput{InstanceIds: []string{externalRef}})
	if err != nil && !isNotFound(err) {
		return fmt.Errorf("terminating instance %s: %w", externalRef, err)
	}
	return nil
}

func (b *Builder) findInstance(ctx context.Context, externalName string) (string, error) {
	out, err := b.ec2.DescribeInstances(ctx, &ec2.DescribeInstancesInput{Filters: []ec2types.Filter{
		nameFilter(externalName),
		{Name: awssdk.String("instance-state-name"), Values: []string{"pending", "running", "stopping", "stopped"}},
	}})
	if err != nil {
		return "", fmt.Errorf("looking up instance %q: %w", externalName, err)
	}
	for _, r := range out.Reservations {
		for _, inst := range r.Instances {
			return awssdk.ToString(inst.InstanceId), nil
		}
	}
	return "", nil
}

// --- Inspect --------------------------------------------------------------

func (b *Builder) Inspect(ctx context.Context) ([]builder.Resource, error) {
	managed := ec2types.Filter{Name: awssdk.String("tag:" + laforgeManagedTag), Values: []string{"true"}}
	var out []builder.Resource

	insts, err := b.ec2.DescribeInstances(ctx, &ec2.DescribeInstancesInput{Filters: []ec2types.Filter{managed}})
	if err != nil {
		return nil, fmt.Errorf("describing instances: %w", err)
	}
	for _, r := range insts.Reservations {
		for _, inst := range r.Instances {
			out = append(out, builder.Resource{
				ExternalRef: awssdk.ToString(inst.InstanceId),
				Kind:        "host",
				State:       mapInstanceState(inst.State),
			})
		}
	}

	vpcs, err := b.ec2.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{Filters: []ec2types.Filter{managed}})
	if err != nil {
		return nil, fmt.Errorf("describing VPCs: %w", err)
	}
	for _, v := range vpcs.Vpcs {
		out = append(out, builder.Resource{ExternalRef: awssdk.ToString(v.VpcId), Kind: "network"})
	}
	return out, nil
}

func mapInstanceState(s *ec2types.InstanceState) string {
	if s == nil {
		return builder.PowerStateOther
	}
	switch s.Name {
	case ec2types.InstanceStateNameRunning:
		return builder.PowerStateRunning
	case ec2types.InstanceStateNameStopped:
		return builder.PowerStateStopped
	default:
		return builder.PowerStateOther
	}
}

// --- Access (per-team security group) -------------------------------------

// ensureTeamSecurityGroup returns the id of this team's security group in the
// subnet's VPC, creating it if absent. Ingress rules are managed by
// Open/CloseAccess; a freshly created group has no ingress (closed by
// default), which is the safe starting state.
func (b *Builder) ensureTeamSecurityGroup(ctx context.Context, team, subnetID string) (string, error) {
	sub, err := b.ec2.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{SubnetIds: []string{subnetID}})
	if err != nil || len(sub.Subnets) == 0 {
		return "", fmt.Errorf("resolving VPC for subnet %s: %w", subnetID, err)
	}
	vpcID := awssdk.ToString(sub.Subnets[0].VpcId)

	name := "laforge-team-" + team
	existing, err := b.ec2.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{Filters: []ec2types.Filter{
		{Name: awssdk.String("group-name"), Values: []string{name}},
		{Name: awssdk.String("vpc-id"), Values: []string{vpcID}},
	}})
	if err == nil && len(existing.SecurityGroups) > 0 {
		return awssdk.ToString(existing.SecurityGroups[0].GroupId), nil
	}
	created, err := b.ec2.CreateSecurityGroup(ctx, &ec2.CreateSecurityGroupInput{
		GroupName:   awssdk.String(name),
		Description: awssdk.String("LaForge team " + team + " ingress control"),
		VpcId:       awssdk.String(vpcID),
		TagSpecifications: []ec2types.TagSpecification{
			b.tagSpec(ec2types.ResourceTypeSecurityGroup, "team-"+team, team, name),
		},
	})
	if err != nil {
		return "", fmt.Errorf("creating security group for team %s: %w", team, err)
	}
	return awssdk.ToString(created.GroupId), nil
}

// teamSecurityGroups finds every LaForge security group for a team (across
// VPCs), for Open/CloseAccess to act on all of them.
func (b *Builder) teamSecurityGroups(ctx context.Context, team string) ([]string, error) {
	out, err := b.ec2.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{Filters: []ec2types.Filter{
		{Name: awssdk.String("tag:" + laforgeTeamTag), Values: []string{team}},
		{Name: awssdk.String("tag:" + laforgeManagedTag), Values: []string{"true"}},
	}})
	if err != nil {
		return nil, fmt.Errorf("listing team %s security groups: %w", team, err)
	}
	ids := make([]string, 0, len(out.SecurityGroups))
	for _, g := range out.SecurityGroups {
		ids = append(ids, awssdk.ToString(g.GroupId))
	}
	return ids, nil
}

// allIngress is the rule Open/CloseAccess adds/removes: allow all inbound.
// The real event policy would scope this to scored ports; a draft opens
// everything so "open == reachable" is unambiguous.
func allIngress() []ec2types.IpPermission {
	return []ec2types.IpPermission{{
		IpProtocol: awssdk.String("-1"),
		IpRanges:   []ec2types.IpRange{{CidrIp: awssdk.String("0.0.0.0/0")}},
	}}
}

func (b *Builder) OpenAccess(ctx context.Context, team string) error {
	sgs, err := b.teamSecurityGroups(ctx, team)
	if err != nil {
		return err
	}
	for _, sg := range sgs {
		_, err := b.ec2.AuthorizeSecurityGroupIngress(ctx, &ec2.AuthorizeSecurityGroupIngressInput{
			GroupId: awssdk.String(sg), IpPermissions: allIngress(),
		})
		if err != nil && !isDuplicate(err) {
			return fmt.Errorf("opening access on %s: %w", sg, err)
		}
	}
	return nil
}

func (b *Builder) CloseAccess(ctx context.Context, team string) error {
	sgs, err := b.teamSecurityGroups(ctx, team)
	if err != nil {
		return err
	}
	for _, sg := range sgs {
		_, err := b.ec2.RevokeSecurityGroupIngress(ctx, &ec2.RevokeSecurityGroupIngressInput{
			GroupId: awssdk.String(sg), IpPermissions: allIngress(),
		})
		if err != nil && !isNotFound(err) {
			return fmt.Errorf("closing access on %s: %w", sg, err)
		}
	}
	// NOTE(untested): revoking ingress blocks NEW connections immediately;
	// AWS security groups are stateful, so already-established flows can
	// linger until they idle out. "Terminate established connections" (the
	// contract's requirement) would need a Network ACL deny or forcing the
	// instances' interfaces down -- deferred until testable.
	return nil
}

// --- Power ----------------------------------------------------------------

func (b *Builder) PowerAction(ctx context.Context, team, externalRef, action string, force bool) error {
	switch action {
	case builder.PowerStart:
		_, err := b.ec2.StartInstances(ctx, &ec2.StartInstancesInput{InstanceIds: []string{externalRef}})
		return wrapPower(action, externalRef, err)
	case builder.PowerStop:
		_, err := b.ec2.StopInstances(ctx, &ec2.StopInstancesInput{InstanceIds: []string{externalRef}, Force: awssdk.Bool(force)})
		return wrapPower(action, externalRef, err)
	case builder.PowerReboot:
		// EC2 RebootInstances is always the guest-cooperative reboot; a hard
		// reset is a stop(force)+start, which we do when force is set.
		if force {
			if _, err := b.ec2.StopInstances(ctx, &ec2.StopInstancesInput{InstanceIds: []string{externalRef}, Force: awssdk.Bool(true)}); err != nil {
				return wrapPower(action, externalRef, err)
			}
			_, err := b.ec2.StartInstances(ctx, &ec2.StartInstancesInput{InstanceIds: []string{externalRef}})
			return wrapPower(action, externalRef, err)
		}
		_, err := b.ec2.RebootInstances(ctx, &ec2.RebootInstancesInput{InstanceIds: []string{externalRef}})
		return wrapPower(action, externalRef, err)
	default:
		return fmt.Errorf("unknown power action %q", action)
	}
}

func wrapPower(action, ref string, err error) error {
	if err != nil {
		return fmt.Errorf("power %s on %s: %w", action, ref, err)
	}
	return nil
}

// --- Network access (visible_from) ----------------------------------------

// ConfigureNetworkAccess enforces a team's `visible_from` + `ports:` policy with
// per-host security groups -- the AWS analogue of the Incus/MicroCloud per-host
// NIC ACL (see internal/builder/incus's proven
// mechanics). For each host it ensures a security group whose inbound rules allow
// exactly the host's declared TCP/UDP ports, and only from its own network's CIDR
// (same-network siblings) plus the CIDRs of its `visible_from` networks
// (cross-network); it then makes that SG the instance's group. A host that
// declared no ports gets an SG with no inbound rules -- reachable on nothing.
// Because security groups are stateful and evaluated per-ENI, this filters
// same-subnet traffic too, so no separate same-network handling is needed (unlike
// OVN's network ACLs).
//
// DRAFT: never run against a real account. Two known gaps, both documented here
// rather than guessed at:
//   - Cross-network reachability still needs ROUTING between the team's networks.
//     Today each network is its own VPC (see DeployNetwork), so cross-network
//     traffic needs VPC peering + route-table entries, which this does not yet
//     create -- the intended end state is one VPC per team with a subnet per
//     network, which route for free. Until that exists, the visible_from CIDR
//     rules are correct but cross-network packets won't route to be evaluated.
//   - This makes the per-host SG the instance's authoritative group, so the
//     legacy per-team OpenAccess/CloseAccess SG no longer governs these hosts.
//     Reconciling the access schedule with the port firewall on AWS is an open
//     design question (the OVN builders keep them separate via NIC removal).
func (b *Builder) ConfigureNetworkAccess(ctx context.Context, team string, networks []builder.NetworkAccess) error {
	cidrByName := make(map[string]string, len(networks))
	for _, na := range networks {
		cidrByName[na.ExternalName] = na.CIDR
	}
	for _, na := range networks {
		// Allowed sources: this host's own network (same-net) + every
		// visible_from network's CIDR (cross-net). A visible_from naming a
		// network not in this set is a content error the loader already reports.
		sources := []string{na.CIDR}
		for _, from := range na.VisibleFrom {
			if c := cidrByName[from]; c != "" {
				sources = append(sources, c)
			}
		}
		for _, h := range na.Hosts {
			if h.ExternalRef == "" {
				continue // not yet deployed; a later pass will configure it
			}
			if err := b.ensureHostFirewall(ctx, team, h, sources); err != nil {
				return fmt.Errorf("configuring host firewall for %s (team %s): %w", h.ExternalRef, team, err)
			}
		}
	}
	return nil
}

// ensureHostFirewall creates/updates the host's own security group (inbound =
// its declared ports from the allowed source CIDRs, default-deny otherwise) and
// makes it the instance's group. Idempotent: rules are replaced each pass so an
// edited policy converges.
func (b *Builder) ensureHostFirewall(ctx context.Context, team string, h builder.HostAccess, sources []string) error {
	// Resolve the instance's VPC (a security group is VPC-scoped).
	desc, err := b.ec2.DescribeInstances(ctx, &ec2.DescribeInstancesInput{InstanceIds: []string{h.ExternalRef}})
	if err != nil {
		return fmt.Errorf("describing instance %s: %w", h.ExternalRef, err)
	}
	var vpcID string
	for _, r := range desc.Reservations {
		for _, inst := range r.Instances {
			vpcID = awssdk.ToString(inst.VpcId)
		}
	}
	if vpcID == "" {
		return fmt.Errorf("instance %s has no VPC yet", h.ExternalRef)
	}

	name := "laforge-hostfw-" + h.ExternalRef
	sgID, err := b.ensureNamedSecurityGroup(ctx, team, vpcID, name, "LaForge host firewall for "+h.ExternalRef)
	if err != nil {
		return err
	}

	// Converge the inbound rules: revoke whatever's there, then authorize the
	// current declared-port-from-allowed-source set. Revoke of an empty/foreign
	// rule set is a no-op we tolerate.
	if cur, err := b.securityGroupIngress(ctx, sgID); err == nil && len(cur) > 0 {
		_, _ = b.ec2.RevokeSecurityGroupIngress(ctx, &ec2.RevokeSecurityGroupIngressInput{
			GroupId: awssdk.String(sgID), IpPermissions: cur,
		})
	}
	perms := portPermissions(sources, h.TCPPorts, h.UDPPorts)
	if len(perms) > 0 {
		if _, err := b.ec2.AuthorizeSecurityGroupIngress(ctx, &ec2.AuthorizeSecurityGroupIngressInput{
			GroupId: awssdk.String(sgID), IpPermissions: perms,
		}); err != nil && !isDuplicate(err) {
			return fmt.Errorf("authorizing host firewall rules on %s: %w", sgID, err)
		}
	}

	// Make this SG the instance's group, so the firewall is what governs it.
	if _, err := b.ec2.ModifyInstanceAttribute(ctx, &ec2.ModifyInstanceAttributeInput{
		InstanceId: awssdk.String(h.ExternalRef), Groups: []string{sgID},
	}); err != nil {
		return fmt.Errorf("attaching host firewall SG to %s: %w", h.ExternalRef, err)
	}
	return nil
}

// ensureNamedSecurityGroup adopts or creates a security group by name in a VPC.
func (b *Builder) ensureNamedSecurityGroup(ctx context.Context, team, vpcID, name, desc string) (string, error) {
	existing, err := b.ec2.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{Filters: []ec2types.Filter{
		{Name: awssdk.String("group-name"), Values: []string{name}},
		{Name: awssdk.String("vpc-id"), Values: []string{vpcID}},
	}})
	if err == nil && len(existing.SecurityGroups) > 0 {
		return awssdk.ToString(existing.SecurityGroups[0].GroupId), nil
	}
	created, err := b.ec2.CreateSecurityGroup(ctx, &ec2.CreateSecurityGroupInput{
		GroupName:   awssdk.String(name),
		Description: awssdk.String(desc),
		VpcId:       awssdk.String(vpcID),
		TagSpecifications: []ec2types.TagSpecification{
			b.tagSpec(ec2types.ResourceTypeSecurityGroup, name, team, name),
		},
	})
	if err != nil {
		return "", fmt.Errorf("creating security group %s: %w", name, err)
	}
	return awssdk.ToString(created.GroupId), nil
}

// securityGroupIngress returns a security group's current inbound permissions.
func (b *Builder) securityGroupIngress(ctx context.Context, sgID string) ([]ec2types.IpPermission, error) {
	out, err := b.ec2.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{GroupIds: []string{sgID}})
	if err != nil || len(out.SecurityGroups) == 0 {
		return nil, err
	}
	return out.SecurityGroups[0].IpPermissions, nil
}

// portPermissions turns declared TCP/UDP ports and allowed source CIDRs into EC2
// ingress permissions: one permission per port (range) per protocol, from all
// the source CIDRs. An empty port set yields nothing (default-deny).
func portPermissions(sources []string, tcp, udp []string) []ec2types.IpPermission {
	ranges := make([]ec2types.IpRange, 0, len(sources))
	for _, s := range sources {
		ranges = append(ranges, ec2types.IpRange{CidrIp: awssdk.String(s)})
	}
	var perms []ec2types.IpPermission
	add := func(proto string, ports []string) {
		for _, p := range ports {
			from, to, ok := parsePortRange(p)
			if !ok {
				continue
			}
			perms = append(perms, ec2types.IpPermission{
				IpProtocol: awssdk.String(proto),
				FromPort:   awssdk.Int32(from),
				ToPort:     awssdk.Int32(to),
				IpRanges:   ranges,
			})
		}
	}
	add("tcp", tcp)
	add("udp", udp)
	return perms
}

// parsePortRange parses "80" or "8000-8100" into an inclusive [from,to]. Returns
// ok=false for anything it can't parse, which the caller skips.
func parsePortRange(p string) (int32, int32, bool) {
	p = strings.TrimSpace(p)
	if lo, hi, found := strings.Cut(p, "-"); found {
		l, err1 := strconv.Atoi(strings.TrimSpace(lo))
		h, err2 := strconv.Atoi(strings.TrimSpace(hi))
		if err1 != nil || err2 != nil || l < 0 || h > 65535 || l > h {
			return 0, 0, false
		}
		return int32(l), int32(h), true
	}
	n, err := strconv.Atoi(p)
	if err != nil || n < 0 || n > 65535 {
		return 0, 0, false
	}
	return int32(n), int32(n), true
}

// --- DNS ------------------------------------------------------------------

// Compile-time check that the draft satisfies the full Builder contract.
// ConfigureExternalAccess (DRAFT) would make each host reachable from outside
// the environment on its `public:` ports. On AWS this is the simple case the
// per-builder design calls out: associate a public IP (or Elastic IP) with the
// instance and authorize those ports in its security group, so the endpoint is
// just <public-ip>:<port> with no remap. Not implemented yet -- returns a clear
// error rather than silently exposing nothing, same honest-DRAFT stance as this
// builder's security-group network access.
func (b *Builder) ConfigureExternalAccess(ctx context.Context, team string, hosts []builder.ExternalHost) ([]builder.ExternalEndpoint, error) {
	return nil, fmt.Errorf("external access (public: ports) on the AWS builder is not implemented yet -- it needs a public IP per host plus a security-group ingress rule per public port")
}

var _ builder.Builder = (*Builder)(nil)
