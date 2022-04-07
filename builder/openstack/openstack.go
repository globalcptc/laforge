package main

import (
	"context"
	"fmt"
	"net/http"

	"golang.org/x/sync/semaphore"

	"github.com/gen0cide/laforge/ent"
	"github.com/gen0cide/laforge/ent/network"
	"github.com/gen0cide/laforge/ent/provisionednetwork"
	"github.com/gen0cide/laforge/logging"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
)

const (
	ID          = "openstack"
	Name        = "Openstack"
	Description = "Builder that interfaces with Openstack"
	Author      = "Tenchi Mata <github.com/0xk7>"
	Version     = "0.1"
)

type openstackBuilder struct {
	HttpClient         http.Client
	IdentityEndpoint   http.Client
	Username           string
	Password           string
	TenantID           string
	Logger             *logging.Logger
	MaxWorkers         int
	DeployWorkerPool   *semaphore.Weighted
	TeardownWorkerPool *semaphore.Weighted
}

func (builder openstackBuilder) generateBuildID(build *ent.Build) string {
	buildId, err := build.ID.MarshalText()
	if err != nil {
		buildId = []byte(fmt.Sprint(build.Revision))
	}
	return fmt.Sprintf("%s", buildId)
}

func (builder openstackBuilder) generateVmName(competition *ent.Competition, team *ent.Team, host *ent.Host, build *ent.Build) string {
	return (competition.HclID + "-Team-" + fmt.Sprintf("%02d", team.TeamNumber) + "-" + host.Hostname + "-" + builder.generateBuildID(build))
}

func (builder openstackBuilder) generateRouterName(competition *ent.Competition, team *ent.Team, build *ent.Build) string {
	return (competition.HclID + "-Team-" + fmt.Sprintf("%02d", team.TeamNumber) + "-" + builder.generateBuildID(build))
}

func (builder openstackBuilder) generateNetworkName(competition *ent.Competition, team *ent.Team, network *ent.Network, build *ent.Build) string {
	return (competition.HclID + "-Team-" + fmt.Sprintf("%02d", team.TeamNumber) + "-" + network.Name + "-" + builder.generateBuildID(build))
}

func (builder openstackBuilder) DeployHost(ctx context.Context, provisionedHost *ent.ProvisionedHost) (err error) {
	host, err := provisionedHost.QueryProvisionedHostToHost().Only(ctx)
	if err != nil {
		return err
	}

	//add configuration here
	authOpts, err := openstack.AuthOptionsFromEnv()
	if err != nil {
		return err
	}
	provider, err := openstack.AuthenticatedClient(authOpts)
	if err != nil {
		return err
	}
	client, err := openstack.NewComputeV2(provider, gophercloud.EndpointOpts{
		Region: "RegionOne",
	})
	if err != nil {
		return err
	}

	build, err := provisionedHost.QueryProvisionedHostToPlan().QueryPlanToBuild().Only(ctx)
	if err != nil {
		return err
	}

	competition, err := build.QueryBuildToCompetition().Only(ctx)
	if err != nil {
		return err
	}

	network, err := provisionedHost.QueryProvisionedHostToProvisionedNetwork().QueryProvisionedNetworkToNetwork().Only(ctx)
	if err != nil {
		return err
	}

	team, err := provisionedHost.QueryProvisionedHostToProvisionedNetwork().QueryProvisionedNetworkToTeam().Only(ctx)
	if err != nil {
		return err
	}

	// generate vm name from ent
	vmName := builder.generateVmName(competition, team, host, build)
	networkName := builder.generateNetworkName(competition, team, network, build)

	err = builder.DeployWorkerPool.Acquire(ctx, int64(1))
	if err != nil {
		return
	}
	defer builder.DeployWorkerPool.Release(int64(1))

	//build server
	server, err := servers.Create(client, servers.CreateOpts{
		Name:      vmName,
		FlavorRef: "flavor_name",
		ImageRef:  "image_name",
		Networks:  networkName,
	}).Extract()
	if err != nil {
		fmt.Println("Unable to create server: %s", err)
	}
	fmt.Println("Server ID: %s", server.ID)

	id := server.ID
	newVars := host.Vars
	newVars["openstack_instance_id"] = id
	err = provisionedHost.Update().SetVars(newVars).Exec(ctx)
	if err != nil {
		return err
	}
	fmt.Println("Created tagged instance with ID " + id)

	return
}

func (builder openstackBuilder) DeployNetwork(ctx context.Context, provisionedNetwork *ent.ProvisionedNetwork) (err error) {
	entBuild, err := provisionedNetwork.QueryProvisionedNetworkToBuild().Only(ctx)
	if err != nil {
		return fmt.Errorf("couldn't query build from network \"%s\": %v", provisionedNetwork.Name, err)
	}
	entEnvironment, err := provisionedNetwork.QueryProvisionedNetworkToBuild().QueryBuildToEnvironment().Only(ctx)
	if err != nil {
		return fmt.Errorf("couldn't query build from network \"%s\": %v", provisionedNetwork.Name, err)
	}
	entCompetition, err := provisionedNetwork.QueryProvisionedNetworkToBuild().QueryBuildToEnvironment().QueryEnvironmentToCompetition().Only(ctx)
	if err != nil {
		return fmt.Errorf("couldn't query build from environment \"%s\": %v", entEnvironment.Name, err)
	}
	entNetwork, err := provisionedNetwork.QueryProvisionedNetworkToNetwork().Only(ctx)
	if err != nil {
		return fmt.Errorf("couldn't query build from network \"%s\": %v", provisionedNetwork.Name, err)
	}
	entTeam, err := provisionedNetwork.QueryProvisionedNetworkToTeam().Only(ctx)
	if err != nil {
		return fmt.Errorf("couldn't query build from network \"%s\": %v", provisionedNetwork.Name, err)
	}

	//add configuration here
	authOpts, err := openstack.AuthOptionsFromEnv()
	if err != nil {
		return err
	}
	provider, err := openstack.AuthenticatedClient(authOpts)
	if err != nil {
		return err
	}
	client, err := openstack.NewComputeV2(provider, gophercloud.EndpointOpts{
		Region: "RegionOne",
	})
	if err != nil {
		return err
	}

	tier1Name := builder.generateRouterName(entCompetition, entTeam, entBuild)
	up := true

	opts := networks.CreateOpts{Name: tier1Name, AdminStateUp: &up}

	// Execute the operation and get back a networks.Network struct
	results, err := networks.Create(client, opts).Extract()
	if err != nil {
		return fmt.Errorf("failed to create network: %v", err)
	}
	newVars := entNetwork.Vars
	newVars["openstack_network_id"] = results.ID
	err = provisionedNetwork.Update().SetVars(newVars).Exec(ctx)
	if err != nil {
		return err
	}
	fmt.Println(results)

	return
}

