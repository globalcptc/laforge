package incus

import (
	"context"
	"fmt"
	"strings"

	"github.com/gen0cide/laforge/ent"
	incusclient "github.com/lxc/incus/v6/client"
	"github.com/lxc/incus/v6/shared/api"
)

func (builder *IncusBuilder) DeployHost(
	ctx context.Context,
	provisionedHost *ent.ProvisionedHost,
) error {
	if err := builder.acquireDeployWorker(ctx); err != nil {
		return err
	}
	defer builder.DeployWorkerPool.Release(1)

	host, err := provisionedHost.QueryHost().Only(ctx)
	if err != nil {
		return fmt.Errorf("failed to query host for provisioned host %s: %w", provisionedHost.ID, err)
	}
	disk, err := host.QueryDisk().Only(ctx)
	if err != nil {
		return fmt.Errorf("failed to query disk for host %q: %w", host.Hostname, err)
	}
	provisionedNetwork, err := provisionedHost.QueryProvisionedNetwork().Only(ctx)
	if err != nil {
		return fmt.Errorf("failed to query network for host %q: %w", host.Hostname, err)
	}
	team, err := provisionedNetwork.QueryTeam().Only(ctx)
	if err != nil {
		return fmt.Errorf("failed to query team for host %q: %w", host.Hostname, err)
	}
	build, err := provisionedHost.QueryBuild().Only(ctx)
	if err != nil {
		return fmt.Errorf("failed to query build for host %q: %w", host.Hostname, err)
	}
	competition, err := build.QueryCompetition().Only(ctx)
	if err != nil {
		return fmt.Errorf("failed to query competition for host %q: %w", host.Hostname, err)
	}
	agentFile, err := provisionedHost.QueryGinFileMiddleware().Only(ctx)
	if err != nil {
		return fmt.Errorf("failed to query agent download for host %q: %w", host.Hostname, err)
	}

	client, hostConfig, project, err := builder.teamTarget(team)
	if err != nil {
		return err
	}
	bridge := provisionedNetwork.Vars[varNetwork]
	if bridge == "" {
		return fmt.Errorf("network %q is missing %s", provisionedNetwork.Name, varNetwork)
	}
	if provisionedHost.SubnetIP == "" {
		return fmt.Errorf("provisioned host %q has no subnet IP", host.Hostname)
	}

	size, ok := builder.Config.InstanceSizes[host.InstanceSize]
	if !ok {
		return fmt.Errorf("no Incus instance size configured for %q", host.InstanceSize)
	}
	image, ok := builder.Config.Images[host.OS]
	if !ok {
		return fmt.Errorf("no Incus image configured for OS %q", host.OS)
	}

	password := competition.RootPassword
	if host.OverridePassword != "" {
		password = host.OverridePassword
	}
	agentURL := fmt.Sprintf(
		"%s/api/download/%s",
		strings.TrimRight(builder.Config.LaForgeServerURL, "/"),
		agentFile.URLID,
	)

	// Register the DHCP reservation and any ingress before the VM boots so
	// its first DHCP request is answered.
	if err := builder.reconcileGateway(ctx, team, reconcileExclusions{}); err != nil {
		return err
	}

	instanceConfig := map[string]string{
		"limits.cpu":           size.CPU,
		"limits.memory":        size.Memory,
		"cloud-init.user-data": userData(host.OS, agentURL, password),
	}
	for key, value := range builder.Config.ImageConfig[host.OS] {
		instanceConfig[key] = value
	}

	name := instanceName(host, build)
	instancePut := api.InstancePut{
		Description: fmt.Sprintf("LaForge team %d host %s", team.TeamNumber, host.Hostname),
		Config:      instanceConfig,
		Devices: map[string]map[string]string{
			"eth0": {
				"type":    "nic",
				"network": bridge,
				"name":    "eth0",
				"hwaddr":  macAddress(provisionedHost.ID.String()),
			},
			"root": {
				"type": "disk",
				"pool": hostConfig.StoragePool,
				"path": "/",
				"size": fmt.Sprintf("%dGiB", disk.Size),
			},
		},
		Profiles: []string{},
	}

	projectClient := client.UseProject(project)
	instance, _, err := projectClient.GetInstance(name)
	if err != nil && !isNotFound(err) {
		return fmt.Errorf("failed to get Incus instance %q: %w", name, err)
	}
	if isNotFound(err) {
		op, err := projectClient.CreateInstance(api.InstancesPost{
			Name:  name,
			Type:  api.InstanceTypeVM,
			Start: true,
			Source: api.InstanceSource{
				Type:  "image",
				Alias: image,
			},
			InstancePut: instancePut,
		})
		if err != nil {
			return fmt.Errorf("failed to create Incus instance %q: %w", name, err)
		}
		if err := op.Wait(); err != nil {
			return fmt.Errorf("failed waiting for Incus instance %q: %w", name, err)
		}
	} else {
		if err := validateExistingInstance(instance, instancePut); err != nil {
			return fmt.Errorf("existing Incus instance %q is not reusable: %w", name, err)
		}
		if instance.StatusCode == api.Stopped {
			if err := startInstance(projectClient, name); err != nil {
				return err
			}
		}
	}

	vars := cloneVars(provisionedHost.Vars)
	vars[varInstance] = name
	if host.Vars["public_ingress"] == "true" {
		vars["PublicIP"] = hostConfig.IngressListenAddress
	}
	if err := provisionedHost.Update().SetVars(vars).Exec(ctx); err != nil {
		return fmt.Errorf("failed to update vars for Incus instance %q: %w", name, err)
	}
	provisionedHost.Vars = vars

	return nil
}

