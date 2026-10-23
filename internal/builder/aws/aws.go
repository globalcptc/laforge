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
//     public route table (0.0.0.0/0 -> IGW), and a custom Network
//     ACL used as the access schedule's kill switch.
//   - network   -> a subnet in that team VPC, carved from the network's CIDR
//     (added to the VPC as a secondary CIDR association). Networks
//     in the same VPC route to each other for free, which is what
//     makes cross-network `visible_from` actually enforceable.
//   - host      -> an EC2 instance in that subnet, with a per-object security
//     group whose ingress rules ConfigureNetworkAccess converges.
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
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/servicequotas"

	"github.com/globalcptc/laforge/internal/agentdelivery"
	"github.com/globalcptc/laforge/internal/builder"
)

const (
	laforgeExternalNameTag = "laforge:external-name"
	laforgeExternalRefTag  = "laforge:external-ref"
	laforgeTeamTag         = "laforge:team"
	laforgeBuildTag        = "laforge:build"   // the build uuid, so a team's VPC is scoped per-build, not shared by team number across builds
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
	// KeyName is an existing EC2 key pair (in this region) associated with every
	// launched instance, so the operator can retrieve a Windows box's admin
	// password with `aws ec2 get-password-data --priv-launch-key`. Deliberately
	// NOT set via user-data: a user-data password is readable from the instance
	// metadata by anyone on the box (a competitor), whereas the key pair keeps it
	// out of the guest entirely. Empty means no key pair (falls back to AWS_KEY_NAME).
	KeyName string `json:"key_name,omitempty"`
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
	KeyName          string // EC2 key pair associated with launched instances for Windows password retrieval (AWS_KEY_NAME), optional

	// AccessKeyID/SecretAccessKey are the enrolled wizard credential (from the
	// builder_credential store). When set they are used as a static credentials
	// provider; when empty the builder falls back to the ambient AWS chain (a
	// runner instance-profile role).
	AccessKeyID     string
	SecretAccessKey string
}

// Builder is the EC2/VPC + ECS/Fargate builder.
type Builder struct {
	ec2    *ec2.Client
	ecs    *ecs.Client
	logs   *cloudwatchlogs.Client
	cw     *cloudwatch.Client
	quotas *servicequotas.Client
	cfg    Config

	// Cached AZ for every subnet this builder creates: an AZ that offers ALL the
	// builder's configured instance types, because a subnet created without an
	// explicit AZ lands in whichever AZ AWS picks -- and a legacy AZ like
	// us-east-1e offers no t3/current types, so RunInstances into it fails
	// "instance type not supported in your requested Availability Zone".
	// Best-effort: "" means "couldn't determine one, let AWS choose", never a hard
	// failure. Resolved under azMu and cached only on SUCCESS (azResolved) -- a
	// transient/permission failure must not stick as "" and silently disable
	// AZ-pinning for every later subnet.
	azMu       sync.Mutex
	azCached   string
	azResolved bool
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
		// Always capture Fargate container output somewhere -- a container that
		// exits should never be a black box ("essential container exited, exit
		// code 1" with no logs). The builder ensure-creates this group; an
		// environment's own container_logs still overrides per container.
		cfg.LogGroup = envOr("AWS_ECS_LOG_GROUP", "/laforge/fargate")
	}
	if cfg.AgentInitImage == "" {
		cfg.AgentInitImage = envOr("AWS_ECS_AGENT_INIT_IMAGE", "alpine:3.20")
	}
	if cfg.KeyName == "" {
		cfg.KeyName = os.Getenv("AWS_KEY_NAME")
	}
	return &Builder{
		ec2:    ec2.NewFromConfig(awsCfg),
		ecs:    ecs.NewFromConfig(awsCfg),
		logs:   cloudwatchlogs.NewFromConfig(awsCfg),
		cw:     cloudwatch.NewFromConfig(awsCfg),
		quotas: servicequotas.NewFromConfig(awsCfg),
		cfg:    cfg,
	}, nil
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
		KeyName:          in.KeyName,
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

