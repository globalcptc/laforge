// Containers on AWS take two shapes. A single-image container runs as an
// ECS/Fargate task (this file); a compose container runs the same way as on every
// other builder -- on a Docker-capable machine the agent runs `docker compose up`
// on -- which here is an EC2 instance booted from the `compose-host` image (see
// DeployContainer's ComposeHost branch), because Fargate forbids the nested Docker
// a real compose project needs.
//
// The Fargate model mirrors the Incus native-container path (the LaForge agent
// runs as the container's supervising entrypoint so a container checks in and runs
// steps/validators like a host), adapted to Fargate's constraints:
//
//   - Fargate pulls the image and runs it -- there's no file API to push the
//     agent in -- so the agent is DOWNLOADED at start: an init container
//     (a small public image) fetches the patched agent by its one-time-token URL
//     onto a shared volume, and the app container runs it as its entrypoint.
//   - Fargate doesn't hand back the image's own entrypoint, so the command the
//     agent supervises comes from the content `command:` if set, else is fetched
//     from the image's registry config (imageEntrypoint).
//
// Not yet exercised against a real AWS account. Known Fargate (single-image)
// limits: a static Address isn't applied (the ENI gets a dynamic IP); the app
// image must have a shell for the agent's `sh -c`; the one-time download token
// won't survive a task restart; imageEntrypoint only does anonymous Docker Hub
// pulls (set `command:` to avoid it).
package aws

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"

	"github.com/globalcptc/laforge/internal/builder"
	"github.com/globalcptc/laforge/internal/builder/imagemeta"
)

const (
	agentMountPath      = "/laforge"
	agentBinInContainer = agentMountPath + "/laforge-agent"
	agentVolumeName     = "laforge-agent"
	agentInitContainer  = "laforge-agent-init"
)