func (builder openstackBuilder) DeployTeam(ctx context.Context, entTeam *ent.Team) (err error) {
	entProNetwork, err := entTeam.QueryTeamToProvisionedNetwork().Where(
		provisionednetwork.HasProvisionedNetworkToNetworkWith(
			network.NameEQ("vdi"),
		),
	).First(ctx)

	if err != nil {
		return fmt.Errorf("failed to query vdi provisioned network from entTeam: %v", err)
	}
	err = builder.DeployNetwork(ctx, entProNetwork)
	if err != nil {
		return fmt.Errorf("failed to pre-create Tier-1 network: %v", err)
	}
	return
}

func (builder openstackBuilder) TeardownHost(ctx context.Context, provisionedHost *ent.ProvisionedHost) (err error) {
	host, err := provisionedHost.QueryProvisionedHostToHost().Only(ctx)
	if err != nil {
		return err
	}
	entBuild, err := provisionedHost.QueryProvisionedHostToProvisionedNetwork().QueryProvisionedNetworkToBuild().Only(ctx)
	if err != nil {
		return err
	}
	entCompetition, err := entBuild.QueryBuildToCompetition().Only(ctx)
	if err != nil {
		return fmt.Errorf("couldn't query competition from build \"%s\": %v", entBuild.ID, err)
	}
	entTeam, err := provisionedHost.QueryProvisionedHostToProvisionedNetwork().QueryProvisionedNetworkToTeam().Only(ctx)
	if err != nil {
		return fmt.Errorf("couldn't query build from network \"%s\": %v", provisionedHost.ID, err)
	}
	authOpts, err := openstack.AuthOptionsFromEnv()
	if err != nil {
		return err
	}
	provider, err := openstack.AuthenticatedClient(authOpts)
	if err != nil {
		return err
	}
	client, err := openstack.NewComputeV2(provider, gophercloud.EndpointOpts{
		Region: "RegionOne",
	})
	if err != nil {
		return err
	}

	// vmName := builder.generateVmName(entCompetition, entTeam, host, entBuild)

	err = builder.TeardownWorkerPool.Acquire(ctx, int64(1))
	if err != nil {
		return
	}
	defer builder.TeardownWorkerPool.Release(int64(1))

	// var instances []string
	instanceId := provisionedHost.Vars["openstack_instance_id"]
	result := servers.Delete(client, instanceId)
	fmt.Println(result)

	return
}

func (builder openstackBuilder) TeardownNetwork(ctx context.Context, provisionedNetwork *ent.ProvisionedNetwork) (err error) {
	entBuild, err := provisionedNetwork.QueryProvisionedNetworkToBuild().Only(ctx)
	if err != nil {
		return fmt.Errorf("couldn't query build from network \"%s\": %v", provisionedNetwork.Name, err)
	}
	entEnvironment, err := provisionedNetwork.QueryProvisionedNetworkToBuild().QueryBuildToEnvironment().Only(ctx)
	if err != nil {
		return fmt.Errorf("couldn't query build from network \"%s\": %v", provisionedNetwork.Name, err)
	}
	entCompetition, err := entEnvironment.EnvironmentToCompetition(ctx)
	if err != nil {
		return fmt.Errorf("couldn't query build from environment \"%s\": %v", entEnvironment.Name, err)
	}
	entNetwork, err := provisionedNetwork.QueryProvisionedNetworkToNetwork().Only(ctx)
	if err != nil {
		return fmt.Errorf("couldn't query build from network \"%s\": %v", provisionedNetwork.Name, err)
	}
	entTeam, err := provisionedNetwork.QueryProvisionedNetworkToTeam().Only(ctx)
	if err != nil {
		return fmt.Errorf("couldn't query build from network \"%s\": %v", provisionedNetwork.Name, err)
	}

	authOpts, err := openstack.AuthOptionsFromEnv()
	if err != nil {
		return err
	}
	provider, err := openstack.AuthenticatedClient(authOpts)
	if err != nil {
		return err
	}
	client, err := openstack.NewComputeV2(provider, gophercloud.EndpointOpts{
		Region: "RegionOne",
	})
	if err != nil {
		return err
	}

	// networkName := builder.generateNetworkName(entCompetition[0], entTeam, entNetwork, entBuild)

	err = builder.TeardownWorkerPool.Acquire(ctx, int64(1))
	if err != nil {
		return
	}
	defer builder.TeardownWorkerPool.Release(int64(1))

	networkId := provisionedNetwork.Vars["openstack_instance_id"]
	result := networks.Delete(client, networkId)
	fmt.Println(result)

	return
}

func (builder openstackBuilder) TeardownTeam(ctx context.Context, entTeam *ent.Team) (err error) {
	return
}
