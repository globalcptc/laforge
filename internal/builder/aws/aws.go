// Package aws is a builder.Builder backed by real EC2/VPC + ECS/Fargate calls
// via aws-sdk-go-v2. It is written to the same contract every other builder
// satisfies (internal/builder.Builder) and follows the same conventions --
// deterministic ExternalName-based "ensure" semantics, team-tagged resources,
// power/access as infrastructure operations.
//
// It has NOT yet been exercised against a real AWS account end to end: there is
// no CI account, so the live smoke test and the packet-reachability probe
// (builder_live_test.go / netaccess_probe_test.go) skip unless AWS credentials
// and LAFORGE_AWS_TEST_* are set. Treat the firewall/access paths as implemented
// but pending that real-traffic verification (see docs/builder-authoring.md).
//
// Mapping from LaForge concepts to AWS:
//   - team      -> one VPC (the isolation boundary) with an internet gateway, a
//                  public route table (0.0.0.0/0 -> IGW), and a custom Network
//                  ACL used as the access schedule's kill switch.
//   - network   -> a subnet in that team VPC, carved from the network's CIDR
//                  (added to the VPC as a secondary CIDR association). Networks
//                  in the same VPC route to each other for free, which is what
//                  makes cross-network `visible_from` actually enforceable.
//   - host      -> an EC2 instance in that subnet, with a per-object security
//                  group whose ingress rules ConfigureNetworkAccess converges.
//   - container -> an ECS/Fargate task (see fargate.go), same per-object SG.
//
// Every resource is tagged laforgeManagedTag=true, laforgeTeamTag=<team>, and
// (where it has one) laforgeExternalNameTag=<ExternalName>; instance/container
// security groups also get laforgeExternalRefTag=<externalRef> once the ref is
// known, so ConfigureNetworkAccess can find an object's own SG from the ref the
// runner hands it. Tags are how "ensure" (adopt-or-create) and Inspect find what
// already exists.
package aws

import (
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/ecs"

	"github.com/globalcptc/laforge/internal/builder"
)

const (
	laforgeExternalNameTag = "laforge:external-name"
	laforgeExternalRefTag  = "laforge:external-ref"
	laforgeTeamTag         = "laforge:team"
	laforgeManagedTag      = "laforge:managed" // "true" on everything we create, for Inspect's filter

	// NACL rule numbers. A custom NACL is deny-by-default, so every team subnet
	// gets a baseline allow (ingress + egress) at allowRuleNumber; CloseAccess
	// adds a deny-all ingress at denyIngressRuleNumber, which -- being a LOWER
	// number -- is evaluated first and wins. OpenAccess deletes it. NACLs are
	// stateless and ordered, so the deny severs established flows too, which a
	// (stateful) security-group change cannot.
	denyIngressRuleNumber = 1
	allowRuleNumber       = 100
)

// ConfigInput is the JSON shape stored in builder_config.aws_config and sent by
// the wizard. All fields are optional; an empty field falls back to the
// matching AWS_* environment variable in New, so a runner-env deployment keeps
// working unchanged.
type ConfigInput struct {
	Region           string `json:"region,omitempty"`
	ECSCluster       string `json:"ecs_cluster,omitempty"`
	ExecutionRoleARN string `json:"execution_role_arn,omitempty"`
	LogGroup         string `json:"log_group,omitempty"`
	AgentInitImage   string `json:"agent_init_image,omitempty"`
	// Account is the AWS account id the enrolled credential belongs to (captured
	// at connect via STS). Display-only — shown in the builders list so you can
	// tell which account a builder deploys into; the builder itself doesn't use it.
	Account string `json:"account,omitempty"`
}

// Config is what the resolver hands the builder. Credentials and region come
// from the standard AWS chain (env, shared config, instance profile) via
// LoadDefaultConfig; Region overrides the chain's region when set. Images maps a
// LaForge `os` name to an AMI id; Sizes maps a LaForge `size` name to an EC2
// instance type (and, for Fargate, a "cpu/memory" pair -- see fargate.go).
type Config struct {
	Region string
	Images map[string]string // os name -> AMI id
	Sizes  map[string]string // size name -> instance type (e.g. "t3.medium")

	// Fargate (container) settings; account-specific infrastructure.
	Cluster          string // ECS cluster (AWS_ECS_CLUSTER, default "default")
	ExecutionRoleARN string // task execution role (AWS_ECS_EXECUTION_ROLE_ARN)
	LogGroup         string // CloudWatch log group for task logs (AWS_ECS_LOG_GROUP), optional
	AgentInitImage   string // public image the agent-download init container runs (AWS_ECS_AGENT_INIT_IMAGE, default "alpine:3.20")

	// AccessKeyID/SecretAccessKey are the enrolled wizard credential (from the
	// builder_credential store). When set they are used as a static credentials
	// provider; when empty the builder falls back to the ambient AWS chain (a
	// runner instance-profile role).
	AccessKeyID     string
	SecretAccessKey string
}

// Builder is the EC2/VPC + ECS/Fargate builder.
type Builder struct {
	ec2 *ec2.Client
	ecs *ecs.Client
	cfg Config
}