// DeployContainer runs a LaForge `container:` as an ECS/Fargate task. externalRef
// is the task ARN. DRAFT -- see the file header.
func (b *Builder) DeployContainer(ctx context.Context, spec builder.ContainerSpec) (string, error) {
	if spec.ComposeHost {
		// A compose project needs a real Docker daemon, which Fargate can't give
		// (no privileged/nested Docker). So -- exactly like the other builders --
		// it runs on a Docker-capable machine the agent runs `docker compose up`
		// on; on AWS that machine is an EC2 instance booted from the `compose-host`
		// image. DeployHost gives it its per-object SG, agent delivery, firewall
		// and external-access handling for free.
		return b.DeployHost(ctx, spec.ComposeHostSpec())
	}
	if b.cfg.ExecutionRoleARN == "" {
		return "", fmt.Errorf("AWS Fargate needs an ECS task execution role -- set the builder's execution role ARN")
	}
	// Ensure the ECS cluster exists before RunTask -- AWS does NOT auto-create a
	// cluster (not even "default"), so a missing one is a flat
	// ClusterNotFoundException. Like the VPC/IGW/SG, the builder provisions it.
	if err := b.ensureCluster(ctx); err != nil {
		return "", err
	}
	// Ensure the CloudWatch log group exists so the awslogs driver can write --
	// otherwise Fargate fails the container at startup (or, with no group
	// configured at all, a crash leaves no logs). Only for the default awslogs
	// path; an environment's own container_logs driver manages its own sink.
	if spec.LogDriver == "" {
		if err := b.ensureLogGroup(ctx); err != nil {
			return "", err
		}
	}
	subnetID, vpcID, err := b.subnetForNetwork(ctx, spec.Network)
	if err != nil {
		return "", fmt.Errorf("resolving subnet for container %q: %w", spec.ExternalName, err)
	}
	// Per-object SG, like a host. The task's SG membership is fixed at launch,
	// but ConfigureNetworkAccess converges this SG's RULES afterward, so the
	// container gets the same port firewall as an EC2 host.
	sgID, err := b.ensureObjectSecurityGroup(ctx, spec.ExternalName, spec.Team, vpcID)
	if err != nil {
		return "", err
	}

	// The command the agent supervises: content Command if set, else the image's
	// own entrypoint (Fargate doesn't expose it, so fetch it from the registry).
	// Shell-quote each argv element -- the agent runs this via `sh -c`, so a
	// space-join would re-split a multi-word arg (e.g. nginx's "daemon off;").
	supervise := imagemeta.ShellJoin(spec.Command)
	if supervise == "" && len(spec.AgentDownloadURL) > 0 {
		ep, err := imagemeta.Entrypoint(ctx, spec.Image)
		if err != nil {
			return "", fmt.Errorf("resolving entrypoint of image %q (set command: on the container to avoid this): %w", spec.Image, err)
		}
		supervise = ep
	}

	cpu, mem := fargateSize(b.cfg.Sizes[spec.Size])
	appName := sanitizeECSName(spec.ExternalName)

	var appEnv []ecstypes.KeyValuePair
	for k, v := range spec.Env {
		appEnv = append(appEnv, ecstypes.KeyValuePair{Name: awssdk.String(k), Value: awssdk.String(v)})
	}

	var containers []ecstypes.ContainerDefinition
	var volumes []ecstypes.Volume
	var appEntry []string
	var appDeps []ecstypes.ContainerDependency
	var appMounts []ecstypes.MountPoint

	// With agent delivery on, plant the agent via an init container onto a shared
	// volume and make it the app's entrypoint, supervising the image's command.
	if spec.AgentDownloadURL != "" {
		volumes = append(volumes, ecstypes.Volume{Name: awssdk.String(agentVolumeName)})
		agentMount := ecstypes.MountPoint{SourceVolume: awssdk.String(agentVolumeName), ContainerPath: awssdk.String(agentMountPath)}
		containers = append(containers, ecstypes.ContainerDefinition{
			Name:             awssdk.String(agentInitContainer),
			Image:            awssdk.String(b.cfg.AgentInitImage),
			Essential:        awssdk.Bool(false),
			EntryPoint:       []string{"sh", "-c"},
			Command:          []string{fmt.Sprintf("set -e; wget -O %s %q; chmod +x %s", agentBinInContainer, spec.AgentDownloadURL, agentBinInContainer)},
			MountPoints:      []ecstypes.MountPoint{agentMount},
			LogConfiguration: b.logConfig("agent-init", spec.ExternalName),
		})
		appEnv = append(appEnv, ecstypes.KeyValuePair{Name: awssdk.String("LAFORGE_SUPERVISE"), Value: awssdk.String(supervise)})
		appEntry = []string{agentBinInContainer}
		appDeps = []ecstypes.ContainerDependency{{ContainerName: awssdk.String(agentInitContainer), Condition: ecstypes.ContainerConditionSuccess}}
		appMounts = []ecstypes.MountPoint{agentMount}
	}

	containers = append(containers, ecstypes.ContainerDefinition{
		Name:             awssdk.String(appName),
		Image:            awssdk.String(spec.Image),
		Essential:        awssdk.Bool(true),
		Environment:      appEnv,
		EntryPoint:       appEntry,
		DependsOn:        appDeps,
		MountPoints:      appMounts,
		PortMappings:     portMappings(spec.TCPPorts, spec.UDPPorts),
		LogConfiguration: b.appLogConfig(spec),
	})

	family := "laforge-" + appName
	reg, err := b.ecs.RegisterTaskDefinition(ctx, &ecs.RegisterTaskDefinitionInput{
		Family:                  awssdk.String(family),
		RequiresCompatibilities: []ecstypes.Compatibility{ecstypes.CompatibilityFargate},
		NetworkMode:             ecstypes.NetworkModeAwsvpc,
		Cpu:                     awssdk.String(cpu),
		Memory:                  awssdk.String(mem),
		ExecutionRoleArn:        awssdk.String(b.cfg.ExecutionRoleARN),
		RuntimePlatform:         &ecstypes.RuntimePlatform{CpuArchitecture: ecstypes.CPUArchitectureX8664, OperatingSystemFamily: ecstypes.OSFamilyLinux},
		ContainerDefinitions:    containers,
		Volumes:                 volumes,
	})
	if err != nil {
		return "", fmt.Errorf("registering task definition for %q: %w", spec.ExternalName, err)
	}

	run, err := b.ecs.RunTask(ctx, &ecs.RunTaskInput{
		Cluster:        awssdk.String(b.cfg.Cluster),
		TaskDefinition: reg.TaskDefinition.TaskDefinitionArn,
		LaunchType:     ecstypes.LaunchTypeFargate,
		Count:          awssdk.Int32(1),
		NetworkConfiguration: &ecstypes.NetworkConfiguration{
			AwsvpcConfiguration: &ecstypes.AwsVpcConfiguration{
				Subnets:        []string{subnetID},
				SecurityGroups: []string{sgID},
				AssignPublicIp: ecstypes.AssignPublicIpEnabled, // needed to pull the image + reach the API/gateway
			},
		},
		Tags: []ecstypes.Tag{
			{Key: awssdk.String(laforgeManagedTag), Value: awssdk.String("true")},
			{Key: awssdk.String(laforgeExternalNameTag), Value: awssdk.String(spec.ExternalName)},
			{Key: awssdk.String(laforgeTeamTag), Value: awssdk.String(spec.Team)},
		},
		StartedBy: awssdk.String("laforge"),
	})
	if err != nil {
		return "", fmt.Errorf("running Fargate task for %q: %w", spec.ExternalName, err)
	}
	if len(run.Tasks) == 0 {
		reason := "no reason reported"
		if len(run.Failures) > 0 {
			reason = awssdk.ToString(run.Failures[0].Reason)
		}
		return "", fmt.Errorf("Fargate RunTask for %q returned no task: %s", spec.ExternalName, reason)
	}
	arn := awssdk.ToString(run.Tasks[0].TaskArn)
	// Tag the object SG with the task ARN so ConfigureNetworkAccess finds it from
	// the externalRef the runner hands back (it never gets the ExternalName).
	b.tagSGWithRef(ctx, sgID, arn)
	return arn, nil
}

