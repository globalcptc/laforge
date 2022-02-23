package main 

import (
	"context"
	"fmt"

	"github.com/gen0cide/laforge/ent"

	"github.com/rackspace/gophercloud/openstack"
	"github.com/rackspace/gophercloud/openstack/compute/v2/flavors"
	"github.com/rackspace/gophercloud/openstack/compute/v2/servers"
	"github.com/rackspace/gophercloud/pagination"
)

type OpenBuilder struct {
	IdentityEndpoint string
	Username string
	Password string
	TenantID string
}

func (builder OpenBuilder) generateBuildID(build *ent.Build) string {
	buildId, err := build.ID.MarshalText()
	if err != nil {
		buildId = []byte(fmt.Sprint(build.Revision))
	}
	return fmt.Sprintf("%s", buildId)
}

func (builder OpenBuilder) generateVmName(competition *ent.Competition, team *ent.Team, host *ent.Host, build *ent.Build) string {
	return (competition.HcID + "-Team-" + fmt.Sprintf("%02d", team.TeamNumber) + "-" + host.Hostname + "-" + builder.generateBuildID(build))
}

func (builder OpenBuilder) DeployHOST(ctx context.Context, provisionedHost *ent.ProvisionedHost) (err error) {
	host, err := provisionedHost.QueryProvisionedHostToHost().Only(ctx)
	if err != nil {
		return
	}

	//add configuration here
	authOpts, err := openstack.AuthOptionsFromEnv()
	if err != nil {
		return
	}
	provider, err := openstack.AuthenticatedClient(authOpts)
	if err != nil {
		return
	}
	client, err := openstack.NewComputeV2(provider, gophercloud.EndpointOpts{
		Region: "RegionOne",
	})
	if err != nil {
		return
	}

	build, err := provisionedHost.QueryProvisionedHostToPlan().QueryPlanToBuild().Only(ctx)
	if err != nil {
		return
	}

	competition, err := build.QueryBuildToCompetition().Only(ctx)
	if err != nil {
		return
	}

	team, err := provisionedHost.QueryProvisionedHostToProvisionedNetwork().QueryProvisionedNetworkToTeam().Only(ctx)
	if err != nil {
		return
	}

	// generate vm name from ent
	name := builder.generateVmName(competition, team, host, build)

	//build server
	server, err := servers.Create(client, servers.CreateOpts{
		Name: name,
		FlavorName: "flavor_name",
		ImageName: "image_name",
	}).Extract()
	if err != nil {
		fmt.Println("Unable to create server: %s", err)
	}
	fmt.Println("Server ID: %s", server.ID)

	id := *server.id
	newVars := host.Vars
	newVars["InstanceId"] = id
	err = host.Update().SetVars(newVars).Exec(ctx)
	if err != nil {
		return err
	}
	fmt.Println("Created tagged instance with ID " + id)

	return
}

func (builder OpenBuilder) DeployNetwork(ctx context.Context, provisionedNetwork *ent.ProvisionedNetwork) (err error) {
	
	return
}

func (build OpenBuilder) TeardownHost(ctx context.Context, provisionedHost *ent.ProvisionedHost) (err error) {
	return
}

func (build OpenBuilder) TeardownNetwork(ctx context.Context, provisionedNetwork *ent.ProvisionedNetwork) (err error) {
	return
}