package openstack

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"

	"golang.org/x/sync/semaphore"

	"github.com/gen0cide/laforge/ent"
	"github.com/gen0cide/laforge/logging"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/secgroups"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/routers"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
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
	ServerUrl          string            `json:"laforge_server_url"`
	IdentityVersion    string            `json:"identify_version"`
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
	ExternalNetworkId  string            `json:"external_network_id"`
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
	return string(buildId)
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

func (builder OpenstackBuilder) newAuthProvider() (*gophercloud.ProviderClient, error) {
	u, err := url.Parse(builder.Config.AuthUrl)
	if err != nil {
		return nil, fmt.Errorf("unable to parse auth_url \"%s\" from builder config", builder.Config.AuthUrl)
	}
	u.Path = path.Join(u.Path, builder.Config.IdentityVersion)

	authOpts := gophercloud.AuthOptions{
		IdentityEndpoint: u.String(),
		Username:         builder.Config.Username,
		Password:         builder.Config.Password,
		TenantID:         builder.Config.ProjectID,
		TenantName:       builder.Config.ProjectName,
	}
	if builder.Config.DomainName != "" {
		authOpts.DomainName = builder.Config.DomainName
	} else {
		authOpts.DomainID = builder.Config.DomainId
	}
	return openstack.AuthenticatedClient(authOpts)
}