func buildFilter(build string) ec2types.Filter {
	return filter("tag:"+laforgeBuildTag, build)
}

var managedFilter = filter("tag:"+laforgeManagedTag, "true")

// buildFromExternalName extracts the build uuid from a deterministic
// ExternalName ("build-<uuid>-team-<n>-<kind>-<name>" -- see
// internal/runner.deterministicExternalName). Returns "" when name isn't in that
// form (so the caller falls back to a team-only scope, e.g. a pre-build-scoping
// legacy resource during teardown).
func buildFromExternalName(name string) string {
	const prefix = "build-"
	if !strings.HasPrefix(name, prefix) {
		return ""
	}
	rest := name[len(prefix):]
	i := strings.Index(rest, "-team-")
	if i <= 0 {
		return ""
	}
	return rest[:i]
}

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
func (b *Builder) ensureTeamVPC(ctx context.Context, build, team, seedCIDR, vpcCIDR string) (string, error) {
	if id, err := b.findTeamVPC(ctx, build, team); err != nil {
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
	// Tag the VPC with the build uuid so it's scoped per-build: two builds that
	// happen to share a team number must NOT share (or tear down) one VPC.
	vpcTags := b.tagSpec(ec2types.ResourceTypeVpc, "team-"+team+"-vpc", team, "laforge-team-"+team)
	if build != "" {
		vpcTags.Tags = append(vpcTags.Tags, tag(laforgeBuildTag, build))
	}
	out, err := b.ec2.CreateVpc(ctx, &ec2.CreateVpcInput{
		CidrBlock:         awssdk.String(vpcCIDR),
		TagSpecifications: []ec2types.TagSpecification{vpcTags},
	})
	if err != nil {
		return "", fmt.Errorf("creating VPC for team %s: %w", team, err)
	}
	created := awssdk.ToString(out.Vpc.VpcId)
	// Dedup a parallel-create race: adopt the lowest id, best-effort delete the
	// freshly-created extras (they have no subnets yet).
	id, err := b.findTeamVPC(ctx, build, team)
	if err != nil || id == "" {
		return created, nil // describe hiccup -- use what we just made
	}
	if id != created {
		_, _ = b.ec2.DeleteVpc(ctx, &ec2.DeleteVpcInput{VpcId: awssdk.String(created)})
	}
	return id, nil
}

// findTeamVPC returns the managed VPC for a build's team. When build is set it
// scopes to that build's own VPC (the normal case); when build is "" it falls
// back to team-only, which only matters during teardown of a legacy VPC created
// before VPCs were build-scoped.
func (b *Builder) findTeamVPC(ctx context.Context, build, team string) (string, error) {
	filters := []ec2types.Filter{teamFilter(team), managedFilter}
	if build != "" {
		filters = append(filters, buildFilter(build))
	}
	out, err := b.ec2.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{Filters: filters})
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

// vpcSetupLocks serializes each team's VPC-level resource creation (VPC, IGW,
// route table, NACL, CIDR) across the networks DeployNetwork is called for
// concurrently. Process-global so it holds regardless of how many builder
// instances the runner resolves; keyed by build+team (the VPC's identity before
// it exists). An entry is tiny and the set is bounded by builds*teams, so these
// are never reclaimed -- not worth the bookkeeping.
var vpcSetupLocks sync.Map // key string -> *sync.Mutex

func vpcSetupLock(key string) *sync.Mutex {
	m, _ := vpcSetupLocks.LoadOrStore(key, &sync.Mutex{})
	return m.(*sync.Mutex)
}

// ensureTeamVPCResources builds (or adopts) the team's shared VPC-level resources
// under the per-team lock, so concurrent DeployNetwork calls don't race to create
// them. Returns the ids the per-network subnet setup needs.
func (b *Builder) ensureTeamVPCResources(ctx context.Context, spec builder.NetworkSpec) (vpcID, rtID, naclID string, err error) {
	lk := vpcSetupLock(buildFromExternalName(spec.ExternalName) + "/" + spec.Team)
	lk.Lock()
	defer lk.Unlock()

	vpcID, err = b.ensureTeamVPC(ctx, buildFromExternalName(spec.ExternalName), spec.Team, spec.CIDR, spec.VPCCIDR)
	if err != nil {
		return "", "", "", err
	}
	igwID, err := b.ensureInternetGateway(ctx, vpcID, spec.Team)
	if err != nil {
		return "", "", "", err
	}
	rtID, err = b.ensureTeamRouteTable(ctx, vpcID, spec.Team, igwID)
	if err != nil {
		return "", "", "", err
	}
	naclID, err = b.ensureTeamNACL(ctx, vpcID, spec.Team)
	if err != nil {
		return "", "", "", err
	}
	// Ensure the VPC carries this network's CIDR. When the environment declared
	// an overall supernet (spec.VPCCIDR), the VPC already spans it and the loader
	// has verified every network fits inside it, so the subnet carves straight
	// out -- no association needed (and none wanted: a network outside the
	// declared space is a content bug we'd rather surface at subnet creation than
	// paper over). Only in the inference path (no supernet) might a network land
	// in a different /16 and need its containing block associated.
	if spec.VPCCIDR == "" {
		if err = b.ensureVPCCIDR(ctx, vpcID, spec.CIDR); err != nil {
			return "", "", "", err
		}
	}
	return vpcID, rtID, naclID, nil
}

func (b *Builder) DeployNetwork(ctx context.Context, spec builder.NetworkSpec) (string, error) {
	if spec.CIDR == "" {
		return "", fmt.Errorf("network %q has no CIDR -- AWS requires one for the subnet", spec.ExternalName)
	}
	// The VPC and its IGW/route-table/NACL are PER-TEAM, shared by all of a team's
	// networks -- but the runner calls DeployNetwork for them concurrently. Each
	// "ensure" is describe-then-create, which races: several see no IGW and all
	// create+attach one, the losers get "already has an internet gateway attached"
	// (a 15-30s retry) and leak the orphan IGW they created (burning the IGW
	// quota). Serialize the per-team VPC-level setup so exactly one network builds
	// the shared resources and the rest adopt them. Per-network work (the subnet)
	// stays concurrent, below.
	vpcID, rtID, naclID, err := b.ensureTeamVPCResources(ctx, spec)
	if err != nil {
		return "", err
	}

	// Ensure: adopt a subnet already tagged with this network's ExternalName.
	subnetID, err := b.findSubnet(ctx, spec.ExternalName)
	if err != nil {
		return "", err
	}
	if subnetID == "" {
		// Pin the subnet to an AZ that offers the builder's instance types -- a
		// subnet created without one lands wherever AWS chooses, and a legacy AZ
		// (e.g. us-east-1e) offers no t3/current types, so every host in it fails
		// to launch. Best-effort: "" (can't determine one) lets AWS pick, as before.
		az := b.subnetAZ(ctx)
		in := &ec2.CreateSubnetInput{
			VpcId:             awssdk.String(vpcID),
			CidrBlock:         awssdk.String(spec.CIDR),
			TagSpecifications: []ec2types.TagSpecification{b.tagSpec(ec2types.ResourceTypeSubnet, spec.ExternalName, spec.Team, spec.DisplayName)},
		}
		if az != "" {
			in.AvailabilityZone = awssdk.String(az)
		}
		created, err := b.ec2.CreateSubnet(ctx, in)
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
	// Resolve externalRef to a real subnet id. Normally it's the stored
	// external_ref ("subnet-..."); when the object never recorded one (its deploy
	// failed before CreateSubnet) the runner falls back to the deterministic
	// ExternalName, so look the subnet up by its laforge:external-name tag. An
	// empty result means the subnet was never created -- nothing to delete, but
	// the team VPC may still exist from a partial deploy, so fall through to its
	// teardown below.
	subnetID := externalRef
	if !strings.HasPrefix(subnetID, "subnet-") {
		id, err := b.findSubnet(ctx, externalRef)
		if err != nil {
			return err
		}
		subnetID = id
	}
	vpcID := ""
	if subnetID != "" {
		if sub, err := b.ec2.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{SubnetIds: []string{subnetID}}); err == nil && len(sub.Subnets) > 0 {
			vpcID = awssdk.ToString(sub.Subnets[0].VpcId)
		}
		if _, err := b.ec2.DeleteSubnet(ctx, &ec2.DeleteSubnetInput{SubnetId: awssdk.String(subnetID)}); err != nil && !isNotFound(err) && !isMalformed(err) {
			return fmt.Errorf("deleting subnet %s: %w", subnetID, err)
		}
	}
	// Even with no subnet of our own, the team VPC may still be out there (a
	// partial deploy that created the VPC then failed), so find it by tag and
	// tear it down once no managed subnets remain -- that's what reclaims an
	// orphaned VPC instead of leaking it. Scope to this build when the ref still
	// carries its ExternalName; fall back to team-only to also reclaim a legacy
	// VPC created before VPCs were build-scoped.
	if vpcID == "" {
		build := buildFromExternalName(externalRef)
		if id, err := b.findTeamVPC(ctx, build, team); err == nil && id != "" {
			vpcID = id
		} else if build != "" {
			if id, err := b.findTeamVPC(ctx, "", team); err == nil {
				vpcID = id
			}
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

// subnetForNetwork resolves a network's ExternalName to its subnet id, VPC id,
// CIDR, and Availability Zone (a security group is VPC-scoped, so callers need
// the VPC; the AZ lets the launch path verify it can run the host's instance
// type before RunInstances fails opaquely).
func (b *Builder) subnetForNetwork(ctx context.Context, networkExternalName string) (subnetID, vpcID, cidr, az string, err error) {
	out, err := b.ec2.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{Filters: []ec2types.Filter{nameFilter(networkExternalName)}})
	if err != nil {
		return "", "", "", "", fmt.Errorf("looking up subnet for network %q: %w", networkExternalName, err)
	}
	if len(out.Subnets) == 0 {
		return "", "", "", "", fmt.Errorf("no subnet found for network %q -- its DeployNetwork may not have run yet", networkExternalName)
	}
	s := out.Subnets[0]
	return awssdk.ToString(s.SubnetId), awssdk.ToString(s.VpcId), awssdk.ToString(s.CidrBlock), awssdk.ToString(s.AvailabilityZone), nil
}

// azOffersType reports whether instanceType is offered in az, and returns the
// sorted list of AZs in the region that DO offer it (for a clear error).
// BEST-EFFORT: on any API error it returns ok=true so a missing
// ec2:DescribeInstanceTypeOfferings permission (or a throttle) never becomes a
// false launch block -- RunInstances stays the backstop.
func (b *Builder) azOffersType(ctx context.Context, az, instanceType string) (ok bool, supported []string, err error) {
	out, err := b.ec2.DescribeInstanceTypeOfferings(ctx, &ec2.DescribeInstanceTypeOfferingsInput{
		LocationType: ec2types.LocationTypeAvailabilityZone,
		Filters:      []ec2types.Filter{filter("instance-type", instanceType)},
	})
	if err != nil {
		return true, nil, err
	}
	for _, o := range out.InstanceTypeOfferings {
		loc := awssdk.ToString(o.Location)
		supported = append(supported, loc)
		if loc == az {
			ok = true
		}
	}
	sort.Strings(supported)
	return ok, supported, nil
}

// subnetAZ returns an Availability Zone that offers EVERY instance type in the
// builder's size map, resolved once and cached. Pinning subnets to it stops AWS
// from dropping one into a legacy AZ (e.g. us-east-1e) that can't run current
// instance types. BEST-EFFORT: "" means "let AWS choose the subnet's AZ" -- the
// behavior before this optimization -- which is what we fall back to whenever the
// lookup can't run (e.g. the credential lacks ec2:DescribeInstanceTypeOfferings,
// or no single AZ offers every type). This is an optimization, never a gate: a
// missing permission must not fail the whole build's network deploys.
func (b *Builder) subnetAZ(ctx context.Context) string {
	b.azMu.Lock()
	defer b.azMu.Unlock()
	if b.azResolved {
		return b.azCached
	}
	az, err := b.resolveSubnetAZ(ctx)
	if err != nil {
		// Type-aware resolution failed -- in practice almost always because the
		// runner credential is DENIED ec2:DescribeInstanceTypeOfferings (a 403,
		// confirmed in the field). We must NOT fall through to letting AWS pick
		// the subnet's AZ: on this account AWS defaults to the legacy us-east-1e,
		// which offers no t3/current types, so every host then fails to launch.
		// Instead fall back to the first standard Availability Zone the account
		// has (DescribeAvailabilityZones -- a far more commonly granted call). Not
		// type-aware, but the alphabetically-first AZ (us-east-1a) is a current
		// one, so it reliably beats AWS's legacy default. Grant
		// ec2:DescribeInstanceTypeOfferings to restore the precise, type-checked
		// choice (and the pre-launch guard and readiness warning).
		log.Printf("aws: type-aware subnet AZ lookup failed (%v) -- falling back to the first standard AZ so subnets avoid AWS's legacy-AZ default; grant ec2:DescribeInstanceTypeOfferings to restore type-aware selection", err)
		if fb, fbErr := b.firstStandardAZ(ctx); fbErr != nil {
			log.Printf("aws: could not list Availability Zones for the fallback either (%v) -- letting AWS choose the subnet AZ, which may be unusable", fbErr)
			return "" // both lookups denied/failed; nothing safe to pin to
		} else if fb != "" {
			b.azCached = fb
			b.azResolved = true
			return fb
		}
		return ""
	}
	b.azCached = az
	b.azResolved = true
	return az
}

// firstStandardAZ returns the alphabetically-first standard (opt-in-not-required)
// Availability Zone the account has available in the region. It's the
// permission-light fallback for subnetAZ when the type-aware lookup is denied:
// DescribeAvailabilityZones is far more commonly granted than
// DescribeInstanceTypeOfferings, and the first standard AZ (e.g. us-east-1a) is a
// current one -- unlike AWS's own unspecified-AZ default, which on some accounts
// is the legacy us-east-1e that can't run t3. Opt-in zones (Local Zones,
// Wavelength) are excluded.
func (b *Builder) firstStandardAZ(ctx context.Context) (string, error) {
	out, err := b.ec2.DescribeAvailabilityZones(ctx, &ec2.DescribeAvailabilityZonesInput{
		Filters: []ec2types.Filter{filter("state", "available"), filter("zone-type", "availability-zone")},
	})
	if err != nil {
		return "", err
	}
	var names []string
	for _, z := range out.AvailabilityZones {
		if z.OptInStatus == ec2types.AvailabilityZoneOptInStatusOptInNotRequired {
			names = append(names, awssdk.ToString(z.ZoneName))
		}
	}
	if len(names) == 0 {
		return "", fmt.Errorf("no standard Availability Zones returned")
	}
	sort.Strings(names)
	return names[0], nil
}

func (b *Builder) resolveSubnetAZ(ctx context.Context) (string, error) {
	seen := map[string]bool{}
	var types []string
	for _, t := range b.cfg.Sizes {
		if t != "" && !seen[t] {
			seen[t] = true
			types = append(types, t)
		}
	}
	if len(types) == 0 {
		return "", nil // nothing to constrain; let AWS pick the subnet's AZ
	}
	// Intersect the AZs that offer each configured instance type.
	counts := map[string]int{}
	for _, t := range types {
		out, err := b.ec2.DescribeInstanceTypeOfferings(ctx, &ec2.DescribeInstanceTypeOfferingsInput{
			LocationType: ec2types.LocationTypeAvailabilityZone,
			Filters:      []ec2types.Filter{filter("instance-type", t)},
		})
		if err != nil {
			return "", fmt.Errorf("listing AZs for instance type %q: %w", t, err)
		}
		az := map[string]bool{}
		for _, o := range out.InstanceTypeOfferings {
			az[awssdk.ToString(o.Location)] = true
		}
		for a := range az {
			counts[a]++
		}
	}
	var common []string
	for a, n := range counts {
		if n == len(types) {
			common = append(common, a)
		}
	}
	if len(common) == 0 {
		return "", fmt.Errorf("no single Availability Zone offers all configured instance types %v -- choose types that share an AZ", types)
	}
	sort.Strings(common) // deterministic: same AZ every time, so all subnets co-locate
	return common[0], nil
}

// collapseIfAllNetworks replaces a host's per-network source CIDRs with ONE
// covering supernet when the sources include every team network -- "visible from
// everything," so one block is equivalent and cuts AWS's per-(port x source) rule
// count back to one source per port. Only when ALL networks are present: with a
// partial set, a supernet could silently widen access to a network the policy
// excluded, so sources are returned unchanged. Defensive: if the supernet can't
// be computed or comes out implausibly broad (< /8), keep the explicit sources.
func collapseIfAllNetworks(sources, allNetworkCIDRs []string) []string {
	if len(sources) < 2 || len(allNetworkCIDRs) == 0 {
		return sources
	}
	have := make(map[string]bool, len(sources))
	for _, s := range sources {
		have[s] = true
	}
	for _, c := range allNetworkCIDRs {
		if !have[c] {
			return sources // a team network isn't a source -> can't widen safely
		}
	}
	super, prefix, ok := minimalSupernet(allNetworkCIDRs)
	if !ok || prefix < 8 {
		return sources
	}
	return []string{super}
}

// minimalSupernet returns the smallest single CIDR that contains all of the
// given IPv4 CIDRs, with its prefix length. Pure, unit-tested offline.
func minimalSupernet(cidrs []string) (cidr string, prefix int, ok bool) {
	var lo, hi uint32
	first := true
	for _, c := range cidrs {
		_, ipnet, err := net.ParseCIDR(c)
		if err != nil {
			return "", 0, false
		}
		base := ipnet.IP.To4()
		mask := net.IP(ipnet.Mask).To4()
		if base == nil || mask == nil {
			return "", 0, false
		}
		n := binary.BigEndian.Uint32(base)
		b := n | ^binary.BigEndian.Uint32(mask) // broadcast of this block
		if first {
			lo, hi, first = n, b, false
			continue
		}
		if n < lo {
			lo = n
		}
		if b > hi {
			hi = b
		}
	}
	if first {
		return "", 0, false
	}
	// The minimal power-of-two block covering [lo, hi]: the prefix is the count of
	// leading bits lo and hi share once aligned, found from the highest differing bit.
	bits := 0
	for x := lo ^ hi; x > 0; x >>= 1 {
		bits++
	}
	prefix = 32 - bits
	maskBits := ^uint32(0)
	if prefix < 32 {
		maskBits <<= uint(32 - prefix)
	}
	network := lo & maskBits
	out := make(net.IP, 4)
	binary.BigEndian.PutUint32(out, network)
	return fmt.Sprintf("%s/%d", out.String(), prefix), prefix, true
}

// awsReservedAddress reports whether ip falls in AWS's per-subnet reserved range,
// which AWS forbids assigning to an instance: the FIRST FOUR addresses (network,
// VPC router, Amazon DNS, future use) and the BROADCAST address. reason names
// which, for a clear error. This is why content that places a host at .1/.2/.3 --
// fine on Incus/MicroCloud, where only .1 is the gateway -- fails on AWS.
func awsReservedAddress(cidr, ip string) (reserved bool, reason string) {
	_, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return false, ""
	}
	addr := net.ParseIP(ip).To4()
	base := ipnet.IP.To4()
	m := net.IP(ipnet.Mask).To4()
	if addr == nil || base == nil || m == nil || !ipnet.Contains(addr) {
		return false, ""
	}
	a := binary.BigEndian.Uint32(addr)
	net0 := binary.BigEndian.Uint32(base)
	broadcast := net0 | ^binary.BigEndian.Uint32(m)
	switch {
	case a >= net0 && a <= net0+3:
		return true, fmt.Sprintf("one of the first four (network, router, DNS, reserved) addresses of %s", cidr)
	case a == broadcast:
		return true, fmt.Sprintf("the broadcast address of %s", cidr)
	}
	return false, ""
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
	subnetID, vpcID, cidr, az, err := b.subnetForNetwork(ctx, spec.Network)
	if err != nil {
		return "", err
	}
	// The subnet's AZ decides where the instance lands, and a legacy AZ (e.g.
	// us-east-1e) offers no t3/current types -- RunInstances then fails with an
	// opaque, retried 400. Catch it here as a clear PERMANENT error naming the
	// AZs that WOULD work, so the fix (tear down + redeploy so the subnet is
	// recreated in a good AZ -- see subnetAZ -- and grant
	// ec2:DescribeInstanceTypeOfferings so auto-pinning works) is obvious instead
	// of three rounds of the AWS message. Best-effort: skipped if the offerings
	// lookup itself can't run, so a missing permission never turns into a false
	// block -- RunInstances stays the backstop.
	if az != "" {
		if offered, supported, offErr := b.azOffersType(ctx, az, instanceType); offErr == nil && !offered {
			return "", fmt.Errorf("host %q: its network's subnet is in Availability Zone %s, which does not offer instance type %q (supported in this region: %s) -- tear down and redeploy so the subnet is recreated in a supported AZ, and ensure the runner credential has ec2:DescribeInstanceTypeOfferings so AZ auto-pinning works: %w",
				spec.ExternalName, az, instanceType, strings.Join(supported, ", "), builder.ErrPermanentConfig)
		}
	}
	// A host whose content address lands in AWS's reserved range can never launch
	// (AWS rejects it outright), so fail now with a clear, PERMANENT error -- no
	// point retrying -- instead of three rounds of the opaque AWS message.
	if spec.Address != "" {
		if reserved, reason := awsReservedAddress(cidr, spec.Address); reserved {
			return "", fmt.Errorf("host %q address %s is %s, which AWS reserves and won't assign -- use a last_octet of 4 or higher (AWS keeps the first four and last of every subnet): %w",
				spec.ExternalName, spec.Address, reason, builder.ErrPermanentConfig)
		}
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
	// Associate the configured EC2 key pair so the operator can decrypt a Windows
	// box's admin password with `aws ec2 get-password-data --priv-launch-key`.
	// Harmless on Linux (the key lands in the default user's authorized_keys; the
	// agent, not SSH, drives the host). If the named key pair doesn't exist in the
	// region, RunInstances fails loudly with InvalidKeyPair.NotFound -- the right
	// outcome, since launching without it would silently lock Windows out again.
	if b.cfg.KeyName != "" {
		in.KeyName = awssdk.String(b.cfg.KeyName)
	}
	if spec.AgentDownloadURL != "" {
		// Render the agent install for THIS image's mechanism: a stock AWS Linux
		// AMI runs cloud-init, a stock AWS Windows AMI runs EC2Launch
		// (<powershell>), NOT cloudbase-init -- so a Windows box would boot clean
		// but never install the agent if handed the cloudbase form. EC2 user-data
		// must be base64.
		var data string
		// No config drive on EC2, so the agent is always downloaded (not embedded);
		// the temporary admin password still rides user-data so a host is reachable
		// before its agent runs.
		if agentdelivery.PlatformFor(spec.OS) == agentdelivery.Windows {
			data = agentdelivery.WindowsEC2LaunchUserData(spec.AgentDownloadURL, spec.TempAdminPassword)
		} else {
			data = agentdelivery.LinuxUserData(spec.AgentDownloadURL, spec.TempAdminPassword)
		}
		in.UserData = awssdk.String(base64.StdEncoding.EncodeToString([]byte(data)))
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
	// Resolve externalRef to a real instance id. The runner hands us the stored
	// external_ref when there is one (an "i-..." id), but falls back to the
	// deterministic ExternalName when the object has none -- e.g. its deploy
	// failed before RunInstances ever ran. In that case look the instance up by
	// its laforge:external-name tag; an empty result means nothing was created,
	// so there's nothing to terminate and destroy is a no-op (minus any SG/EIP a
	// partial deploy may have made, cleaned below).
	instanceID := externalRef
	if !strings.HasPrefix(instanceID, "i-") {
		id, err := b.findInstance(ctx, externalRef)
		if err != nil {
			return err
		}
		instanceID = id
	}
	if instanceID != "" {
		// Release any Elastic IP we attached for external access, so a teardown
		// doesn't leak (and keep paying for) an allocation.
		b.releaseEIPForRef(ctx, instanceID)
		if _, err := b.ec2.TerminateInstances(ctx, &ec2.TerminateInstancesInput{InstanceIds: []string{instanceID}}); err != nil && !isNotFound(err) && !isMalformed(err) {
			return fmt.Errorf("terminating instance %s: %w", instanceID, err)
		}
	}
	// Delete the host's per-object security group (found by the ref it was tagged
	// with at deploy). If we don't, it leaks AND blocks the team VPC teardown
	// later (a VPC won't delete while a non-default SG remains in it). The SG
	// can't go while the instance's ENI still references it, so a
	// DependencyViolation here just means the instance is still terminating --
	// deleteObjectSecurityGroup returns that as a retryable error so the task runs
	// again and succeeds once the ENI is gone. A failed-deploy SG tagged by name
	// only (never tagged with a ref) isn't found here and is instead swept by the
	// VPC teardown.
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

	// Single-image containers are Fargate TASKS, not EC2 instances, so list them
	// too -- otherwise the power-state poll (which marks any object whose
	// external_ref Inspect doesn't return as "missing") reports every running
	// container as missing. (A compose container is an EC2 instance and is already
	// covered by the DescribeInstances pass above.)
	tasks, err := b.inspectTasks(ctx)
	if err != nil {
		return nil, err
	}
	out = append(out, tasks...)
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
	allCIDRs := make([]string, 0, len(networks))
	for _, na := range networks {
		cidrByName[na.ExternalName] = na.CIDR
		allCIDRs = append(allCIDRs, na.CIDR)
	}
	for _, na := range networks {
		sources := []string{na.CIDR}
		for _, from := range na.VisibleFrom {
			if c := cidrByName[from]; c != "" {
				sources = append(sources, c)
			}
		}
		// AWS counts a rule per (port x source CIDR), capped at 60 per SG by
		// default -- so a many-ported host visible from several networks blows the
		// limit (16 ports x 4 networks = 64). When a host sees EVERY team network,
		// its sources cover the whole VPC, so one supernet is equivalent and cuts
		// the rules back to one per port. Partial visibility can't widen safely and
		// is left as-is.
		sources = collapseIfAllNetworks(sources, allCIDRs)
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
			if isRuleLimitExceeded(err) {
				// Rules = declared ports x allowed sources; a many-ported, widely
				// visible host exceeds AWS's 60-rule default. Retrying can't help, so
				// fail permanently with the fix: raise the quota or trim the policy.
				return fmt.Errorf("%w: %s needs more than AWS's limit of security-group rules (ports x visible-from sources). Raise the \"Inbound/outbound rules per security group\" quota (Service Quotas, default 60, up to 1000), or reduce this host's ports:/visible_from: %w",
					builder.ErrPermanentConfig, h.ExternalRef, err)
			}
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
