package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/x/sync/semaphore"

	"github.com/gen0cide/laforge/ent"
	"github.com/gen0cide/laforge/ent/network"
	"github.com/gen0cide/laforge/ent/provisionednetwork"
	"github.com/gen0cide/laforge/logging"

	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/pagination"
)

const (
	ID			= "openstack"
	Name		= "Openstack"
	Description	= "Builder that interfaces with Openstack"
	Author		= "Tenchi Mata <github.com/0xk7>"
	Version		= "0.1"
)

type openstackBuilder struct {
	HttpClient			http.Client
	IdentityEndpoint 	http.Client
	Username 			string
	Password 			string
	TenantID 			string
	Logger				*logging.Logger
	MaxWorkers			int
}

func (builder openstackBuilder) generateBuildID(build *ent.Build) string {
	buildId, err := build.ID.MarshalText()
	if err != nil {
		buildId = []byte(fmt.Sprint(build.Revision))
	}
	return fmt.Sprintf("%s", buildId)
}

func (builder openstackBuilder) generateVmName(competition *ent.Competition, team *ent.Team, host *ent.Host, build *ent.Build) string {
	return (competition.HcID + "-Team-" + fmt.Sprintf("%02d", team.TeamNumber) + "-" + host.Hostname + "-" + builder.generateBuildID(build))
}

func (builder openstackBuilder) DeployHOST(ctx context.Context, provisionedHost *ent.ProvisionedHost) (err error) {
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

	team, err := provisionedHost.QueryProvisionedHostToProvisionedNetwork().QueryProvisionedNetworkToTeam().Only(ctx)
	if err != nil {
		return err
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

func (builder openstackBuilder) DeployNetwork(ctx context.Context, provisionedNetwork *ent.ProvisionedNetwork) (err error) {
	entNetwork, err := provisionedNetwork.QuertProvisionedNetworkToNetwork().Only(ctx)
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

	opts := networks.CreateOpts{Name: entNetwork, AdminStateUp: networks.Up}

	// Execute the operation and get back a networks.Network struct
	results, err := networks.Create(client, opts).Extract()

	id := networks.Get(client, "id").Extract()
	newVars := entNetwork.Vars
	newVars[""] = id
	err = entNetwork.Update.SetVars(newVars).Exec(ctx)
	if err != nil {
		return err
	}
	fmt.Println(results)

	return
}

func (build openstackBuilder) TeardownHost(ctx context.Context, provisionedHost *ent.ProvisionedHost) (err error) {
	host, err := provisionedHost.QueryProvisionedHostToHost().Only(ctx)
	if err != nil {
		return err
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

	var instances []string
	instances[0] = host.InstanceId
	result, err := servers.Delete(client, instances[0])
	if err != nil {
		return err
	}
	fmt.Println(result)

	return
}

func (build openstackBuilder) TeardownNetwork(ctx context.Context, provisionedNetwork *ent.ProvisionedNetwork) (err error) {
	entNetwork, err := provisionedNetwork.QueryProvisionedNetworkToNetwork().Only(ctx)
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

	result, err := networks.Delete(client, "id")
	if err != nil {
		return err
	}
	fmt.Println(result)

	return
}