func (builder OpenstackBuilder) DeployHost(ctx context.Context, entProvisionedHost *ent.ProvisionedHost) (err error) {
	entHost, err := entProvisionedHost.QueryProvisionedHostToHost().Only(ctx)
	if err != nil {
		return fmt.Errorf("failed querying host from provisioned host: %v", err)
	}
	entProvisionedNetwork, err := entProvisionedHost.QueryProvisionedHostToProvisionedNetwork().Only(ctx)
	if err != nil {
		return fmt.Errorf("failed to query provisioned network from provisioned host: %v", err)
	}
	entBuild, err := entProvisionedHost.QueryProvisionedHostToPlan().QueryPlanToBuild().Only(ctx)
	if err != nil {
		return fmt.Errorf("failed querying build from provisioned host: %v", err)
	}
	entCompetition, err := entBuild.QueryBuildToCompetition().Only(ctx)
	if err != nil {
		return fmt.Errorf("failed querying competition from provisioned host: %v", err)
	}
	entTeam, err := entProvisionedHost.QueryProvisionedHostToProvisionedNetwork().QueryProvisionedNetworkToTeam().Only(ctx)
	if err != nil {
		return fmt.Errorf("failed querying team from provisioned host: %v", err)
	}

	// ###################
	// Wait on open thread
	// ###################
	err = builder.DeployWorkerPool.Acquire(ctx, int64(1))
	if err != nil {
		return fmt.Errorf("failed to acquire thread: %v", err)
	}
	defer builder.DeployWorkerPool.Release(int64(1))

	// #############################
	// Generate authenticated client
	// #############################
	provider, err := builder.newAuthProvider()
	if err != nil {
		return fmt.Errorf("failed to authenticate: %v", err)
	}
	endpointOpts := gophercloud.EndpointOpts{
		Region: builder.Config.RegionName,
	}
	computeClient, err := openstack.NewComputeV2(provider, endpointOpts)
	if err != nil {
		return fmt.Errorf("failed to create compute v2 client: %v", err)
	}

	// ###########
	// Deploy Host
	// ###########
	vmName := builder.generateVmName(entCompetition, entTeam, entHost, entBuild)

	// Generate host IP address
	networkAddrParts := strings.Split(entProvisionedNetwork.Cidr, "/")
	networkAddr := networkAddrParts[0]
	networkOctetStrings := strings.Split(networkAddr, ".")
	networkOctets := []byte{0, 0, 0, 0}
	for i, octetString := range networkOctetStrings {
		octet, err := strconv.Atoi(octetString)
		if err != nil {
			return fmt.Errorf("error while parsing IPv4 Address %s: %v", entProvisionedNetwork.Cidr, err)
		}
		networkOctets[i] = byte(octet)
	}
	_, ipv4Net, err := net.ParseCIDR(entProvisionedNetwork.Cidr)
	if err != nil {
		return fmt.Errorf("error while parsing cidr: %v", err)
	}
	if len(ipv4Net.Mask) != 4 {
		return fmt.Errorf("mask is not correct length")
	}
	hostAddress := strings.Join(append(networkOctetStrings[:3], fmt.Sprint(entHost.LastOctet)), ".")

	// Create Security Group for host
	builder.Logger.Log.Debugf("Creating security group %s", vmName)
	opts := secgroups.CreateOpts{
		Name:        vmName,
		Description: fmt.Sprintf("Sec group for VM %s", vmName),
	}
	osSecGroup, err := secgroups.Create(computeClient, opts).Extract()
	if err != nil {
		return fmt.Errorf("failed to create security group: %v", err)
	}

	newVars := entHost.Vars
	newVars["openstack_secgroup_id"] = osSecGroup.ID
	err = entProvisionedHost.Update().SetVars(newVars).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update provisioned host vars: %v", err)
	}
	ruleOpts := make([]secgroups.CreateRuleOpts, 0)
	for _, port := range entHost.ExposedTCPPorts {
		destPort, err := strconv.Atoi(port)
		if err != nil {
			return fmt.Errorf("could not convert TCP port \"%s\" to integer: %v", port, err)
		}
		opts := secgroups.CreateRuleOpts{
			ParentGroupID: osSecGroup.ID,
			FromPort:      destPort,
			ToPort:        destPort,
			IPProtocol:    "TCP",
			CIDR:          "0.0.0.0/0",
		}
		ruleOpts = append(ruleOpts, opts)
	}
	for _, port := range entHost.ExposedUDPPorts {
		destPort, err := strconv.Atoi(port)
		if err != nil {
			return fmt.Errorf("could not convert UDP port \"%s\" to integer: %v", port, err)
		}
		opts := secgroups.CreateRuleOpts{
			ParentGroupID: osSecGroup.ID,
			FromPort:      destPort,
			ToPort:        destPort,
			IPProtocol:    "UDP",
			CIDR:          "0.0.0.0/0",
		}
		ruleOpts = append(ruleOpts, opts)
	}
	for _, opts := range ruleOpts {
		_, err = secgroups.CreateRule(computeClient, opts).Extract()
		if err != nil {
			return fmt.Errorf("failed to create security group %s rule to port %d", opts.IPProtocol, opts.FromPort)
		}
	}

	// Generate host configuration
	var adminPassword string
	if len(entHost.OverridePassword) > 0 {
		adminPassword = entHost.OverridePassword
	} else {
		adminPassword = entCompetition.RootPassword
	}
	agentFile, err := entProvisionedHost.QueryProvisionedHostToGinFileMiddleware().First(ctx)
	if err != nil {
		return fmt.Errorf("failed to query gin file middleware from provisioned host: %v", err)
	}
	agentUrl := fmt.Sprintf("%s/api/download/%s", builder.Config.ServerUrl, agentFile.URLID)

	var userData string
	if strings.HasPrefix(entHost.OS, "w2k") {
		userData = fmt.Sprintf(`<script>
powershell -Command mkdir $env:PROGRAMDATA\Laforge -Force
powershell -Command do{	$test = Test-Connection 1.1.1.1 -Quiet; Start-Sleep -s 5}until($test)
powershell -Command Invoke-WebRequest %s -OutFile $env:PROGRAMDATA\Laforge\laforge.exe
powershell -Command %%PROGRAMDATA%%\Laforge\laforge.exe -service install
powershell -Command %%PROGRAMDATA%%\Laforge\laforge.exe -service start
powershell -Command logoff
</script>`, agentUrl)
	} else {
		userData = fmt.Sprintf(`#!/bin/bash
while [ ! -f "/laforge.bin" ]
do
curl -sL -o /laforge.bin %s
sleep 10
done
chmod +x /laforge.bin
cd /
./laforge.bin -service install
./laforge.bin -service start
`, agentUrl)
	}

	// Create the host
	builder.Logger.Log.Debugf("Deploying host with image \"%s\" and flavor \"%s\"", builder.Config.Images[entHost.OS], builder.Config.Flavors[entHost.InstanceSize])
	server, err := servers.Create(computeClient, servers.CreateOpts{
		Name:           vmName,
		ImageRef:       builder.Config.Images[entHost.OS],
		FlavorRef:      builder.Config.Flavors[entHost.InstanceSize],
		SecurityGroups: []string{osSecGroup.ID},
		UserData:       []byte(userData),
		Networks: []servers.Network{{
			UUID:    entProvisionedNetwork.Vars["openstack_network_id"],
			FixedIP: hostAddress,
		}},
		AdminPass:  adminPassword,
		AccessIPv4: hostAddress,
	}).Extract()
	if err != nil {
		builder.Logger.Log.Errorf("Unable to create server: %v", err)
		return
	}
	builder.Logger.Log.WithField("root_password", server.AdminPass).Debugf("Server ID: %s", server.ID)

	// Store Openstack instance ID in provisioned host vars
	newVars["openstack_instance_id"] = server.ID
	err = entProvisionedHost.Update().SetVars(newVars).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update provisioned host vars: %v", err)
	}
	return
}