// ensureCluster makes sure the configured ECS cluster exists, creating it if
// absent. A plain Fargate cluster has no instances and no standing cost, so --
// like the VPC/IGW/SG the builder already ensure-creates -- LaForge provisions
// it rather than making the operator pre-create one. AWS does NOT auto-create
// any cluster, including "default" (the config default), so even that name needs
// this. The cluster is account/region-scoped and SHARED across every team and
// build, so teardown never deletes it (a Fargate cluster at rest is free).
func (b *Builder) ensureCluster(ctx context.Context) error {
	name := b.cfg.Cluster
	if name == "" {
		name = "default"
	}
	if out, err := b.ecs.DescribeClusters(ctx, &ecs.DescribeClustersInput{Clusters: []string{name}}); err == nil {
		for _, c := range out.Clusters {
			if awssdk.ToString(c.Status) == "ACTIVE" {
				return nil
			}
		}
	}
	if _, err := b.ecs.CreateCluster(ctx, &ecs.CreateClusterInput{
		ClusterName: awssdk.String(name),
		Tags:        []ecstypes.Tag{{Key: awssdk.String(laforgeManagedTag), Value: awssdk.String("true")}},
	}); err != nil {
		return fmt.Errorf("ensuring ECS cluster %q exists: %w", name, err)
	}
	return nil
}

// inspectTasks lists the cluster's running LaForge Fargate tasks as container
// Resources, so the power-state poll finds a deployed container by its task ARN
// (external_ref) instead of marking it missing. StartedBy="laforge" scopes to
// our own tasks; desired-status RUNNING means a stopped/failed task is correctly
// absent (reported missing). A not-yet-created cluster simply has no tasks.
func (b *Builder) inspectTasks(ctx context.Context) ([]builder.Resource, error) {
	cluster := b.cfg.Cluster
	if cluster == "" {
		cluster = "default"
	}
	var out []builder.Resource
	var next *string
	for {
		list, err := b.ecs.ListTasks(ctx, &ecs.ListTasksInput{
			Cluster:       awssdk.String(cluster),
			StartedBy:     awssdk.String("laforge"),
			DesiredStatus: ecstypes.DesiredStatusRunning,
			NextToken:     next,
		})
		if err != nil {
			if isECSClusterNotFound(err) {
				return out, nil // no cluster yet -> no tasks
			}
			return nil, fmt.Errorf("listing ECS tasks: %w", err)
		}
		for _, arn := range list.TaskArns {
			out = append(out, builder.Resource{ExternalRef: arn, Kind: "container", State: builder.PowerStateRunning})
		}
		if list.NextToken == nil {
			return out, nil
		}
		next = list.NextToken
	}
}

