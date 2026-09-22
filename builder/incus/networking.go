package incus

import (
	"context"
	"fmt"

	"github.com/gen0cide/laforge/ent"
	incusclient "github.com/lxc/incus/v6/client"
	"github.com/lxc/incus/v6/shared/api"
)

func (builder *IncusBuilder) DeployNetwork(
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
	network, err := provisionedNetwork.QueryNetwork().Only(ctx)
	if err != nil {
		return fmt.Errorf("failed to query network for provisioned network %q: %w", provisionedNetwork.Name, err)
	}
	if _, err := networkGateway(provisionedNetwork.Cidr); err != nil {
		return err
	}

	client, _, project, err := builder.teamTarget(team)
	if err != nil {
		return err
	}

	bridge := networkName(network, team, build)
	description := fmt.Sprintf("LaForge team %d network %s", team.TeamNumber, network.Name)
	if err := ensureBridge(client, bridge, description); err != nil {
		return err
	}

	vars := cloneVars(provisionedNetwork.Vars)
	vars[varNetwork] = bridge
	if err := provisionedNetwork.Update().SetVars(vars).Exec(ctx); err != nil {
		return fmt.Errorf("failed to update vars for network %q: %w", provisionedNetwork.Name, err)
	}
	provisionedNetwork.Vars = vars

	if err := builder.reconcileGateway(ctx, team, reconcileExclusions{}); err != nil {
		return err
	}

	builder.Logger.Log.Infof(
		"Incus network %q for team %d attached to gateway in project %q",
		bridge,
		team.TeamNumber,
		project,
	)
	return nil
}

// ensureBridge creates an isolated bridge. It has no host address, DHCP or
// NAT, so identical team CIDRs never touch the host routing table; the team
// gateway provides routing, DHCP and NAT.
func ensureBridge(client incusclient.InstanceServer, name string, description string) error {
	desired := map[string]string{
		"ipv4.address": "none",
		"ipv6.address": "none",
	}

	existing, _, err := client.GetNetwork(name)
	if err == nil {
		for key, value := range desired {
			if existing.Config[key] != value {
				return fmt.Errorf(
					"existing Incus network %q config %q is %q, want %q",
					name,
					key,
					existing.Config[key],
					value,
				)
			}
		}
		return nil
	}
	if !isNotFound(err) {
		return fmt.Errorf("failed to get Incus network %q: %w", name, err)
	}

	if err := client.CreateNetwork(api.NetworksPost{
		Name: name,
		Type: "bridge",
		NetworkPut: api.NetworkPut{
			Description: description,
			Config:      desired,
		},
	}); err != nil {
		return fmt.Errorf("failed to create Incus network %q: %w", name, err)
	}

	return nil
}

func (builder *IncusBuilder) TeardownNetwork(
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
	bridge := provisionedNetwork.Vars[varNetwork]
	if bridge == "" {
		return nil
	}

	client, _, _, err := builder.teamTarget(team)
	if err != nil {
		return err
	}

	// Detach the bridge from the gateway before deleting it.
	if err := builder.reconcileGateway(ctx, team, reconcileExclusions{
		networks: map[string]struct{}{bridge: {}},
	}); err != nil {
		return err
	}
	if err := client.DeleteNetwork(bridge); err != nil && !isNotFound(err) {
		return fmt.Errorf("failed to delete Incus network %q: %w", bridge, err)
	}

	vars := cloneVars(provisionedNetwork.Vars)
	delete(vars, varNetwork)
	if err := provisionedNetwork.Update().SetVars(vars).Exec(ctx); err != nil {
		return fmt.Errorf("failed to clear vars for network %q: %w", provisionedNetwork.Name, err)
	}
	provisionedNetwork.Vars = vars

	return nil
}