func (builder OpenstackBuilder) DeployNetwork(ctx context.Context, entProvisionedNetwork *ent.ProvisionedNetwork) (err error) {
	entBuild, err := entProvisionedNetwork.QueryProvisionedNetworkToBuild().Only(ctx)
	if err != nil {
		return fmt.Errorf("couldn't query build from network \"%s\": %v", entProvisionedNetwork.Name, err)
	}
	entEnvironment, err := entProvisionedNetwork.QueryProvisionedNetworkToBuild().QueryBuildToEnvironment().Only(ctx)
	if err != nil {
		return fmt.Errorf("couldn't query build from network \"%s\": %v", entProvisionedNetwork.Name, err)
	}
	entCompetition, err := entProvisionedNetwork.QueryProvisionedNetworkToBuild().QueryBuildToEnvironment().QueryEnvironmentToCompetition().Only(ctx)
	if err != nil {
		return fmt.Errorf("couldn't query build from environment \"%s\": %v", entEnvironment.Name, err)
	}
	entNetwork, err := entProvisionedNetwork.QueryProvisionedNetworkToNetwork().Only(ctx)
	if err != nil {
		return fmt.Errorf("couldn't query build from network \"%s\": %v", entProvisionedNetwork.Name, err)
	}
	entTeam, err := entProvisionedNetwork.QueryProvisionedNetworkToTeam().Only(ctx)
	if err != nil {
		return fmt.Errorf("couldn't query build from network \"%s\": %v", entProvisionedNetwork.Name, err)
	}

	// ###################
	// Wait on open thread
	// ###################
	err = builder.DeployWorkerPool.Acquire(ctx, int64(1))
	if err != nil {
		return fmt.Errorf("failed to acquire thread: %v", err)
	}
	defer builder.DeployWorkerPool.Release(int64(1))

	// #############################
	// Generate authenticated client
	// #############################
	provider, err := builder.newAuthProvider()
	if err != nil {
		return fmt.Errorf("failed to authenticate: %v", err)
	}
	endpointOpts := gophercloud.EndpointOpts{
		Name:   "neutron",
		Region: builder.Config.RegionName,
	}
	networkClient, err := openstack.NewNetworkV2(provider, endpointOpts)
	if err != nil {
		return fmt.Errorf("failed to create openstack network client: %v", err)
	}

	// ##############
	// Deploy network
	// ##############
	// Create the network
	osNetwork, err := networks.Create(networkClient, networks.CreateOpts{
		Name:         builder.generateNetworkName(entCompetition, entTeam, entNetwork, entBuild),
		AdminStateUp: gophercloud.Enabled,
	}).Extract()
	if err != nil {
		return fmt.Errorf("failed to create network: %v", err)
	}

	// Store the Openstack network ID in provisioned network vars
	newVars := entProvisionedNetwork.Vars
	newVars["openstack_network_id"] = osNetwork.ID
	err = entProvisionedNetwork.Update().SetVars(newVars).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update provisioned network vars: %v", err)
	}

	// Generate router IP address
	networkAddrParts := strings.Split(entProvisionedNetwork.Cidr, "/")
	networkAddr := networkAddrParts[0]
	networkOctetStrings := strings.Split(networkAddr, ".")
	networkOctets := []byte{0, 0, 0, 0}
	for i, octetString := range networkOctetStrings {
		octet, err := strconv.Atoi(octetString)
		if err != nil {
			return fmt.Errorf("error while parsing IPv4 Address %s: %v", entProvisionedNetwork.Cidr, err)
		}
		networkOctets[i] = byte(octet)
	}
	_, ipv4Net, err := net.ParseCIDR(entProvisionedNetwork.Cidr)
	if err != nil {
		return fmt.Errorf("error while parsing cidr: %v", err)
	}
	if len(ipv4Net.Mask) != 4 {
		return fmt.Errorf("mask is not correct length")
	}
	routerAddress := strings.Join(append(networkOctetStrings[:3], "254"), ".")

	// Create openstack subnet on network
	osSubnet, err := subnets.Create(networkClient, subnets.CreateOpts{
		NetworkID:       osNetwork.ID,
		CIDR:            entNetwork.Cidr,
		Name:            builder.generateNetworkName(entCompetition, entTeam, entNetwork, entBuild),
		Description:     fmt.Sprintf("%s@%d Subnet for  Network \"%s\" for Team %d", entEnvironment.Name, entBuild.EnvironmentRevision, entNetwork.Name, entTeam.TeamNumber),
		AllocationPools: []subnets.AllocationPool{},
		GatewayIP:       &routerAddress,
		IPVersion:       gophercloud.IPv4,
	}).Extract()
	if err != nil {
		return fmt.Errorf("failed to create subnet: %v", err)
	}

	// Store Openstack subnet ID in provisioned network vars
	newVars["openstack_subnet_id"] = osSubnet.ID
	err = entProvisionedNetwork.Update().SetVars(newVars).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update provisioned network vars: %v", err)
	}

	// Create Openstack port for router on subnet
	osPort, err := ports.Create(networkClient, ports.CreateOpts{
		NetworkID:    osNetwork.ID,
		Description:  fmt.Sprintf("%s@%d Router Interface on Network \"%s\" for Team %d", entEnvironment.Name, entBuild.EnvironmentRevision, entNetwork.Name, entTeam.TeamNumber),
		AdminStateUp: gophercloud.Enabled,
		FixedIPs: []ports.IP{{
			SubnetID:  osSubnet.ID,
			IPAddress: routerAddress,
		}},
	}).Extract()
	if err != nil {
		return fmt.Errorf("failed to create port for router: %v", err)
	}

	// Store Openstack port ID in provisioned network vars
	newVars["openstack_subnet_port_id"] = osPort.ID
	err = entProvisionedNetwork.Update().SetVars(newVars).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update provisioned network vars: %v", err)
	}

	// Create Openstack interface on router attached to subnet port
	osRouterId, exists := entTeam.Vars["openstack_router_id"]
	if !exists {
		return fmt.Errorf("failed to get openstack_router_id from team vars")
	}
	osInterface, err := routers.AddInterface(networkClient, osRouterId, routers.AddInterfaceOpts{
		PortID: osPort.ID,
	}).Extract()
	if err != nil {
		return fmt.Errorf("failed to create router interface: %v", err)
	}

	// Store Openstack interface ID in provisioned network vars
	newVars["openstack_router_interface_id"] = osInterface.ID
	err = entProvisionedNetwork.Update().SetVars(newVars).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update provisioned network vars: %v", err)
	}
	return
}