// ensureLogGroup makes sure the awslogs target log group exists, creating it if
// absent. The ECS task execution role (AWS-managed AmazonECSTaskExecutionRolePolicy)
// can write streams + events but NOT create the group, and the awslogs driver
// won't create a missing group on its own -- so without this a container either
// fails to start or (with no group configured) crashes with no captured output.
// Idempotent: a ResourceAlreadyExists is success. No-op if logging isn't awslogs.
func (b *Builder) ensureLogGroup(ctx context.Context) error {
	if b.logs == nil || b.cfg.LogGroup == "" {
		return nil
	}
	_, err := b.logs.CreateLogGroup(ctx, &cloudwatchlogs.CreateLogGroupInput{
		LogGroupName: awssdk.String(b.cfg.LogGroup),
		Tags:         map[string]string{laforgeManagedTag: "true"},
	})
	if err != nil && !isLogGroupExists(err) {
		return fmt.Errorf("ensuring CloudWatch log group %q exists: %w", b.cfg.LogGroup, err)
	}
	return nil
}

// DestroyContainer stops the Fargate task and removes what the deploy created
// alongside it: the task-definition family (deregistered) and the per-object
// security group. Idempotent: a missing/stopped task, an already-gone task def,
// and an absent SG are all not errors.
func (b *Builder) DestroyContainer(ctx context.Context, team, externalRef string) error {
	if externalRef == "" {
		return nil
	}
	// A compose container is an EC2 instance (see DeployContainer's ComposeHost
	// branch); its ref is an instance id, not a task ARN. Route it to DestroyHost
	// (which already tears down its SG and EIP).
	if strings.HasPrefix(externalRef, "i-") {
		return b.DestroyHost(ctx, team, externalRef)
	}
	// Only a real task ARN is a task we can stop. When the object never recorded
	// one (its deploy failed before RunTask), the runner falls back to the
	// deterministic ExternalName -- there's no task to stop, so skip straight to
	// SG cleanup. A missing cluster or task is "already gone," never a hard error,
	// so a config mistake (wrong ECS cluster) can't wedge teardown forever.
	if strings.HasPrefix(externalRef, "arn:") || strings.Contains(externalRef, ":task/") {
		out, err := b.ecs.StopTask(ctx, &ecs.StopTaskInput{
			Cluster: awssdk.String(b.cfg.Cluster),
			Task:    awssdk.String(externalRef),
			Reason:  awssdk.String("laforge teardown"),
		})
		if err != nil && !isNotFound(err) && !isMalformed(err) && !isECSClusterNotFound(err) {
			return fmt.Errorf("stopping Fargate task %s: %w", externalRef, err)
		}
		// Deregister the task-definition family this container registered
		// (best-effort -- a leaked INACTIVE task def costs nothing, so it never
		// blocks teardown).
		if err == nil && out.Task != nil {
			b.deregisterTaskFamily(ctx, awssdk.ToString(out.Task.TaskDefinitionArn))
		}
	}
	// Delete the per-object SG. A stopping task's ENI still references it, so a
	// DependencyViolation here is "task still draining" -- returned as retryable
	// so the SG (and later the VPC) don't leak.
	return b.deleteObjectSecurityGroup(ctx, externalRef)
}

