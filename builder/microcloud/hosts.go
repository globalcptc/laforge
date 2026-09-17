package microcloud

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/canonical/lxd/shared/api"
	"github.com/gen0cide/laforge/ent"
)

const forwardInstanceKey = "user.laforge.instance"

func (builder *MicroCloudBuilder) DeployHost(
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
	if isWindowsOS(host.OS) {
		return fmt.Errorf(
			"MicroCloud Windows image %q is not supported until its Cloudbase-Init and LXD agent contract is validated",
			host.OS,
		)
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

	project := team.Vars["lxd_project"]
	lxdNetwork := provisionedNetwork.Vars["lxd_network"]
	if project == "" {
		return fmt.Errorf("team %d is missing lxd_project", team.TeamNumber)
	}
	if lxdNetwork == "" {
		return fmt.Errorf("network %q is missing lxd_network", provisionedNetwork.Name)
	}

	size, ok := builder.Config.InstanceSizes[host.InstanceSize]
	if !ok {
		return fmt.Errorf("no MicroCloud instance size configured for %q", host.InstanceSize)
	}
	image, ok := builder.Config.Images[host.OS]
	if !ok {
		return fmt.Errorf("no MicroCloud image configured for OS %q", host.OS)
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

	client := builder.Client.UseProject(project)
	name := instanceName(host, build)
	instanceDescription := fmt.Sprintf(
		"LaForge team %d host %s",
		team.TeamNumber,
		host.Hostname,
	)
	instancePut := api.InstancePut{
		Description: instanceDescription,
		Config: map[string]string{
			"limits.cpu":           size.CPU,
			"limits.memory":        size.Memory,
			"cloud-init.user-data": userData(host.OS, agentURL, password),
		},
		Devices: map[string]map[string]string{
			"eth0": {
				"type":         "nic",
				"network":      lxdNetwork,
				"ipv4.address": provisionedHost.SubnetIP,
			},
			"root": {
				"type": "disk",
				"pool": builder.Config.StoragePool,
				"path": "/",
				"size": fmt.Sprintf("%dGiB", disk.Size),
			},
		},
		Profiles: []string{},
	}

	instance, _, err := client.GetInstance(name)
	if err != nil && !isNotFound(err) {
		return fmt.Errorf("failed to get LXD instance %q: %w", name, err)
	}
	if isNotFound(err) {
		op, err := client.CreateInstance(api.InstancesPost{
			Name:  name,
			Type:  api.InstanceTypeVM,
			Start: true,
			Source: api.InstanceSource{
				Type:  api.SourceTypeImage,
				Alias: image,
			},
			InstancePut: instancePut,
		})
		if err != nil {
			return fmt.Errorf("failed to create LXD instance %q: %w", name, err)
		}
		if err := op.Wait(); err != nil {
			return fmt.Errorf("failed waiting for LXD instance %q: %w", name, err)
		}
	} else {
		if err := validateExistingInstance(instance, instancePut, team.Vars["lxd_cluster_member"]); err != nil {
			return fmt.Errorf("existing LXD instance %q is not reusable: %w", name, err)
		}
		if instance.StatusCode == api.Stopped {
			op, err := client.UpdateInstanceState(name, api.InstanceStatePut{
				Action:  "start",
				Timeout: -1,
			}, "")
			if err != nil {
				return fmt.Errorf("failed to start existing LXD instance %q: %w", name, err)
			}
			if err := op.Wait(); err != nil {
				return fmt.Errorf("failed waiting for LXD instance %q to start: %w", name, err)
			}
		}
	}

	vars := cloneVars(provisionedHost.Vars)
	vars["lxd_instance"] = name
	if host.Vars["public_ingress"] == "true" {
		listenAddress, err := builder.ensureIngressForward(
			ctx,
			client,
			team,
			lxdNetwork,
			name,
			provisionedHost.SubnetIP,
			host.ExposedTCPPorts,
			host.ExposedUDPPorts,
		)
		if err != nil {
			return err
		}
		vars["lxd_forward_address"] = listenAddress
		vars["PublicIP"] = listenAddress
	}

	if err := provisionedHost.Update().SetVars(vars).Exec(ctx); err != nil {
		return fmt.Errorf("failed to update vars for LXD instance %q: %w", name, err)
	}
	provisionedHost.Vars = vars

	return nil
}

func validateExistingInstance(
	instance *api.Instance,
	desired api.InstancePut,
	expectedMember string,
) error {
	if instance.StatusCode == api.Error || strings.EqualFold(instance.Status, "error") {
		return fmt.Errorf("instance is in error state")
	}
	if instance.Type != string(api.InstanceTypeVM) {
		return fmt.Errorf("type is %q, want %q", instance.Type, api.InstanceTypeVM)
	}
	if expectedMember != "" && instance.Location != expectedMember {
		return fmt.Errorf("location is %q, want %q", instance.Location, expectedMember)
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

func (builder *MicroCloudBuilder) ensureIngressForward(
	ctx context.Context,
	client interface {
		GetNetworkForwards(string) ([]api.NetworkForward, error)
		GetNetworkForward(string, string) (*api.NetworkForward, string, error)
		CreateNetworkForward(string, api.NetworkForwardsPost) error
		UpdateNetworkForward(string, string, api.NetworkForwardPut, string) error
	},
	team *ent.Team,
	networkName string,
	instanceName string,
	targetAddress string,
	tcpPorts []string,
	udpPorts []string,
) (string, error) {
	lock := builder.teamLock(team)
	lock.Lock()
	defer lock.Unlock()

	team, err := refreshTeam(ctx, team)
	if err != nil {
		return "", err
	}
	ports := desiredForwardPorts(targetAddress, tcpPorts, udpPorts)
	if len(ports) == 0 {
		return "", fmt.Errorf("public ingress host %q has no exposed ports", instanceName)
	}

	forwards, err := client.GetNetworkForwards(networkName)
	if err != nil {
		return "", fmt.Errorf("failed to list LXD forwards for network %q: %w", networkName, err)
	}
	for _, forward := range forwards {
		if forward.Config[forwardInstanceKey] == instanceName {
			if err := reconcileIngressForward(
				client,
				networkName,
				forward.ListenAddress,
				instanceName,
				ports,
			); err != nil {
				return "", err
			}
			if err := setTeamIngressAddress(ctx, team, forward.ListenAddress); err != nil {
				return "", err
			}
			return forward.ListenAddress, nil
		}
		if forward.Config[forwardInstanceKey] != "" {
			return "", fmt.Errorf(
				"team %d network %q already has LaForge ingress for %q",
				team.TeamNumber,
				networkName,
				forward.Config[forwardInstanceKey],
			)
		}
	}

	provisionedNetworks, err := team.QueryProvisionedNetworks().All(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to query networks for team %d: %w", team.TeamNumber, err)
	}
	for _, provisionedNetwork := range provisionedNetworks {
		siblingNetwork := provisionedNetwork.Vars["lxd_network"]
		if siblingNetwork == "" || siblingNetwork == networkName {
			continue
		}
		siblingForwards, err := client.GetNetworkForwards(siblingNetwork)
		if err != nil {
			return "", fmt.Errorf("failed to list LXD forwards for network %q: %w", siblingNetwork, err)
		}
		for _, forward := range siblingForwards {
			if owner := forward.Config[forwardInstanceKey]; owner != "" {
				return "", fmt.Errorf(
					"team %d already has LaForge ingress for %q on network %q",
					team.TeamNumber,
					owner,
					siblingNetwork,
				)
			}
		}
	}

	if current := team.Vars["ingress_address"]; current != "" {
		return "", fmt.Errorf(
			"team %d already has ingress address %q",
			team.TeamNumber,
			current,
		)
	}

	if err := client.CreateNetworkForward(networkName, api.NetworkForwardsPost{
		ListenAddress: "0.0.0.0",
		NetworkForwardPut: api.NetworkForwardPut{
			Description: fmt.Sprintf("LaForge ingress for %s", instanceName),
			Config: map[string]string{
				forwardInstanceKey: instanceName,
			},
			Ports: ports,
		},
	}); err != nil {
		return "", fmt.Errorf("failed to create LXD ingress forward for %q: %w", instanceName, err)
	}

	forwards, err = client.GetNetworkForwards(networkName)
	if err != nil {
		return "", fmt.Errorf("failed to read allocated LXD ingress address for %q: %w", instanceName, err)
	}
	for _, forward := range forwards {
		if forward.Config[forwardInstanceKey] != instanceName {
			continue
		}

		if err := setTeamIngressAddress(ctx, team, forward.ListenAddress); err != nil {
			return "", err
		}
		return forward.ListenAddress, nil
	}

	return "", fmt.Errorf("LXD did not return the allocated ingress forward for %q", instanceName)
}

func desiredForwardPorts(
	targetAddress string,
	tcpPorts []string,
	udpPorts []string,
) []api.NetworkForwardPort {
	ports := make([]api.NetworkForwardPort, 0, 2)
	if len(tcpPorts) > 0 {
		ports = append(ports, api.NetworkForwardPort{
			Description:   "LaForge team ingress TCP",
			Protocol:      "tcp",
			ListenPort:    strings.Join(tcpPorts, ","),
			TargetPort:    strings.Join(tcpPorts, ","),
			TargetAddress: targetAddress,
		})
	}
	if len(udpPorts) > 0 {
		ports = append(ports, api.NetworkForwardPort{
			Description:   "LaForge team ingress UDP",
			Protocol:      "udp",
			ListenPort:    strings.Join(udpPorts, ","),
			TargetPort:    strings.Join(udpPorts, ","),
			TargetAddress: targetAddress,
		})
	}
	return ports
}

func reconcileIngressForward(
	client interface {
		GetNetworkForward(string, string) (*api.NetworkForward, string, error)
		UpdateNetworkForward(string, string, api.NetworkForwardPut, string) error
	},
	networkName string,
	listenAddress string,
	instanceName string,
	ports []api.NetworkForwardPort,
) error {
	forward, etag, err := client.GetNetworkForward(networkName, listenAddress)
	if err != nil {
		return fmt.Errorf("failed to get LXD ingress forward %q: %w", listenAddress, err)
	}

	config := cloneVars(forward.Config)
	config[forwardInstanceKey] = instanceName
	delete(config, "target_address")
	desired := api.NetworkForwardPut{
		Description: fmt.Sprintf("LaForge ingress for %s", instanceName),
		Config:      config,
		Ports:       ports,
	}
	current := forward.Writable()
	current.Normalise()
	desired.Normalise()
	if reflect.DeepEqual(current, desired) {
		return nil
	}

	if err := client.UpdateNetworkForward(networkName, listenAddress, desired, etag); err != nil {
		return fmt.Errorf("failed to reconcile LXD ingress forward %q: %w", listenAddress, err)
	}
	return nil
}

func setTeamIngressAddress(ctx context.Context, team *ent.Team, address string) error {
	fresh, err := refreshTeam(ctx, team)
	if err != nil {
		return err
	}
	team = fresh

	vars := cloneVars(team.Vars)
	if current := vars["ingress_address"]; current != "" && current != address {
		return fmt.Errorf("team %d already has ingress address %q", team.TeamNumber, current)
	}
	vars["ingress_address"] = address
	if err := team.Update().SetVars(vars).Exec(ctx); err != nil {
		return fmt.Errorf("failed to update ingress address for team %d: %w", team.TeamNumber, err)
	}
	team.Vars = vars
	return nil
}

func (builder *MicroCloudBuilder) TeardownHost(
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

	project := team.Vars["lxd_project"]
	lxdNetwork := provisionedNetwork.Vars["lxd_network"]
	name := provisionedHost.Vars["lxd_instance"]
	if project == "" || lxdNetwork == "" || name == "" {
		build, err := provisionedHost.QueryBuild().Only(ctx)
		if err != nil {
			return fmt.Errorf("failed to query build for host teardown: %w", err)
		}
		environment, err := build.QueryEnvironment().Only(ctx)
		if err != nil {
			return fmt.Errorf("failed to query environment for host teardown: %w", err)
		}
		host, err := provisionedHost.QueryHost().Only(ctx)
		if err != nil {
			return fmt.Errorf("failed to query host for teardown: %w", err)
		}
		network, err := provisionedNetwork.QueryNetwork().Only(ctx)
		if err != nil {
			return fmt.Errorf("failed to query network for host teardown: %w", err)
		}
		if project == "" {
			project = projectName(environment, team, build)
		}
		if lxdNetwork == "" {
			lxdNetwork = networkName(network, build)
		}
		if name == "" {
			name = instanceName(host, build)
		}
	}
	client := builder.Client.UseProject(project)

	forwardAddress := provisionedHost.Vars["lxd_forward_address"]
	if forwardAddress == "" {
		forwards, err := client.GetNetworkForwards(lxdNetwork)
		if err != nil && !isNotFound(err) {
			return fmt.Errorf("failed to list LXD forwards for network %q: %w", lxdNetwork, err)
		}
		for _, forward := range forwards {
			if forward.Config[forwardInstanceKey] == name {
				forwardAddress = forward.ListenAddress
				break
			}
		}
	}
	if forwardAddress != "" && lxdNetwork != "" {
		if err := client.DeleteNetworkForward(lxdNetwork, forwardAddress); err != nil && !isNotFound(err) {
			return fmt.Errorf("failed to delete LXD forward %q: %w", forwardAddress, err)
		}

		lock := builder.teamLock(team)
		lock.Lock()
		freshTeam, refreshErr := refreshTeam(ctx, team)
		if refreshErr != nil {
			lock.Unlock()
			return refreshErr
		}
		vars := cloneVars(freshTeam.Vars)
		delete(vars, "ingress_address")
		err := freshTeam.Update().SetVars(vars).Exec(ctx)
		if err == nil {
			team.Vars = vars
		}
		lock.Unlock()
		if err != nil {
			return fmt.Errorf("failed to clear ingress address for team %d: %w", team.TeamNumber, err)
		}
	}

	if name != "" {
		instance, _, err := client.GetInstance(name)
		if err != nil && !isNotFound(err) {
			return fmt.Errorf("failed to get LXD instance %q for deletion: %w", name, err)
		}
		if err == nil {
			if instance.IsActive() {
				op, err := client.UpdateInstanceState(name, api.InstanceStatePut{
					Action:  "stop",
					Timeout: -1,
					Force:   true,
				}, "")
				if err != nil {
					return fmt.Errorf("failed to stop LXD instance %q: %w", name, err)
				}
				if err := op.Wait(); err != nil {
					return fmt.Errorf("failed waiting for LXD instance %q to stop: %w", name, err)
				}
			}

			op, err := client.DeleteInstance(name)
			if err != nil {
				return fmt.Errorf("failed to delete LXD instance %q: %w", name, err)
			}
			if err := op.Wait(); err != nil {
				return fmt.Errorf("failed waiting for LXD instance %q deletion: %w", name, err)
			}
		}
	}

	vars := cloneVars(provisionedHost.Vars)
	delete(vars, "lxd_instance")
	delete(vars, "lxd_forward_address")
	delete(vars, "PublicIP")
	if err := provisionedHost.Update().SetVars(vars).Exec(ctx); err != nil {
		return fmt.Errorf("failed to clear vars for provisioned host %s: %w", provisionedHost.ID, err)
	}

	return nil
}
