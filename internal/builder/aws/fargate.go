// Containers on AWS are ECS/Fargate tasks, not EC2 VMs -- a different shape from
// the rest of this builder. This file is a DRAFT: it has never run against a real
// AWS account. The model mirrors the Incus native-container path (the LaForge
// agent runs as the container's supervising entrypoint so a container checks in
// and runs steps/validators like a host), adapted to Fargate's constraints:
//
//   - Fargate pulls the image and runs it -- there's no file API to push the
//     agent in -- so the agent is DOWNLOADED at start: an init container
//     (a small public image) fetches the patched agent by its one-time-token URL
//     onto a shared volume, and the app container runs it as its entrypoint.
//   - Fargate doesn't hand back the image's own entrypoint (Incus did), so the
//     command the agent supervises comes from the content `command:` if set,
//     else is fetched from the image's registry config (imageEntrypoint).
//
// Account-specific inputs (ECS cluster, task execution role, log group) come from
// the environment -- see Config. Known DRAFT gaps: a static Address isn't applied
// (Fargate's ENI gets a dynamic IP); the app image must have a shell for the
// agent's `sh -c`; the one-time download token won't survive a task restart;
// imageEntrypoint only does anonymous Docker Hub pulls.
package aws

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
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
	if b.cfg.ExecutionRoleARN == "" {
		return "", fmt.Errorf("AWS Fargate needs an ECS task execution role -- set AWS_ECS_EXECUTION_ROLE_ARN")
	}
	subnetID, err := b.subnetForNetwork(ctx, spec.Network)
	if err != nil {
		return "", fmt.Errorf("resolving subnet for container %q: %w", spec.ExternalName, err)
	}
	sgID, err := b.ensureTeamSecurityGroup(ctx, spec.Team, subnetID)
	if err != nil {
		return "", err
	}

	// The command the agent supervises: content Command if set, else the image's
	// own entrypoint (Fargate doesn't expose it, so fetch it from the registry).
	supervise := strings.Join(spec.Command, " ")
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
		LogConfiguration: b.logConfig("app", spec.ExternalName),
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
	return awssdk.ToString(run.Tasks[0].TaskArn), nil
}

// DestroyContainer stops the Fargate task. Idempotent: a missing/stopped task is
// not an error.
func (b *Builder) DestroyContainer(ctx context.Context, team, externalRef string) error {
	if externalRef == "" {
		return nil
	}
	_, err := b.ecs.StopTask(ctx, &ecs.StopTaskInput{
		Cluster: awssdk.String(b.cfg.Cluster),
		Task:    awssdk.String(externalRef),
		Reason:  awssdk.String("laforge teardown"),
	})
	if err != nil && !isNotFound(err) {
		return fmt.Errorf("stopping Fargate task %s: %w", externalRef, err)
	}
	return nil
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

// Image entrypoint resolution (the Docker Hub registry dance) moved to
// internal/builder/imagemeta -- OpenStack Zun needs it identically, so it lives
// once there rather than duplicated per builder.
