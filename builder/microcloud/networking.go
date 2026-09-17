package microcloud

import (
	"context"
	"fmt"
	"net"
	"sort"
	"strings"

	"github.com/canonical/lxd/shared/api"
	"github.com/gen0cide/laforge/ent"
)

func networkGateway(cidr string) (string, error) {
	ip, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return "", fmt.Errorf("invalid network CIDR %q: %w", cidr, err)
	}

	ipv4 := ip.To4()
	if ipv4 == nil {
		return "", fmt.Errorf("MicroCloud builder only supports IPv4 networks, got %q", cidr)
	}

	ones, bits := ipNet.Mask.Size()
	if bits != 32 || ones > 24 {
		return "", fmt.Errorf(
			"network CIDR %q cannot use the required .254 gateway",
			cidr,
		)
	}

	gateway := append(net.IP(nil), ipv4...)
	gateway[3] = 254
	if !ipNet.Contains(gateway) {
		return "", fmt.Errorf("gateway %s is outside network CIDR %q", gateway, cidr)
	}

	return fmt.Sprintf("%s/%d", gateway.String(), ones), nil
}

func (builder *MicroCloudBuilder) DeployNetwork(
	ctx context.Context,
	provisionedNetwork *ent.ProvisionedNetwork,
) error {
	if err := builder.acquireDeployWorker(ctx); err != nil {
		return err
	}
	defer builder.DeployWorkerPool.Release(1)

	team, err := provisionedNetwork.QueryTeam().Only(ctx)
	if err != nil {
		return fmt.Errorf("failed to query team for provisioned network %q: %w", provisionedNetwork.Name, err)
	}
	build, err := provisionedNetwork.QueryBuild().Only(ctx)
	if err != nil {
		return fmt.Errorf("failed to query build for provisioned network %q: %w", provisionedNetwork.Name, err)
	}
	environment, err := build.QueryEnvironment().Only(ctx)
	if err != nil {
		return fmt.Errorf("failed to query environment for provisioned network %q: %w", provisionedNetwork.Name, err)
	}
	network, err := provisionedNetwork.QueryNetwork().Only(ctx)
	if err != nil {
		return fmt.Errorf("failed to query network for provisioned network %q: %w", provisionedNetwork.Name, err)
	}

	project := team.Vars["lxd_project"]
	if project == "" {
		return fmt.Errorf("team %d is missing lxd_project", team.TeamNumber)
	}
	client := builder.Client.UseProject(project)

	name := networkName(network, build)
	gateway, err := networkGateway(provisionedNetwork.Cidr)
	if err != nil {
		return err
	}
	description := fmt.Sprintf(
		"LaForge %s team %d network %s",
		environment.Name,
		team.TeamNumber,
		network.Name,
	)
	managedConfig := map[string]string{
		"network":      builder.Config.UplinkNetwork,
		"ipv4.address": gateway,
		"ipv4.dhcp":    "true",
		"ipv4.nat":     "true",
		"ipv6.address": "none",
	}

	existing, etag, err := client.GetNetwork(name)
	if err != nil && !isNotFound(err) {
		return fmt.Errorf("failed to get LXD network %q: %w", name, err)
	}
	if isNotFound(err) {
		err = client.CreateNetwork(api.NetworksPost{
			Name: name,
			Type: "ovn",
			NetworkPut: api.NetworkPut{
				Description: description,
				Config:      managedConfig,
			},
		})
		if err != nil {
			cleanupErr := client.DeleteNetwork(name)
			if cleanupErr != nil && !isNotFound(cleanupErr) {
				return fmt.Errorf(
					"failed to create LXD OVN network %q: %w (cleanup also failed: %v)",
					name,
					err,
					cleanupErr,
				)
			}
			return fmt.Errorf("failed to create LXD OVN network %q: %w", name, err)
		}
	} else {
		if existing.Type != "ovn" {
			return fmt.Errorf("existing LXD network %q has type %q, want ovn", name, existing.Type)
		}

		config := cloneVars(existing.Config)
		needsUpdate := existing.Description != description
		for key, value := range managedConfig {
			if config[key] != value {
				config[key] = value
				needsUpdate = true
			}
		}
		if needsUpdate {
			if err := client.UpdateNetwork(name, api.NetworkPut{
				Description: description,
				Config:      config,
			}, etag); err != nil {
				return fmt.Errorf("failed to reconcile LXD OVN network %q: %w", name, err)
			}
		}
	}

	vars := cloneVars(provisionedNetwork.Vars)
	vars["lxd_network"] = name
	if err := provisionedNetwork.Update().SetVars(vars).Exec(ctx); err != nil {
		return fmt.Errorf("failed to update vars for provisioned network %q: %w", provisionedNetwork.Name, err)
	}
	provisionedNetwork.Vars = vars

	lock := builder.teamLock(team)
	lock.Lock()
	defer lock.Unlock()

	team, err = refreshTeam(ctx, team)
	if err != nil {
		return err
	}
	if err := builder.peerTeamNetworks(ctx, client, team, name); err != nil {
		return err
	}
	if err := builder.updateTeamNetworks(ctx, team); err != nil {
		return err
	}

	return nil
}