// deregisterTaskFamily deregisters every revision of the task-definition family
// named by taskDefArn ("arn:...:task-definition/laforge-<name>:<rev>"). Each
// DeployContainer registers a new revision, so teardown clears the whole family.
// Best-effort throughout; a listing or deregister hiccup just leaves an inert
// INACTIVE revision behind.
func (b *Builder) deregisterTaskFamily(ctx context.Context, taskDefArn string) {
	family := taskFamilyFromARN(taskDefArn)
	if family == "" {
		return
	}
	var next *string
	for {
		list, err := b.ecs.ListTaskDefinitions(ctx, &ecs.ListTaskDefinitionsInput{
			FamilyPrefix: awssdk.String(family),
			NextToken:    next,
		})
		if err != nil {
			return
		}
		for _, arn := range list.TaskDefinitionArns {
			// FamilyPrefix is a prefix match, so guard against a sibling family
			// (e.g. laforge-web vs laforge-web2) by comparing the family exactly.
			if taskFamilyFromARN(arn) != family {
				continue
			}
			_, _ = b.ecs.DeregisterTaskDefinition(ctx, &ecs.DeregisterTaskDefinitionInput{TaskDefinition: awssdk.String(arn)})
		}
		if list.NextToken == nil {
			return
		}
		next = list.NextToken
	}
}

// taskFamilyFromARN pulls the family out of a task-definition ARN or a
// "family:revision" string: "arn:...:task-definition/laforge-web01:3" ->
// "laforge-web01". Returns "" if there's nothing usable.
func taskFamilyFromARN(arn string) string {
	s := arn
	if i := strings.LastIndex(s, "/"); i >= 0 {
		s = s[i+1:] // drop the arn prefix, keep "laforge-web01:3"
	}
	if i := strings.LastIndex(s, ":"); i >= 0 {
		s = s[:i] // drop ":<revision>"
	}
	return s
}

// fargateSize maps a content size to Fargate CPU/memory units. The shared Sizes
// map holds EC2 instance types, which don't apply to Fargate, so a "cpu/memory"
// value (e.g. "512/1024") is honored and anything else falls back to the smallest
// valid combo. DRAFT.
func fargateSize(s string) (cpu, mem string) {
	if a, b, ok := strings.Cut(s, "/"); ok && a != "" && b != "" {
		return a, b
	}
	return "256", "512"
}

func sanitizeECSName(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	out := b.String()
	if out == "" {
		out = "container"
	}
	if len(out) > 200 {
		out = out[:200]
	}
	return out
}

func portMappings(tcp, udp []string) []ecstypes.PortMapping {
	var out []ecstypes.PortMapping
	add := func(ports []string, proto ecstypes.TransportProtocol) {
		for _, p := range ports {
			if n, err := strconv.Atoi(strings.TrimSpace(p)); err == nil {
				out = append(out, ecstypes.PortMapping{ContainerPort: awssdk.Int32(int32(n)), Protocol: proto})
			}
		}
	}
	add(tcp, ecstypes.TransportProtocolTcp)
	add(udp, ecstypes.TransportProtocolUdp)
	return out
}

func (b *Builder) logConfig(streamPrefix, externalName string) *ecstypes.LogConfiguration {
	if b.cfg.LogGroup == "" {
		return nil
	}
	return &ecstypes.LogConfiguration{
		LogDriver: ecstypes.LogDriverAwslogs,
		Options: map[string]string{
			"awslogs-group":         b.cfg.LogGroup,
			"awslogs-region":        b.cfg.Region,
			"awslogs-stream-prefix": streamPrefix + "-" + externalName,
		},
	}
}

// appLogConfig is the app container's logging. When the environment set
// container_logs, Fargate ships the app's console output NATIVELY via that
// driver (e.g. "splunk" straight to HEC) -- the LaForge agent never touches it,
// which is what keeps the gateway out of the data path at scale. With no
// container_logs it falls back to the awslogs default (if a LogGroup is set).
func (b *Builder) appLogConfig(spec builder.ContainerSpec) *ecstypes.LogConfiguration {
	if spec.LogDriver != "" {
		return &ecstypes.LogConfiguration{
			LogDriver: ecstypes.LogDriver(spec.LogDriver),
			Options:   spec.LogOptions,
		}
	}
	return b.logConfig("app", spec.ExternalName)
}

// Image entrypoint resolution (the Docker Hub registry dance) moved to
// internal/builder/imagemeta -- OpenStack Zun needs it identically, so it lives
// once there rather than duplicated per builder.
