package openstack

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

type OpenstackBuilder struct {
	Config             OpenstackBuilderConfig
	HttpClient         http.Client
	Logger             *logging.Logger
	DeployWorkerPool   *semaphore.Weighted
	TeardownWorkerPool *semaphore.Weighted
}

type OpenstackBuilderConfig struct {
	AuthUrl            string            `json:"auth_url"`
	Username           string            `json:"username"`
	Password           string            `json:"password"`
	ProjectID          string            `json:"project_id"`
	ProjectName        string            `json:"project_name"`
	RegionName         string            `json:"region_name"`
	DomainName         string            `json:"domain_name"`
	DomainId           string            `json:"domain_id"`
	MaxBuildWorkers    int               `json:"max_build_workers"`
	MaxTeardownWorkers int               `json:"max_teardown_workers"`
	Flavors            map[string]string `json:"flavors"`
	Images             map[string]string `json:"images"`
}

func (builder OpenstackBuilder) ID() string {
	return ID
}

func (builder OpenstackBuilder) Name() string {
	return Name
}

func (builder OpenstackBuilder) Description() string {
	return Description
}

func (builder OpenstackBuilder) Author() string {
	return Author
}

func (builder OpenstackBuilder) Version() string {
	return Version
}

func (builder OpenstackBuilder) generateBuildID(build *ent.Build) string {
	buildId, err := build.ID.MarshalText()
	if err != nil {
		buildId = []byte(fmt.Sprint(build.Revision))
	}
	return fmt.Sprintf("%s", buildId)
}

func (builder OpenstackBuilder) generateVmName(competition *ent.Competition, team *ent.Team, host *ent.Host, build *ent.Build) string {
	return (competition.HclID + "-Team-" + fmt.Sprintf("%02d", team.TeamNumber) + "-" + host.Hostname + "-" + builder.generateBuildID(build))
}

func (builder OpenstackBuilder) generateRouterName(competition *ent.Competition, team *ent.Team, build *ent.Build) string {
	return (competition.HclID + "-Team-" + fmt.Sprintf("%02d", team.TeamNumber) + "-" + builder.generateBuildID(build))
}

func (builder OpenstackBuilder) generateNetworkName(competition *ent.Competition, team *ent.Team, network *ent.Network, build *ent.Build) string {
	return (competition.HclID + "-Team-" + fmt.Sprintf("%02d", team.TeamNumber) + "-" + network.Name + "-" + builder.generateBuildID(build))
}

func (builder OpenstackBuilder) DeployHost(ctx context.Context, provisionedHost *ent.ProvisionedHost) (err error) {
	host, err := provisionedHost.QueryProvisionedHostToHost().Only(ctx)
	if err != nil {
		return err
	}

	authOpts := gophercloud.AuthOptions{
		IdentityEndpoint: builder.Config.AuthUrl,
		Username:         builder.Config.Username,
		Password:         builder.Config.Password,
		TenantID:         builder.Config.ProjectID,
		TenantName:       builder.Config.ProjectName,
	}
	if builder.Config.DomainId != "" {
		authOpts.DomainID = builder.Config.DomainId
	} else {
		authOpts.DomainName = builder.Config.DomainName
	}
	provider, err := openstack.AuthenticatedClient(authOpts)
	if err != nil {
		return fmt.Errorf("failed to authenticate: %v", err)
	}
	client, err := openstack.NewComputeV2(provider, gophercloud.EndpointOpts{
		Type:   "compute",
		Region: builder.Config.RegionName,
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

	builder.Logger.Log.Debugf("Deploying host with image \"%s\" and flavor \"%s\"", builder.Config.Images[host.OS], builder.Config.Flavors[host.InstanceSize])
	//build server
	server, err := servers.Create(client, servers.CreateOpts{
		Name:      vmName,
		FlavorRef: builder.Config.Flavors[host.InstanceSize],
		ImageRef:  builder.Config.Images[host.OS],
		Networks:  networkName,
	}).Extract()
	if err != nil {
		builder.Logger.Log.Errorf("Unable to create server: %v", err)
		return
	}
	builder.Logger.Log.Debugf("Server ID: %s", server.ID)

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

func (builder OpenstackBuilder) DeployNetwork(ctx context.Context, provisionedNetwork *ent.ProvisionedNetwork) (err error) {
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

func (builder OpenstackBuilder) DeployTeam(ctx context.Context, entTeam *ent.Team) (err error) {
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

func (builder OpenstackBuilder) TeardownHost(ctx context.Context, provisionedHost *ent.ProvisionedHost) (err error) {
	// host, err := provisionedHost.QueryProvisionedHostToHost().Only(ctx)
	// if err != nil {
	// 	return err
	// }
	// entBuild, err := provisionedHost.QueryProvisionedHostToProvisionedNetwork().QueryProvisionedNetworkToBuild().Only(ctx)
	// if err != nil {
	// 	return err
	// }
	// entCompetition, err := entBuild.QueryBuildToCompetition().Only(ctx)
	// if err != nil {
	// 	return fmt.Errorf("couldn't query competition from build \"%s\": %v", entBuild.ID, err)
	// }
	// entTeam, err := provisionedHost.QueryProvisionedHostToProvisionedNetwork().QueryProvisionedNetworkToTeam().Only(ctx)
	// if err != nil {
	// 	return fmt.Errorf("couldn't query build from network \"%s\": %v", provisionedHost.ID, err)
	// }
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

func (builder OpenstackBuilder) TeardownNetwork(ctx context.Context, provisionedNetwork *ent.ProvisionedNetwork) (err error) {
	// entBuild, err := provisionedNetwork.QueryProvisionedNetworkToBuild().Only(ctx)
	// if err != nil {
	// 	return fmt.Errorf("couldn't query build from network \"%s\": %v", provisionedNetwork.Name, err)
	// }
	// entEnvironment, err := provisionedNetwork.QueryProvisionedNetworkToBuild().QueryBuildToEnvironment().Only(ctx)
	// if err != nil {
	// 	return fmt.Errorf("couldn't query build from network \"%s\": %v", provisionedNetwork.Name, err)
	// }
	// entCompetition, err := entEnvironment.EnvironmentToCompetition(ctx)
	// if err != nil {
	// 	return fmt.Errorf("couldn't query build from environment \"%s\": %v", entEnvironment.Name, err)
	// }
	// entNetwork, err := provisionedNetwork.QueryProvisionedNetworkToNetwork().Only(ctx)
	// if err != nil {
	// 	return fmt.Errorf("couldn't query build from network \"%s\": %v", provisionedNetwork.Name, err)
	// }
	// entTeam, err := provisionedNetwork.QueryProvisionedNetworkToTeam().Only(ctx)
	// if err != nil {
	// 	return fmt.Errorf("couldn't query build from network \"%s\": %v", provisionedNetwork.Name, err)
	// }

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

func (builder OpenstackBuilder) TeardownTeam(ctx context.Context, entTeam *ent.Team) (err error) {
	return
}