func (builder *MicroCloudBuilder) peerTeamNetworks(
	ctx context.Context,
	client interface {
		GetNetworkPeer(string, string) (*api.NetworkPeer, string, error)
		CreateNetworkPeer(string, api.NetworkPeersPost) error
	},
	team *ent.Team,
	newNetwork string,
) error {
	provisionedNetworks, err := team.QueryProvisionedNetworks().All(ctx)
	if err != nil {
		return fmt.Errorf("failed to query networks for team %d: %w", team.TeamNumber, err)
	}

	project := team.Vars["lxd_project"]
	for _, provisionedNetwork := range provisionedNetworks {
		sibling := provisionedNetwork.Vars["lxd_network"]
		if sibling == "" || sibling == newNetwork {
			continue
		}

		if err := ensurePeer(client, project, newNetwork, sibling); err != nil {
			return err
		}
		if err := ensurePeer(client, project, sibling, newNetwork); err != nil {
			return err
		}

		peer, _, err := client.GetNetworkPeer(newNetwork, peerName(sibling))
		if err != nil {
			return fmt.Errorf("failed to verify LXD peering %q to %q: %w", newNetwork, sibling, err)
		}
		if !strings.EqualFold(peer.Status, "created") {
			return fmt.Errorf(
				"LXD peering %q to %q did not activate (status %q)",
				newNetwork,
				sibling,
				peer.Status,
			)
		}
	}

	return nil
}

func ensurePeer(
	client interface {
		GetNetworkPeer(string, string) (*api.NetworkPeer, string, error)
		CreateNetworkPeer(string, api.NetworkPeersPost) error
	},
	project string,
	sourceNetwork string,
	targetNetwork string,
) error {
	name := peerName(targetNetwork)
	peer, _, err := client.GetNetworkPeer(sourceNetwork, name)
	if err != nil && !isNotFound(err) {
		return fmt.Errorf("failed to get LXD peering %q on %q: %w", name, sourceNetwork, err)
	}
	if err == nil {
		if peer.TargetProject != project || peer.TargetNetwork != targetNetwork {
			return fmt.Errorf("LXD peering %q on %q targets the wrong network", name, sourceNetwork)
		}
		return nil
	}

	if err := client.CreateNetworkPeer(sourceNetwork, api.NetworkPeersPost{
		Name:          name,
		TargetProject: project,
		TargetNetwork: targetNetwork,
		NetworkPeerPut: api.NetworkPeerPut{
			Description: fmt.Sprintf("LaForge peering to %s", targetNetwork),
		},
	}); err != nil {
		return fmt.Errorf("failed to create LXD peering %q to %q: %w", sourceNetwork, targetNetwork, err)
	}
	return nil
}

func (builder *MicroCloudBuilder) updateTeamNetworks(ctx context.Context, team *ent.Team) error {
	team, err := refreshTeam(ctx, team)
	if err != nil {
		return err
	}

	provisionedNetworks, err := team.QueryProvisionedNetworks().All(ctx)
	if err != nil {
		return fmt.Errorf("failed to query networks for team %d: %w", team.TeamNumber, err)
	}

	names := make([]string, 0, len(provisionedNetworks))
	for _, provisionedNetwork := range provisionedNetworks {
		if name := provisionedNetwork.Vars["lxd_network"]; name != "" {
			names = append(names, name)
		}
	}
	sort.Strings(names)

	vars := cloneVars(team.Vars)
	vars["lxd_networks"] = strings.Join(names, ",")
	if err := team.Update().SetVars(vars).Exec(ctx); err != nil {
		return fmt.Errorf("failed to update network vars for team %d: %w", team.TeamNumber, err)
	}
	team.Vars = vars
	return nil
}