// New constructs the builder from the ambient AWS credential/region chain,
// filling any empty Config field from its AWS_* environment variable.
func New(ctx context.Context, cfg Config) (*Builder, error) {
	opts := []func(*awsconfig.LoadOptions) error{
		// Adaptive retry runs a client-side rate limiter that backs off on AWS
		// throttling (ThrottlingException), so every call this builder makes --
		// deploy, Inspect, wizard discovery -- rides it out instead of failing a
		// reconcile. (We have been bitten by unthrottled API loops before.)
		awsconfig.WithRetryMode(awssdk.RetryModeAdaptive),
		awsconfig.WithRetryMaxAttempts(8),
	}
	if cfg.Region != "" {
		opts = append(opts, awsconfig.WithRegion(cfg.Region))
	}
	if cfg.AccessKeyID != "" {
		opts = append(opts, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, "")))
	}
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("loading AWS config: %w", err)
	}
	if cfg.Region == "" {
		cfg.Region = awsCfg.Region
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

// ConfigFromInput merges a stored/wizard ConfigInput with the image/size maps
// into a Config. Empty fields are left empty so New can apply env fallbacks.
func ConfigFromInput(in ConfigInput, images, sizes map[string]string) Config {
	return Config{
		Region:           in.Region,
		Images:           images,
		Sizes:            sizes,
		Cluster:          in.ECSCluster,
		ExecutionRoleARN: in.ExecutionRoleARN,
		LogGroup:         in.LogGroup,
		AgentInitImage:   in.AgentInitImage,
	}
}

// ImageNames satisfies internal/orchestrator's imageCatalog interface: the os/
// image names this builder has an AMI configured for (Config.Images), checked
// before a build starts. A compose container needs an entry named
// builder.ComposeHostImage ("compose-host") mapped to a Docker-capable AMI.
func (b *Builder) ImageNames() map[string]bool {
	names := make(map[string]bool, len(b.cfg.Images))
	for name := range b.cfg.Images {
		names[name] = true
	}
	return names
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// --- tag / filter helpers -------------------------------------------------

func tag(key, value string) ec2types.Tag {
	return ec2types.Tag{Key: awssdk.String(key), Value: awssdk.String(value)}
}

func (b *Builder) tagSpec(rt ec2types.ResourceType, externalName, team, displayName string) ec2types.TagSpecification {
	tags := []ec2types.Tag{
		tag(laforgeManagedTag, "true"),
		tag(laforgeTeamTag, team),
	}
	if externalName != "" {
		tags = append(tags, tag(laforgeExternalNameTag, externalName))
	}
	if displayName != "" {
		tags = append(tags, tag("Name", displayName))
	}
	return ec2types.TagSpecification{ResourceType: rt, Tags: tags}
}

func filter(name string, values ...string) ec2types.Filter {
	return ec2types.Filter{Name: awssdk.String(name), Values: values}
}

func nameFilter(externalName string) ec2types.Filter {
	return filter("tag:"+laforgeExternalNameTag, externalName)
}

func teamFilter(team string) ec2types.Filter {
	return filter("tag:"+laforgeTeamTag, team)
}

var managedFilter = filter("tag:"+laforgeManagedTag, "true")

// --- team VPC + egress + NACL (the per-team scaffolding) ------------------

// vpcCIDRFor returns the /16 that contains a network's CIDR. The team VPC is
// created as that /16 so each network's subnet carves straight out of it with NO
// secondary CIDR association -- which is both the standard AWS pattern and the
// fix for "InvalidVpc.Range: CIDR is restricted": AWS rejects adding a /24
// secondary that's really a sub-block of the same /16 as the primary, which is
// exactly what per-/24 association did. A /16 holds up to 256 /24s, so a team's
// networks in one /16 need no association at all (and no CIDR-quota headroom).
// IPv4 only; a network larger than a /16 can't be an AWS VPC and is rejected.
func vpcCIDRFor(networkCIDR string) (string, error) {
	_, ipnet, err := net.ParseCIDR(networkCIDR)
	if err != nil {
		return "", fmt.Errorf("parsing network CIDR %q: %w", networkCIDR, err)
	}
	ip4 := ipnet.IP.To4()
	if ip4 == nil {
		return "", fmt.Errorf("network CIDR %q is not IPv4 (AWS builder is IPv4-only)", networkCIDR)
	}
	if ones, _ := ipnet.Mask.Size(); ones < 16 {
		return "", fmt.Errorf("network CIDR %q is larger than a /16, which exceeds the AWS VPC maximum", networkCIDR)
	}
	return ip4.Mask(net.CIDRMask(16, 32)).String() + "/16", nil
}

// cidrWithin reports whether inner is contained in outer (same or more-specific).
func cidrWithin(inner, outer string) bool {
	innerIP, innerNet, err1 := net.ParseCIDR(inner)
	_, outerNet, err2 := net.ParseCIDR(outer)
	if err1 != nil || err2 != nil {
		return false
	}
	io, _ := innerNet.Mask.Size()
	oo, _ := outerNet.Mask.Size()
	return io >= oo && outerNet.Contains(innerIP)
}

// validateVPCCIDR checks an operator-declared environment supernet is a usable
// AWS VPC CIDR: IPv4, and between /16 (the VPC maximum) and /28 (the minimum).
// Unlike vpcCIDRFor it keeps the declared block as-is rather than rounding to a
// /16, since the operator chose the team's whole address space deliberately.
func validateVPCCIDR(cidr string) error {
	_, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return fmt.Errorf("parsing environment cidr %q: %w", cidr, err)
	}
	if ipnet.IP.To4() == nil {
		return fmt.Errorf("environment cidr %q is not IPv4 (AWS builder is IPv4-only)", cidr)
	}
	ones, _ := ipnet.Mask.Size()
	if ones < 16 || ones > 28 {
		return fmt.Errorf("environment cidr %q is /%d, outside the AWS VPC range of /16 to /28", cidr, ones)
	}
	return nil
}

// ensureTeamVPC finds or creates the team's VPC (one per team) and returns its
// id. When the environment declares an overall address space (vpcCIDR, from
// Environment.CIDR) the VPC is created with exactly that; otherwise the VPC is
// the /16 containing seedCIDR. Either way a team's networks become subnets of it
// with no secondary CIDR association. A describe-before-create guard plus a
// lowest-id dedup makes a concurrent create (networks deploy in parallel)
// converge on one VPC.
func (b *Builder) ensureTeamVPC(ctx context.Context, team, seedCIDR, vpcCIDR string) (string, error) {
	if id, err := b.findTeamVPC(ctx, team); err != nil {
		return "", err
	} else if id != "" {
		return id, nil
	}
	if vpcCIDR != "" {
		if err := validateVPCCIDR(vpcCIDR); err != nil {
			return "", err
		}
	} else {
		if seedCIDR == "" {
			return "", fmt.Errorf("team %s VPC has no CIDR to seed from -- a network needs a CIDR", team)
		}
		inferred, err := vpcCIDRFor(seedCIDR)
		if err != nil {
			return "", err
		}
		vpcCIDR = inferred
	}
	out, err := b.ec2.CreateVpc(ctx, &ec2.CreateVpcInput{
		CidrBlock:         awssdk.String(vpcCIDR),
		TagSpecifications: []ec2types.TagSpecification{b.tagSpec(ec2types.ResourceTypeVpc, "team-"+team+"-vpc", team, "laforge-team-"+team)},
	})
	if err != nil {
		return "", fmt.Errorf("creating VPC for team %s: %w", team, err)
	}
	created := awssdk.ToString(out.Vpc.VpcId)
	// Dedup a parallel-create race: adopt the lowest id, best-effort delete the
	// freshly-created extras (they have no subnets yet).
	id, err := b.findTeamVPC(ctx, team)
	if err != nil || id == "" {
		return created, nil // describe hiccup -- use what we just made
	}
	if id != created {
		_, _ = b.ec2.DeleteVpc(ctx, &ec2.DeleteVpcInput{VpcId: awssdk.String(created)})
	}
	return id, nil
}

func (b *Builder) findTeamVPC(ctx context.Context, team string) (string, error) {
	out, err := b.ec2.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{Filters: []ec2types.Filter{teamFilter(team), managedFilter}})
	if err != nil {
		return "", fmt.Errorf("looking up VPC for team %s: %w", team, err)
	}
	ids := make([]string, 0, len(out.Vpcs))
	for _, v := range out.Vpcs {
		ids = append(ids, awssdk.ToString(v.VpcId))
	}
	if len(ids) == 0 {
		return "", nil
	}
	sort.Strings(ids)
	return ids[0], nil
}