func (builder OpenstackBuilder) DeployTeam(ctx context.Context, entTeam *ent.Team) (err error) {
	entBuild, err := entTeam.QueryTeamToBuild().Only(ctx)
	if err != nil {
		return fmt.Errorf("couldn't query build from team: %v", err)
	}
	entEnvironment, err := entTeam.QueryTeamToBuild().QueryBuildToEnvironment().Only(ctx)
	if err != nil {
		return fmt.Errorf("couldn't query environment from team: %v", err)
	}
	entCompetition, err := entTeam.QueryTeamToBuild().QueryBuildToEnvironment().QueryEnvironmentToCompetition().Only(ctx)
	if err != nil {
		return fmt.Errorf("couldn't query competition from team: %v", err)
	}

	// ###################
	// Wait on open thread
	// ###################
	err = builder.DeployWorkerPool.Acquire(ctx, int64(1))
	if err != nil {
		return fmt.Errorf("failed to acquire thread: %v", err)
	}
	defer builder.DeployWorkerPool.Release(int64(1))

	// #############################
	// Generate authenticated client
	// #############################
	provider, err := builder.newAuthProvider()
	if err != nil {
		return fmt.Errorf("failed to authenticate: %v", err)
	}
	endpointOpts := gophercloud.EndpointOpts{
		Name:   "neutron",
		Region: builder.Config.RegionName,
	}
	networkClient, err := openstack.NewNetworkV2(provider, endpointOpts)
	if err != nil {
		return fmt.Errorf("failed to create openstack network client: %v", err)
	}

	// ###########
	// Deploy team
	// ###########
	// Create router for team
	osRouter, err := routers.Create(networkClient, routers.CreateOpts{
		Name:         builder.generateRouterName(entCompetition, entTeam, entBuild),
		Description:  fmt.Sprintf("%s@%d Router for Team %d", entEnvironment.Name, entBuild.EnvironmentRevision, entTeam.TeamNumber),
		AdminStateUp: gophercloud.Enabled,
		GatewayInfo: &routers.GatewayInfo{
			NetworkID: builder.Config.ExternalNetworkId,
		},
	}).Extract()
	if err != nil {
		return fmt.Errorf("failed to create router: %v", err)
	}

	// Store Openstack router ID in team vars
	newVars := entTeam.Vars
	newVars["openstack_router_id"] = osRouter.ID
	err = entTeam.Update().SetVars(newVars).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update team vars: %v", err)
	}
	return
}