func (builder *MicroCloudBuilder) TeardownNetwork(
	ctx context.Context,
	provisionedNetwork *ent.ProvisionedNetwork,
) error {
	if err := builder.acquireTeardownWorker(ctx); err != nil {
		return err
	}
	defer builder.TeardownWorkerPool.Release(1)

	team, err := provisionedNetwork.QueryTeam().Only(ctx)
	if err != nil {
		return fmt.Errorf("failed to query team for provisioned network %q: %w", provisionedNetwork.Name, err)
	}
	project := team.Vars["lxd_project"]
	name := provisionedNetwork.Vars["lxd_network"]
	if project == "" || name == "" {
		build, err := provisionedNetwork.QueryBuild().Only(ctx)
		if err != nil {
			return fmt.Errorf("failed to query build for network %q teardown: %w", provisionedNetwork.Name, err)
		}
		environment, err := build.QueryEnvironment().Only(ctx)
		if err != nil {
			return fmt.Errorf("failed to query environment for network %q teardown: %w", provisionedNetwork.Name, err)
		}
		network, err := provisionedNetwork.QueryNetwork().Only(ctx)
		if err != nil {
			return fmt.Errorf("failed to query network %q for teardown: %w", provisionedNetwork.Name, err)
		}
		if project == "" {
			project = projectName(environment, team, build)
		}
		if name == "" {
			name = networkName(network, build)
		}
	}

	lock := builder.teamLock(team)
	lock.Lock()
	defer lock.Unlock()

	client := builder.Client.UseProject(project)
	forwards, err := client.GetNetworkForwards(name)
	if err != nil && !isNotFound(err) {
		return fmt.Errorf("failed to list forwards for LXD network %q: %w", name, err)
	}
	removedIngress := false
	for _, forward := range forwards {
		if forward.Config[forwardInstanceKey] == "" {
			continue
		}
		if err := client.DeleteNetworkForward(name, forward.ListenAddress); err != nil && !isNotFound(err) {
			return fmt.Errorf("failed to delete LXD forward %q on %q: %w", forward.ListenAddress, name, err)
		}
		removedIngress = true
	}

	peers, err := client.GetNetworkPeers(name)
	if err != nil && !isNotFound(err) {
		return fmt.Errorf("failed to list peerings for LXD network %q: %w", name, err)
	}
	for _, peer := range peers {
		if peer.TargetProject == project {
			err := client.DeleteNetworkPeer(peer.TargetNetwork, peerName(name))
			if err != nil && !isNotFound(err) {
				return fmt.Errorf(
					"failed to delete reverse LXD peering %q to %q: %w",
					peer.TargetNetwork,
					name,
					err,
				)
			}
		}

		err := client.DeleteNetworkPeer(name, peer.Name)
		if err != nil && !isNotFound(err) {
			return fmt.Errorf("failed to delete LXD peering %q on %q: %w", peer.Name, name, err)
		}
	}

	if err := client.DeleteNetwork(name); err != nil && !isNotFound(err) {
		return fmt.Errorf("failed to delete LXD network %q: %w", name, err)
	}

	vars := cloneVars(provisionedNetwork.Vars)
	delete(vars, "lxd_network")
	if err := provisionedNetwork.Update().SetVars(vars).Exec(ctx); err != nil {
		return fmt.Errorf("failed to clear vars for provisioned network %q: %w", provisionedNetwork.Name, err)
	}
	provisionedNetwork.Vars = vars

	if removedIngress {
		freshTeam, err := refreshTeam(ctx, team)
		if err != nil {
			return err
		}
		teamVars := cloneVars(freshTeam.Vars)
		delete(teamVars, "ingress_address")
		if err := freshTeam.Update().SetVars(teamVars).Exec(ctx); err != nil {
			return fmt.Errorf("failed to clear ingress address for team %d: %w", team.TeamNumber, err)
		}
		team = freshTeam
		team.Vars = teamVars
	}

	return builder.updateTeamNetworks(ctx, team)
}