func (b *Builder) ensureInternetGateway(ctx context.Context, vpcID, team string) (string, error) {
	out, err := b.ec2.DescribeInternetGateways(ctx, &ec2.DescribeInternetGatewaysInput{
		Filters: []ec2types.Filter{filter("attachment.vpc-id", vpcID)},
	})
	if err != nil {
		return "", fmt.Errorf("looking up internet gateway for %s: %w", vpcID, err)
	}
	if len(out.InternetGateways) > 0 {
		return awssdk.ToString(out.InternetGateways[0].InternetGatewayId), nil
	}
	created, err := b.ec2.CreateInternetGateway(ctx, &ec2.CreateInternetGatewayInput{
		TagSpecifications: []ec2types.TagSpecification{b.tagSpec(ec2types.ResourceTypeInternetGateway, "team-"+team+"-igw", team, "laforge-team-"+team)},
	})
	if err != nil {
		return "", fmt.Errorf("creating internet gateway for team %s: %w", team, err)
	}
	igw := awssdk.ToString(created.InternetGateway.InternetGatewayId)
	if _, err := b.ec2.AttachInternetGateway(ctx, &ec2.AttachInternetGatewayInput{
		InternetGatewayId: awssdk.String(igw), VpcId: awssdk.String(vpcID),
	}); err != nil && !isDuplicate(err) {
		return "", fmt.Errorf("attaching internet gateway to %s: %w", vpcID, err)
	}
	return igw, nil
}

// ensureTeamRouteTable returns a managed route table with a default route to the
// IGW, creating it (and the route) if absent.
func (b *Builder) ensureTeamRouteTable(ctx context.Context, vpcID, team, igwID string) (string, error) {
	out, err := b.ec2.DescribeRouteTables(ctx, &ec2.DescribeRouteTablesInput{
		Filters: []ec2types.Filter{filter("vpc-id", vpcID), teamFilter(team), managedFilter},
	})
	if err != nil {
		return "", fmt.Errorf("looking up route table for %s: %w", vpcID, err)
	}
	var rtID string
	if len(out.RouteTables) > 0 {
		rtID = awssdk.ToString(out.RouteTables[0].RouteTableId)
	} else {
		created, err := b.ec2.CreateRouteTable(ctx, &ec2.CreateRouteTableInput{
			VpcId:             awssdk.String(vpcID),
			TagSpecifications: []ec2types.TagSpecification{b.tagSpec(ec2types.ResourceTypeRouteTable, "team-"+team+"-rt", team, "laforge-team-"+team)},
		})
		if err != nil {
			return "", fmt.Errorf("creating route table for team %s: %w", team, err)
		}
		rtID = awssdk.ToString(created.RouteTable.RouteTableId)
	}
	if _, err := b.ec2.CreateRoute(ctx, &ec2.CreateRouteInput{
		RouteTableId: awssdk.String(rtID), DestinationCidrBlock: awssdk.String("0.0.0.0/0"), GatewayId: awssdk.String(igwID),
	}); err != nil && !isDuplicate(err) {
		return "", fmt.Errorf("adding default route on %s: %w", rtID, err)
	}
	return rtID, nil
}