func (builder OpenstackBuilder) TeardownHost(ctx context.Context, entProvisionedHost *ent.ProvisionedHost) (err error) {
	// ###################
	// Wait on open thread
	// ###################
	err = builder.TeardownWorkerPool.Acquire(ctx, int64(1))
	if err != nil {
		return fmt.Errorf("failed to acquire thread: %v", err)
	}
	defer builder.TeardownWorkerPool.Release(int64(1))

	// #############################
	// Generate authenticated client
	// #############################
	provider, err := builder.newAuthProvider()
	if err != nil {
		return fmt.Errorf("failed to authenticate: %v", err)
	}
	endpointOpts := gophercloud.EndpointOpts{
		Region: builder.Config.RegionName,
	}
	computeClient, err := openstack.NewComputeV2(provider, endpointOpts)
	if err != nil {
		return fmt.Errorf("failed to create openstack compute client: %v", err)
	}

	// #############
	// Teardown host
	// #############
	// Delete Openstack instance
	err = servers.Delete(computeClient, entProvisionedHost.Vars["openstack_instance_id"]).ExtractErr()
	if err != nil {
		return fmt.Errorf("failed to delete host: %v", err)
	}
	newVars := entProvisionedHost.Vars
	delete(newVars, "openstack_instance_id")
	err = entProvisionedHost.Update().SetVars(newVars).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update provisioned host vars: %v", err)
	}

	// Delete Openstack security group
	err = secgroups.Delete(computeClient, entProvisionedHost.Vars["openstack_secgroup_id"]).ExtractErr()
	if err != nil {
		return fmt.Errorf("failed to delete security group: %v", err)
	}
	delete(newVars, "openstack_secgroup_id")
	err = entProvisionedHost.Update().SetVars(newVars).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update provisioned host vars: %v", err)
	}
	return
}