func startInstance(client incusclient.InstanceServer, name string) error {
	op, err := client.UpdateInstanceState(name, api.InstanceStatePut{
		Action:  "start",
		Timeout: -1,
	}, "")
	if err != nil {
		return fmt.Errorf("failed to start existing Incus instance %q: %w", name, err)
	}
	if err := op.Wait(); err != nil {
		return fmt.Errorf("failed waiting for Incus instance %q to start: %w", name, err)
	}
	return nil
}

func validateExistingInstance(instance *api.Instance, desired api.InstancePut) error {
	if instance.StatusCode == api.Error || strings.EqualFold(instance.Status, "error") {
		return fmt.Errorf("instance is in error state")
	}
	if instance.Type != string(api.InstanceTypeVM) {
		return fmt.Errorf("type is %q, want %q", instance.Type, api.InstanceTypeVM)
	}

	for key, value := range desired.Config {
		if instance.Config[key] != value {
			return fmt.Errorf("config %q is %q, want %q", key, instance.Config[key], value)
		}
	}
	for deviceName, desiredDevice := range desired.Devices {
		device, ok := instance.Devices[deviceName]
		if !ok {
			return fmt.Errorf("device %q is missing", deviceName)
		}
		for key, value := range desiredDevice {
			if device[key] != value {
				return fmt.Errorf(
					"device %q property %q is %q, want %q",
					deviceName,
					key,
					device[key],
					value,
				)
			}
		}
	}

	return nil
}

func (builder *IncusBuilder) TeardownHost(
	ctx context.Context,
	provisionedHost *ent.ProvisionedHost,
) error {
	if err := builder.acquireTeardownWorker(ctx); err != nil {
		return err
	}
	defer builder.TeardownWorkerPool.Release(1)

	provisionedNetwork, err := provisionedHost.QueryProvisionedNetwork().Only(ctx)
	if err != nil {
		return fmt.Errorf("failed to query network for provisioned host %s: %w", provisionedHost.ID, err)
	}
	team, err := provisionedNetwork.QueryTeam().Only(ctx)
	if err != nil {
		return fmt.Errorf("failed to query team for provisioned host %s: %w", provisionedHost.ID, err)
	}
	if team.Vars[varHost] == "" {
		return nil
	}

	name := provisionedHost.Vars[varInstance]
	if name == "" {
		host, err := provisionedHost.QueryHost().Only(ctx)
		if err != nil {
			return fmt.Errorf("failed to query host for provisioned host %s: %w", provisionedHost.ID, err)
		}
		build, err := provisionedHost.QueryBuild().Only(ctx)
		if err != nil {
			return fmt.Errorf("failed to query build for provisioned host %s: %w", provisionedHost.ID, err)
		}
		name = instanceName(host, build)
	}

	client, _, project, err := builder.teamTarget(team)
	if err != nil {
		return err
	}
	if err := deleteInstance(client.UseProject(project), name); err != nil {
		return err
	}

	// Drop the DHCP reservation and any ingress for this host.
	if err := builder.reconcileGateway(ctx, team, reconcileExclusions{
		hosts: map[string]struct{}{provisionedHost.ID.String(): {}},
	}); err != nil {
		return err
	}

	vars := cloneVars(provisionedHost.Vars)
	delete(vars, varInstance)
	delete(vars, "PublicIP")
	if err := provisionedHost.Update().SetVars(vars).Exec(ctx); err != nil {
		return fmt.Errorf("failed to clear vars for provisioned host %s: %w", provisionedHost.ID, err)
	}
	provisionedHost.Vars = vars

	return nil
}

// deleteInstance force-stops and deletes an instance. A missing instance is
// not an error so teardown stays idempotent.
func deleteInstance(client incusclient.InstanceServer, name string) error {
	instance, _, err := client.GetInstance(name)
	if isNotFound(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to get Incus instance %q: %w", name, err)
	}

	if instance.StatusCode != api.Stopped {
		op, err := client.UpdateInstanceState(name, api.InstanceStatePut{
			Action:  "stop",
			Timeout: -1,
			Force:   true,
		}, "")
		if err != nil && !isNotFound(err) {
			return fmt.Errorf("failed to stop Incus instance %q: %w", name, err)
		}
		if err == nil {
			if err := op.Wait(); err != nil {
				return fmt.Errorf("failed waiting for Incus instance %q to stop: %w", name, err)
			}
		}
	}

	op, err := client.DeleteInstance(name)
	if err != nil {
		if isNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to delete Incus instance %q: %w", name, err)
	}
	if err := op.Wait(); err != nil {
		return fmt.Errorf("failed waiting for Incus instance %q deletion: %w", name, err)
	}

	return nil
}