// ensureTeamNACL returns the team's custom Network ACL, creating it with a
// baseline allow-all (ingress + egress) if absent. CloseAccess later layers a
// lower-numbered deny on top; the baseline is what keeps an "open" team's
// subnets actually reachable (a custom NACL is deny-by-default).
func (b *Builder) ensureTeamNACL(ctx context.Context, vpcID, team string) (string, error) {
	out, err := b.ec2.DescribeNetworkAcls(ctx, &ec2.DescribeNetworkAclsInput{
		Filters: []ec2types.Filter{filter("vpc-id", vpcID), teamFilter(team), managedFilter},
	})
	if err != nil {
		return "", fmt.Errorf("looking up network ACL for %s: %w", vpcID, err)
	}
	if len(out.NetworkAcls) > 0 {
		return awssdk.ToString(out.NetworkAcls[0].NetworkAclId), nil
	}
	created, err := b.ec2.CreateNetworkAcl(ctx, &ec2.CreateNetworkAclInput{
		VpcId:             awssdk.String(vpcID),
		TagSpecifications: []ec2types.TagSpecification{b.tagSpec(ec2types.ResourceTypeNetworkAcl, "team-"+team+"-nacl", team, "laforge-team-"+team)},
	})
	if err != nil {
		return "", fmt.Errorf("creating network ACL for team %s: %w", team, err)
	}
	naclID := awssdk.ToString(created.NetworkAcl.NetworkAclId)
	for _, egress := range []bool{false, true} {
		if _, err := b.ec2.CreateNetworkAclEntry(ctx, &ec2.CreateNetworkAclEntryInput{
			NetworkAclId: awssdk.String(naclID), RuleNumber: awssdk.Int32(allowRuleNumber),
			Protocol: awssdk.String("-1"), RuleAction: ec2types.RuleActionAllow,
			Egress: awssdk.Bool(egress), CidrBlock: awssdk.String("0.0.0.0/0"),
		}); err != nil && !isDuplicate(err) {
			return "", fmt.Errorf("adding baseline allow to NACL %s: %w", naclID, err)
		}
	}
	return naclID, nil
}

// --- Networks (subnets within the team VPC) -------------------------------

func (b *Builder) DeployNetwork(ctx context.Context, spec builder.NetworkSpec) (string, error) {
	if spec.CIDR == "" {
		return "", fmt.Errorf("network %q has no CIDR -- AWS requires one for the subnet", spec.ExternalName)
	}
	vpcID, err := b.ensureTeamVPC(ctx, spec.Team, spec.CIDR, spec.VPCCIDR)
	if err != nil {
		return "", err
	}
	igwID, err := b.ensureInternetGateway(ctx, vpcID, spec.Team)
	if err != nil {
		return "", err
	}
	rtID, err := b.ensureTeamRouteTable(ctx, vpcID, spec.Team, igwID)
	if err != nil {
		return "", err
	}
	naclID, err := b.ensureTeamNACL(ctx, vpcID, spec.Team)
	if err != nil {
		return "", err
	}
	// Ensure the VPC carries this network's CIDR. When the environment declared
	// an overall supernet (spec.VPCCIDR), the VPC already spans it and the loader
	// has verified every network fits inside it, so the subnet carves straight
	// out -- no association needed (and none wanted: a network outside the
	// declared space is a content bug we'd rather surface at subnet creation than
	// paper over). Only in the inference path (no supernet) might a network land
	// in a different /16 and need its containing block associated.
	if spec.VPCCIDR == "" {
		if err := b.ensureVPCCIDR(ctx, vpcID, spec.CIDR); err != nil {
			return "", err
		}
	}

	// Ensure: adopt a subnet already tagged with this network's ExternalName.
	subnetID, err := b.findSubnet(ctx, spec.ExternalName)
	if err != nil {
		return "", err
	}
	if subnetID == "" {
		created, err := b.ec2.CreateSubnet(ctx, &ec2.CreateSubnetInput{
			VpcId:             awssdk.String(vpcID),
			CidrBlock:         awssdk.String(spec.CIDR),
			TagSpecifications: []ec2types.TagSpecification{b.tagSpec(ec2types.ResourceTypeSubnet, spec.ExternalName, spec.Team, spec.DisplayName)},
		})
		if err != nil {
			return "", fmt.Errorf("creating subnet for %q: %w", spec.ExternalName, err)
		}
		subnetID = awssdk.ToString(created.Subnet.SubnetId)
	}
	// Every host gets a public IP. On AWS an instance with no public IP can't
	// egress even through an IGW (the IGW has no address to NAT), so without a
	// public IP a host can't reach the LaForge control plane to check in. It is
	// not exposure: the per-object security group denies all internet inbound
	// (only declared ports from sibling lab CIDRs; `public:` hosts additionally
	// get a 0.0.0.0/0 rule + EIP). A NAT-gateway alternative isn't provisioned,
	// so this is not optional.
	if _, err := b.ec2.ModifySubnetAttribute(ctx, &ec2.ModifySubnetAttributeInput{
		SubnetId: awssdk.String(subnetID), MapPublicIpOnLaunch: &ec2types.AttributeBooleanValue{Value: awssdk.Bool(true)},
	}); err != nil {
		return "", fmt.Errorf("enabling public IPs on subnet %s: %w", subnetID, err)
	}
	if _, err := b.ec2.AssociateRouteTable(ctx, &ec2.AssociateRouteTableInput{
		RouteTableId: awssdk.String(rtID), SubnetId: awssdk.String(subnetID),
	}); err != nil && !isDuplicate(err) && !alreadyAssociated(err) {
		return "", fmt.Errorf("associating route table with subnet %s: %w", subnetID, err)
	}
	if err := b.associateNACL(ctx, naclID, subnetID); err != nil {
		return "", err
	}
	return subnetID, nil
}