func (builder OpenstackBuilder) TeardownNetwork(ctx context.Context, entProvisionedNetwork *ent.ProvisionedNetwork) (err error) {
	entTeam, err := entProvisionedNetwork.QueryProvisionedNetworkToTeam().Only(ctx)
	if err != nil {
		return fmt.Errorf("failed to query team from proviisoned network: %v", err)
	}

	// ###################
	// Wait on open thread
	// ###################
	err = builder.TeardownWorkerPool.Acquire(ctx, int64(1))
	if err != nil {
		return fmt.Errorf("failed to acquire thread: %v", err)
	}
	defer builder.TeardownWorkerPool.Release(int64(1))

	// #############################
	// Generate authenticated client
	// #############################
	provider, err := builder.newAuthProvider()
	if err != nil {
		return fmt.Errorf("failed to authenticate: %v", err)
	}
	endpointOpts := gophercloud.EndpointOpts{
		Name:   "neutron",
		Region: builder.Config.RegionName,
	}
	networkClient, err := openstack.NewNetworkV2(provider, endpointOpts)
	if err != nil {
		return fmt.Errorf("failed to create openstack network client: %v", err)
	}

	// ################
	// Teardown network
	// ################
	// Delete Openstack router interface
	osRouterId, exists := entTeam.Vars["openstack_router_id"]
	if !exists {
		return fmt.Errorf("failed to get openstack_router_id from team vars")
	}
	_, err = routers.RemoveInterface(networkClient, osRouterId, routers.RemoveInterfaceOpts{
		PortID: entProvisionedNetwork.Vars["openstack_subnet_port_id"],
	}).Extract()
	if err != nil {
		return fmt.Errorf("failed to delete router interface: %v", err)
	}
	newVars := entProvisionedNetwork.Vars
	delete(newVars, "openstack_router_interface_id")
	delete(newVars, "openstack_subnet_port_id")
	err = entProvisionedNetwork.Update().SetVars(newVars).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update provisioned network vars: %v", err)
	}

	// Delete Openstack subnet
	err = subnets.Delete(networkClient, entProvisionedNetwork.Vars["openstack_subnet_id"]).ExtractErr()
	if err != nil {
		return fmt.Errorf("failed to delete subnet: %v", err)
	}
	delete(newVars, "openstack_subnet_id")
	err = entProvisionedNetwork.Update().SetVars(newVars).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update provisioned network vars: %v", err)
	}

	// Delete Openstack network
	err = networks.Delete(networkClient, entProvisionedNetwork.Vars["openstack_network_id"]).ExtractErr()
	if err != nil {
		return fmt.Errorf("failed to delete network: %v", err)
	}
	delete(newVars, "openstack_network_id")
	err = entProvisionedNetwork.Update().SetVars(newVars).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update provisioned network vars: %v", err)
	}
	return
}

func (builder OpenstackBuilder) TeardownTeam(ctx context.Context, entTeam *ent.Team) (err error) {
	// ###################
	// Wait on open thread
	// ###################
	err = builder.TeardownWorkerPool.Acquire(ctx, int64(1))
	if err != nil {
		return fmt.Errorf("failed to acquire thread: %v", err)
	}
	defer builder.TeardownWorkerPool.Release(int64(1))

	// #############################
	// Generate authenticated client
	// #############################
	provider, err := builder.newAuthProvider()
	if err != nil {
		return fmt.Errorf("failed to authenticate: %v", err)
	}
	endpointOpts := gophercloud.EndpointOpts{
		Name:   "neutron",
		Region: builder.Config.RegionName,
	}
	networkClient, err := openstack.NewNetworkV2(provider, endpointOpts)
	if err != nil {
		return fmt.Errorf("failed to create openstack network client: %v", err)
	}

	// #############
	// Teardown team
	// #############
	// Delete Openstack router
	err = routers.Delete(networkClient, entTeam.Vars["openstack_router_id"]).ExtractErr()
	if err != nil {
		return fmt.Errorf("failed to delete router: %v", err)
	}
	newVars := entTeam.Vars
	delete(newVars, "openstack_router_id")
	err = entTeam.Update().SetVars(newVars).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update team vars: %v", err)
	}
	return
}