// ensureVPCCIDR makes sure the VPC has a CIDR block that CONTAINS the network's
// CIDR, so the subnet can be carved from it. The common case -- the network is
// inside the VPC's /16 (every /24 of a team that lives in one /16) -- needs
// nothing. Only a network in a different /16 triggers an association, and of its
// containing /16 (not the /24), since AWS forbids a /24 secondary that's a
// sub-block of the primary's /16.
func (b *Builder) ensureVPCCIDR(ctx context.Context, vpcID, cidr string) error {
	out, err := b.ec2.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{VpcIds: []string{vpcID}})
	if err != nil || len(out.Vpcs) == 0 {
		return fmt.Errorf("describing VPC %s: %w", vpcID, err)
	}
	for _, a := range out.Vpcs[0].CidrBlockAssociationSet {
		if cidrWithin(cidr, awssdk.ToString(a.CidrBlock)) {
			return nil // already covered -- the subnet carves from this block
		}
	}
	block, err := vpcCIDRFor(cidr) // the containing /16, not the /24
	if err != nil {
		return err
	}
	if _, err := b.ec2.AssociateVpcCidrBlock(ctx, &ec2.AssociateVpcCidrBlockInput{
		VpcId: awssdk.String(vpcID), CidrBlock: awssdk.String(block),
	}); err != nil && !isDuplicate(err) {
		return fmt.Errorf("associating CIDR %s with VPC %s (VPC CIDR-block quota is 5 /16 blocks by default -- raise it for networks spanning >5 /16 ranges per team): %w", block, vpcID, err)
	}
	return nil
}

// associateNACL points the subnet at the team NACL by replacing its current
// (default) association. Idempotent: a subnet already on this NACL is skipped.
func (b *Builder) associateNACL(ctx context.Context, naclID, subnetID string) error {
	out, err := b.ec2.DescribeNetworkAcls(ctx, &ec2.DescribeNetworkAclsInput{
		Filters: []ec2types.Filter{filter("association.subnet-id", subnetID)},
	})
	if err != nil {
		return fmt.Errorf("looking up NACL association for subnet %s: %w", subnetID, err)
	}
	for _, nacl := range out.NetworkAcls {
		if awssdk.ToString(nacl.NetworkAclId) == naclID {
			return nil // already ours
		}
		for _, a := range nacl.Associations {
			if awssdk.ToString(a.SubnetId) == subnetID {
				if _, err := b.ec2.ReplaceNetworkAclAssociation(ctx, &ec2.ReplaceNetworkAclAssociationInput{
					AssociationId: a.NetworkAclAssociationId, NetworkAclId: awssdk.String(naclID),
				}); err != nil {
					return fmt.Errorf("associating NACL %s with subnet %s: %w", naclID, subnetID, err)
				}
				return nil
			}
		}
	}
	return nil
}

func (b *Builder) DestroyNetwork(ctx context.Context, team, externalRef string) error {
	// externalRef is the subnet id. Find its VPC (for teardown) before deleting.
	vpcID := ""
	if sub, err := b.ec2.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{SubnetIds: []string{externalRef}}); err == nil && len(sub.Subnets) > 0 {
		vpcID = awssdk.ToString(sub.Subnets[0].VpcId)
	}
	if _, err := b.ec2.DeleteSubnet(ctx, &ec2.DeleteSubnetInput{SubnetId: awssdk.String(externalRef)}); err != nil && !isNotFound(err) {
		return fmt.Errorf("deleting subnet %s: %w", externalRef, err)
	}
	if vpcID == "" {
		if id, err := b.findTeamVPC(ctx, team); err == nil {
			vpcID = id
		}
	}
	if vpcID == "" {
		return nil
	}
	// Tear the team VPC down only once its last managed subnet is gone.
	remaining, err := b.ec2.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{
		Filters: []ec2types.Filter{filter("vpc-id", vpcID), managedFilter},
	})
	if err == nil && len(remaining.Subnets) == 0 {
		// Propagate the teardown result: a VPC that won't delete (an ENI still
		// draining, a leftover SG) must fail the task so it retries, not return
		// success and silently leak the whole VPC + IGW + route tables + NACLs.
		return b.teardownTeamVPC(ctx, vpcID)
	}
	return nil
}

// teardownTeamVPC removes everything the team VPC still holds -- its managed
// security groups, IGW, route tables and custom NACLs -- and then the VPC
// itself. The dependency deletes are best-effort (a not-found is fine), but the
// final DeleteVpc error is RETURNED: if anything still references the VPC (a
// terminating instance's ENI, a stopping Fargate task's ENI, a straggler SG) the
// delete fails with DependencyViolation, and the caller retries until it's truly
// gone. Deleting the managed SGs here is what unblocks that DeleteVpc -- a VPC
// won't delete while a non-default security group remains in it.
func (b *Builder) teardownTeamVPC(ctx context.Context, vpcID string) error {
	if sgs, err := b.ec2.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{
		Filters: []ec2types.Filter{filter("vpc-id", vpcID), managedFilter},
	}); err == nil {
		for _, sg := range sgs.SecurityGroups {
			_, _ = b.ec2.DeleteSecurityGroup(ctx, &ec2.DeleteSecurityGroupInput{GroupId: sg.GroupId})
		}
	}
	if igws, err := b.ec2.DescribeInternetGateways(ctx, &ec2.DescribeInternetGatewaysInput{
		Filters: []ec2types.Filter{filter("attachment.vpc-id", vpcID)},
	}); err == nil {
		for _, igw := range igws.InternetGateways {
			_, _ = b.ec2.DetachInternetGateway(ctx, &ec2.DetachInternetGatewayInput{InternetGatewayId: igw.InternetGatewayId, VpcId: awssdk.String(vpcID)})
			_, _ = b.ec2.DeleteInternetGateway(ctx, &ec2.DeleteInternetGatewayInput{InternetGatewayId: igw.InternetGatewayId})
		}
	}
	if rts, err := b.ec2.DescribeRouteTables(ctx, &ec2.DescribeRouteTablesInput{
		Filters: []ec2types.Filter{filter("vpc-id", vpcID), managedFilter},
	}); err == nil {
		for _, rt := range rts.RouteTables {
			_, _ = b.ec2.DeleteRouteTable(ctx, &ec2.DeleteRouteTableInput{RouteTableId: rt.RouteTableId})
		}
	}
	if nacls, err := b.ec2.DescribeNetworkAcls(ctx, &ec2.DescribeNetworkAclsInput{
		Filters: []ec2types.Filter{filter("vpc-id", vpcID), managedFilter},
	}); err == nil {
		for _, n := range nacls.NetworkAcls {
			_, _ = b.ec2.DeleteNetworkAcl(ctx, &ec2.DeleteNetworkAclInput{NetworkAclId: n.NetworkAclId})
		}
	}
	if _, err := b.ec2.DeleteVpc(ctx, &ec2.DeleteVpcInput{VpcId: awssdk.String(vpcID)}); err != nil && !isNotFound(err) {
		return fmt.Errorf("deleting team VPC %s (resources may still be draining -- will retry): %w", vpcID, err)
	}
	return nil
}

func (b *Builder) findSubnet(ctx context.Context, externalName string) (string, error) {
	out, err := b.ec2.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{Filters: []ec2types.Filter{nameFilter(externalName)}})
	if err != nil {
		return "", fmt.Errorf("looking up subnet %q: %w", externalName, err)
	}
	if len(out.Subnets) > 0 {
		return awssdk.ToString(out.Subnets[0].SubnetId), nil
	}
	return "", nil
}

// subnetForNetwork resolves a network's ExternalName to its subnet id and VPC id
// (a security group is VPC-scoped, so callers need both).
func (b *Builder) subnetForNetwork(ctx context.Context, networkExternalName string) (subnetID, vpcID string, err error) {
	out, err := b.ec2.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{Filters: []ec2types.Filter{nameFilter(networkExternalName)}})
	if err != nil {
		return "", "", fmt.Errorf("looking up subnet for network %q: %w", networkExternalName, err)
	}
	if len(out.Subnets) == 0 {
		return "", "", fmt.Errorf("no subnet found for network %q -- its DeployNetwork may not have run yet", networkExternalName)
	}
	return awssdk.ToString(out.Subnets[0].SubnetId), awssdk.ToString(out.Subnets[0].VpcId), nil
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
	subnetID, vpcID, err := b.subnetForNetwork(ctx, spec.Network)
	if err != nil {
		return "", err
	}
	sgID, err := b.ensureObjectSecurityGroup(ctx, spec.ExternalName, spec.Team, vpcID)
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
		// EC2 user-data must be base64; cloud-init (Linux) and cloudbase-init
		// (Windows) consume it the same way the Incus cloud-init.user-data key does.
		in.UserData = awssdk.String(base64.StdEncoding.EncodeToString([]byte(spec.CloudInitUserData)))
	}
	out, err := b.ec2.RunInstances(ctx, in)
	if err != nil {
		return "", fmt.Errorf("launching instance for %q: %w", spec.ExternalName, err)
	}
	if len(out.Instances) == 0 {
		return "", fmt.Errorf("RunInstances for %q returned no instance", spec.ExternalName)
	}
	ref := awssdk.ToString(out.Instances[0].InstanceId)
	// Tag the object SG with the ref so ConfigureNetworkAccess/ExternalAccess can
	// find it from the externalRef the runner hands back (they don't get the name).
	b.tagSGWithRef(ctx, sgID, ref)
	return ref, nil
}

func (b *Builder) DestroyHost(ctx context.Context, team, externalRef string) error {
	// Release any Elastic IP we attached for external access, so a teardown
	// doesn't leak (and keep paying for) an allocation.
	b.releaseEIPForRef(ctx, externalRef)
	if _, err := b.ec2.TerminateInstances(ctx, &ec2.TerminateInstancesInput{InstanceIds: []string{externalRef}}); err != nil && !isNotFound(err) {
		return fmt.Errorf("terminating instance %s: %w", externalRef, err)
	}
	// Delete the host's per-object security group. If we don't, it leaks AND
	// blocks the team VPC teardown later (a VPC won't delete while a non-default
	// SG remains in it). The SG can't go while the instance's ENI still
	// references it, so a DependencyViolation here just means the instance is
	// still terminating -- deleteObjectSecurityGroup returns that as a retryable
	// error so the task runs again and succeeds once the ENI is gone.
	return b.deleteObjectSecurityGroup(ctx, externalRef)
}

// deleteObjectSecurityGroup removes the per-object security group tagged with
// this externalRef (created at deploy by ensureObjectSecurityGroup). Returns a
// retryable error on DependencyViolation (the instance/task still terminating);
// a missing SG, or no SG for this ref, is a no-op success.
func (b *Builder) deleteObjectSecurityGroup(ctx context.Context, ref string) error {
	sgID, err := b.sgForRef(ctx, ref)
	if err != nil {
		return nil // a describe hiccup shouldn't fail teardown; the VPC sweep catches stragglers
	}
	if sgID == "" {
		return nil
	}
	if _, err := b.ec2.DeleteSecurityGroup(ctx, &ec2.DeleteSecurityGroupInput{GroupId: awssdk.String(sgID)}); err != nil && !isNotFound(err) {
		if isDependencyViolation(err) {
			return fmt.Errorf("security group %s still attached (instance/task still terminating); will retry", sgID)
		}
		return fmt.Errorf("deleting security group %s: %w", sgID, err)
	}
	return nil
}

func (b *Builder) findInstance(ctx context.Context, externalName string) (string, error) {
	out, err := b.ec2.DescribeInstances(ctx, &ec2.DescribeInstancesInput{Filters: []ec2types.Filter{
		nameFilter(externalName),
		filter("instance-state-name", "pending", "running", "stopping", "stopped"),
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
	var out []builder.Resource

	insts, err := b.ec2.DescribeInstances(ctx, &ec2.DescribeInstancesInput{Filters: []ec2types.Filter{managedFilter}})
	if err != nil {
		return nil, fmt.Errorf("describing instances: %w", err)
	}
	for _, r := range insts.Reservations {
		for _, inst := range r.Instances {
			if inst.State != nil && inst.State.Name == ec2types.InstanceStateNameTerminated {
				continue
			}
			out = append(out, builder.Resource{
				ExternalRef: awssdk.ToString(inst.InstanceId),
				Kind:        "host",
				State:       mapInstanceState(inst.State),
			})
		}
	}

	// A network is a subnet now, not a VPC.
	subnets, err := b.ec2.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{Filters: []ec2types.Filter{managedFilter}})
	if err != nil {
		return nil, fmt.Errorf("describing subnets: %w", err)
	}
	for _, s := range subnets.Subnets {
		out = append(out, builder.Resource{ExternalRef: awssdk.ToString(s.SubnetId), Kind: "network"})
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

// --- Per-object security group (the port firewall's handle) ---------------

// ensureObjectSecurityGroup returns the id of this object's own security group
// in the VPC, creating it (closed: no ingress, default egress allow-all) if
// absent. ConfigureNetworkAccess converges its ingress rules later; a host with
// no declared ports stays reachable on nothing.
func (b *Builder) ensureObjectSecurityGroup(ctx context.Context, externalName, team, vpcID string) (string, error) {
	existing, err := b.ec2.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{Filters: []ec2types.Filter{
		nameFilter(externalName), filter("vpc-id", vpcID),
	}})
	if err == nil && len(existing.SecurityGroups) > 0 {
		return awssdk.ToString(existing.SecurityGroups[0].GroupId), nil
	}
	created, err := b.ec2.CreateSecurityGroup(ctx, &ec2.CreateSecurityGroupInput{
		GroupName:         awssdk.String("laforge-" + sanitizeECSName(externalName)),
		Description:       awssdk.String("LaForge per-object firewall for " + externalName),
		VpcId:             awssdk.String(vpcID),
		TagSpecifications: []ec2types.TagSpecification{b.tagSpec(ec2types.ResourceTypeSecurityGroup, externalName, team, "")},
	})
	if err != nil {
		return "", fmt.Errorf("creating security group for %q: %w", externalName, err)
	}
	return awssdk.ToString(created.GroupId), nil
}

func (b *Builder) tagSGWithRef(ctx context.Context, sgID, ref string) {
	_, _ = b.ec2.CreateTags(ctx, &ec2.CreateTagsInput{Resources: []string{sgID}, Tags: []ec2types.Tag{tag(laforgeExternalRefTag, ref)}})
}

// sgForRef finds an object's security group from the externalRef the runner
// hands ConfigureNetworkAccess/ExternalAccess (DeployHost/Container tagged it).
func (b *Builder) sgForRef(ctx context.Context, ref string) (string, error) {
	out, err := b.ec2.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{Filters: []ec2types.Filter{
		filter("tag:"+laforgeExternalRefTag, ref), managedFilter,
	}})
	if err != nil {
		return "", fmt.Errorf("looking up security group for ref %s: %w", ref, err)
	}
	if len(out.SecurityGroups) == 0 {
		return "", nil
	}
	return awssdk.ToString(out.SecurityGroups[0].GroupId), nil
}

func (b *Builder) securityGroupIngress(ctx context.Context, sgID string) ([]ec2types.IpPermission, error) {
	out, err := b.ec2.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{GroupIds: []string{sgID}})
	if err != nil || len(out.SecurityGroups) == 0 {
		return nil, err
	}
	return out.SecurityGroups[0].IpPermissions, nil
}

// --- Access schedule (per-team NACL deny toggle) --------------------------

func (b *Builder) teamNACLs(ctx context.Context, team string) ([]string, error) {
	out, err := b.ec2.DescribeNetworkAcls(ctx, &ec2.DescribeNetworkAclsInput{Filters: []ec2types.Filter{teamFilter(team), managedFilter}})
	if err != nil {
		return nil, fmt.Errorf("listing team %s network ACLs: %w", team, err)
	}
	ids := make([]string, 0, len(out.NetworkAcls))
	for _, n := range out.NetworkAcls {
		ids = append(ids, awssdk.ToString(n.NetworkAclId))
	}
	return ids, nil
}

// CloseAccess blocks all ingress to the team's subnets and, because a NACL is
// stateless, drops already-established flows too -- not merely new ones, as a
// (stateful) security-group revoke would. OpenAccess removes the deny.
func (b *Builder) CloseAccess(ctx context.Context, team string) error {
	nacls, err := b.teamNACLs(ctx, team)
	if err != nil {
		return err
	}
	for _, n := range nacls {
		_, err := b.ec2.CreateNetworkAclEntry(ctx, &ec2.CreateNetworkAclEntryInput{
			NetworkAclId: awssdk.String(n), RuleNumber: awssdk.Int32(denyIngressRuleNumber),
			Protocol: awssdk.String("-1"), RuleAction: ec2types.RuleActionDeny,
			Egress: awssdk.Bool(false), CidrBlock: awssdk.String("0.0.0.0/0"),
		})
		if err != nil && !isDuplicate(err) {
			return fmt.Errorf("closing access on NACL %s: %w", n, err)
		}
	}
	return nil
}

func (b *Builder) OpenAccess(ctx context.Context, team string) error {
	nacls, err := b.teamNACLs(ctx, team)
	if err != nil {
		return err
	}
	for _, n := range nacls {
		_, err := b.ec2.DeleteNetworkAclEntry(ctx, &ec2.DeleteNetworkAclEntryInput{
			NetworkAclId: awssdk.String(n), RuleNumber: awssdk.Int32(denyIngressRuleNumber), Egress: awssdk.Bool(false),
		})
		if err != nil && !isNotFound(err) {
			return fmt.Errorf("opening access on NACL %s: %w", n, err)
		}
	}
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

// --- Network access (visible_from + ports) --------------------------------

// ConfigureNetworkAccess enforces a team's `visible_from` + `ports:` policy via
// each object's own security group. All of a team's networks are subnets in the
// one team VPC, so they route to each other for free; the per-object SG (stateful,
// per-ENI) then filters both cross-subnet and same-subnet traffic to exactly the
// host's declared ports from its own network + its `visible_from` networks. A
// host with no declared ports gets no ingress -- reachable on nothing.
func (b *Builder) ConfigureNetworkAccess(ctx context.Context, team string, networks []builder.NetworkAccess) error {
	cidrByName := make(map[string]string, len(networks))
	for _, na := range networks {
		cidrByName[na.ExternalName] = na.CIDR
	}
	for _, na := range networks {
		sources := []string{na.CIDR}
		for _, from := range na.VisibleFrom {
			if c := cidrByName[from]; c != "" {
				sources = append(sources, c)
			}
		}
		for _, h := range na.Hosts {
			if h.ExternalRef == "" {
				continue // not deployed yet; a later pass configures it
			}
			if err := b.convergeHostFirewall(ctx, h, sources); err != nil {
				return fmt.Errorf("configuring firewall for %s (team %s): %w", h.ExternalRef, team, err)
			}
		}
	}
	return nil
}

// convergeHostFirewall replaces the object SG's ingress with exactly the current
// declared-port-from-allowed-source set. Idempotent: rules are rebuilt each pass
// so an edited policy converges. Works for EC2 and Fargate alike -- it edits the
// SG's rules, never which SG the object has (fixed at launch for Fargate).
func (b *Builder) convergeHostFirewall(ctx context.Context, h builder.HostAccess, sources []string) error {
	sgID, err := b.sgForRef(ctx, h.ExternalRef)
	if err != nil {
		return err
	}
	if sgID == "" {
		return nil // object SG not tagged yet (deploy mid-flight); a later pass retries
	}
	if cur, err := b.securityGroupIngress(ctx, sgID); err == nil && len(cur) > 0 {
		if _, err := b.ec2.RevokeSecurityGroupIngress(ctx, &ec2.RevokeSecurityGroupIngressInput{
			GroupId: awssdk.String(sgID), IpPermissions: cur,
		}); err != nil && !isNotFound(err) {
			return fmt.Errorf("clearing ingress on %s: %w", sgID, err)
		}
	}
	perms := portPermissions(sources, h.TCPPorts, h.UDPPorts)
	if len(perms) > 0 {
		if _, err := b.ec2.AuthorizeSecurityGroupIngress(ctx, &ec2.AuthorizeSecurityGroupIngressInput{
			GroupId: awssdk.String(sgID), IpPermissions: perms,
		}); err != nil && !isDuplicate(err) {
			return fmt.Errorf("authorizing firewall rules on %s: %w", sgID, err)
		}
	}
	return nil
}

// portPermissions turns declared TCP/UDP ports and allowed source CIDRs into EC2
// ingress permissions: one permission per port(-range) per protocol, from all the
// source CIDRs. An empty port set yields nothing (default-deny). Pure, so it is
// unit-tested offline (see aws_test.go).
func portPermissions(sources []string, tcp, udp []string) []ec2types.IpPermission {
	if len(sources) == 0 {
		return nil
	}
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

var _ builder.Builder = (*Builder)(nil)
