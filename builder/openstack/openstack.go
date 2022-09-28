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
	"time"

	"golang.org/x/sync/semaphore"

	"github.com/gen0cide/laforge/ent"
	"github.com/gen0cide/laforge/logging"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/bootfromvolume"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/floatingips"
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
	FloatingIPPool     string            `json:"floating_ip_pool"`
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

func (builder OpenstackBuilder) generateBuildID(entBuild *ent.Build) string {
	return string([]rune(entBuild.ID.String())[:8])
}

func (builder OpenstackBuilder) generateVmName(entEnvironment *ent.Environment, entTeam *ent.Team, entHost *ent.Host, entBuild *ent.Build) string {
	return (entEnvironment.Name + "-team-" + fmt.Sprintf("%02d", entTeam.TeamNumber) + "-" + entHost.Hostname + "-" + builder.generateBuildID(entBuild))
}

func (builder OpenstackBuilder) generateRouterName(entEnvironment *ent.Environment, entTeam *ent.Team, entBuild *ent.Build) string {
	return (entEnvironment.Name + "-team-" + fmt.Sprintf("%02d", entTeam.TeamNumber) + "-" + builder.generateBuildID(entBuild))
}

func (builder OpenstackBuilder) generateNetworkName(entEnvironment *ent.Environment, entTeam *ent.Team, entNetwork *ent.Network, entBuild *ent.Build) string {
	return (entEnvironment.Name + "-team-" + fmt.Sprintf("%02d", entTeam.TeamNumber) + "-" + entNetwork.Name + "-" + builder.generateBuildID(entBuild))
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

func waitForObjectTeardown(getFunc func() error) {
	for {
		err := getFunc()
		if err != nil {
			break
		}
		time.Sleep(3 * time.Second)
	}
}

func waitForObject(getFunc func() (bool, error)) error {
	for {
		valid, err := getFunc()
		if valid {
			return nil
		}
		if err != nil {
			return err
		}
		time.Sleep(3 * time.Second)
	}
}

func (builder OpenstackBuilder) DeployHost(ctx context.Context, entProvisionedHost *ent.ProvisionedHost) (err error) {
	ctxClosing := context.Background()
	defer ctxClosing.Done()
	entHost, err := entProvisionedHost.QueryProvisionedHostToHost().Only(ctx)
	if err != nil {
		return fmt.Errorf("failed querying host from provisioned host: %v", err)
	}
	entDisk, err := entHost.QueryHostToDisk().Only(ctx)
	if err != nil {
		return fmt.Errorf("failed querying disk from host: %v", err)
	}
	entProvisionedNetwork, err := entProvisionedHost.QueryProvisionedHostToProvisionedNetwork().Only(ctx)
	if err != nil {
		return fmt.Errorf("failed to query provisioned network from provisioned host: %v", err)
	}
	entNetwork, err := entProvisionedNetwork.QueryProvisionedNetworkToNetwork().Only(ctx)
	if err != nil {
		return fmt.Errorf("failed to query network from provisioned network: %v", err)
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
	entEnvironment, err := entBuild.QueryBuildToEnvironment().Only(ctx)
	if err != nil {
		return fmt.Errorf("failed querying environment from build: %v", err)
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
	vmName := builder.generateVmName(entEnvironment, entTeam, entHost, entBuild)

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
	err = entProvisionedHost.Update().SetVars(newVars).Exec(ctxClosing)
	if err != nil {
		return fmt.Errorf("failed to update provisioned host vars: %v", err)
	}
	input_cidr := entProvisionedNetwork.Cidr
	if entNetwork.VdiVisible {
		vpcCidr, ok := entEnvironment.Config["vpc_cidr"]
		if !ok {
			return fmt.Errorf("couldn't find vpc_cidr in environment \"%v\"", entEnvironment.Name)
		}
		input_cidr = vpcCidr
	}
	if entProvisionedNetwork.Name == "vdi" {
		vdiWhitelist, ok := entEnvironment.Config["vdi_whitelist"]
		if !ok {
			return fmt.Errorf("couldn't find vdi_whitelist in environment \"%v\"", entEnvironment.Name)
		}
		input_cidr = vdiWhitelist
	}
	if entNetwork.Vars["public_net"] == "true" {
		input_cidr = "0.0.0.0/0"
	}
	ruleOpts := make([]secgroups.CreateRuleOpts, 0)
	for _, port := range entHost.ExposedTCPPorts {
		fromPort := 0
		toPort := 0
		portRange := strings.Split(port, "-")
		if len(portRange) == 1 {
			destPort, err := strconv.Atoi(portRange[0])
			if err != nil {
				return fmt.Errorf("could not convert TCP port \"%s\" to integer: %v", port, err)
			}
			fromPort = destPort
			toPort = destPort
		} else {
			fromPort, err = strconv.Atoi(portRange[0])
			if err != nil {
				return fmt.Errorf("could not convert TCP port \"%s\" to integer: %v", port, err)
			}
			toPort, err = strconv.Atoi(portRange[1])
			if err != nil {
				return fmt.Errorf("could not convert TCP port \"%s\" to integer: %v", port, err)
			}
		}
		opts := secgroups.CreateRuleOpts{
			ParentGroupID: osSecGroup.ID,
			FromPort:      fromPort,
			ToPort:        toPort,
			IPProtocol:    "TCP",
			CIDR:          input_cidr,
		}
		ruleOpts = append(ruleOpts, opts)
	}
	for _, port := range entHost.ExposedUDPPorts {
		fromPort := 0
		toPort := 0
		portRange := strings.Split(port, "-")
		if len(portRange) == 1 {
			destPort, err := strconv.Atoi(portRange[0])
			if err != nil {
				return fmt.Errorf("could not convert TCP port \"%s\" to integer: %v", port, err)
			}
			fromPort = destPort
			toPort = destPort
		} else {
			fromPort, err = strconv.Atoi(portRange[0])
			if err != nil {
				return fmt.Errorf("could not convert TCP port \"%s\" to integer: %v", port, err)
			}
			toPort, err = strconv.Atoi(portRange[1])
			if err != nil {
				return fmt.Errorf("could not convert TCP port \"%s\" to integer: %v", port, err)
			}
		}
		opts := secgroups.CreateRuleOpts{
			ParentGroupID: osSecGroup.ID,
			FromPort:      fromPort,
			ToPort:        toPort,
			IPProtocol:    "UDP",
			CIDR:          input_cidr,
		}
		ruleOpts = append(ruleOpts, opts)
	}
	// Always allow ICMP to hosts
	icmp_opts := secgroups.CreateRuleOpts{
		ParentGroupID: osSecGroup.ID,
		FromPort:      -1,
		ToPort:        -1,
		IPProtocol:    "ICMP",
		CIDR:          "0.0.0.0/0",
	}
	ruleOpts = append(ruleOpts, icmp_opts)
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
	if strings.HasPrefix(entHost.OS, "w2k") || strings.HasPrefix(entHost.OS, "win") {
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

	blockOps := []bootfromvolume.BlockDevice{
		{
			UUID:                builder.Config.Images[entHost.OS],
			BootIndex:           0,
			DeleteOnTermination: true,
			DestinationType:     bootfromvolume.DestinationVolume,
			SourceType:          bootfromvolume.SourceImage,
			VolumeSize:          entDisk.Size,
		},
	}

	hostOps := servers.CreateOpts{
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
	}

	createOpts := bootfromvolume.CreateOptsExt{
		CreateOptsBuilder: hostOps,
		BlockDevice:       blockOps,
	}

	// Create the host
	builder.Logger.Log.Debugf("Deploying host with image \"%s\" and flavor \"%s\"", builder.Config.Images[entHost.OS], builder.Config.Flavors[entHost.InstanceSize])
	osServer, err := bootfromvolume.Create(computeClient, createOpts).Extract()
	if err != nil {
		return fmt.Errorf("failed to create server: %v", err)
	}

	// Store Openstack instance ID in provisioned host vars
	newVars["openstack_instance_id"] = osServer.ID
	err = entProvisionedHost.Update().SetVars(newVars).Exec(ctxClosing)
	if err != nil {
		return fmt.Errorf("failed to update provisioned host vars: %v", err)
	}

	// Deply host with public IP
	if entNetwork.Vars["public_net"] == "true" {
		floatingIPOpts := floatingips.CreateOpts{
			Pool: builder.Config.FloatingIPPool,
		}
		fipResult, err := floatingips.Create(computeClient, floatingIPOpts).Extract()
		if err != nil {
			return fmt.Errorf("failed to request floating IP: %v", err)
		}
		fipaddress := fipResult.IP
		// Store Floating IP in provisioned host vars
		newVars["PublicIP"] = fipaddress
		newVars["floatingip_id"] = fipResult.ID
		err = entProvisionedHost.Update().SetVars(newVars).Exec(ctxClosing)
		if err != nil {
			return fmt.Errorf("failed to update provisioned host vars: %v", err)
		}
	}

	err = waitForObject(func() (bool, error) {
		results, err := servers.Get(computeClient, osServer.ID).Extract()
		if err != nil {
			return false, err
		}
		if len(results.AttachedVolumes) == 1 {
			newVars["instance_volume_id"] = results.AttachedVolumes[0].ID
			err = entProvisionedHost.Update().SetVars(newVars).Exec(ctxClosing)
			if err != nil {
				return false, fmt.Errorf("failed to update provisioned host vars: %v", err)
			}
		}
		if results.Status == "ERROR" {
			return false, fmt.Errorf("host in ERROR state")
		}
		if results.Status == "ACTIVE" {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return fmt.Errorf("failed to wait for host to become active: %v", err)
	}
	floatingip, exists := newVars["PublicIP"]
	if exists {
		associateOpts := floatingips.AssociateOpts{
			FloatingIP: floatingip,
		}
		err = floatingips.AssociateInstance(computeClient, newVars["openstack_instance_id"], associateOpts).ExtractErr()
		if err != nil {
			return fmt.Errorf("failed to associate floating IP: %v", err)
		}
	}
	return
}

func (builder OpenstackBuilder) DeployNetwork(ctx context.Context, entProvisionedNetwork *ent.ProvisionedNetwork) (err error) {
	ctxClosing := context.Background()
	defer ctxClosing.Done()
	entBuild, err := entProvisionedNetwork.QueryProvisionedNetworkToBuild().Only(ctx)
	if err != nil {
		return fmt.Errorf("couldn't query build from network \"%s\": %v", entProvisionedNetwork.Name, err)
	}
	entEnvironment, err := entProvisionedNetwork.QueryProvisionedNetworkToBuild().QueryBuildToEnvironment().Only(ctx)
	if err != nil {
		return fmt.Errorf("couldn't query build from network \"%s\": %v", entProvisionedNetwork.Name, err)
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
		Name:         builder.generateNetworkName(entEnvironment, entTeam, entNetwork, entBuild),
		AdminStateUp: gophercloud.Enabled,
	}).Extract()
	if err != nil {
		return fmt.Errorf("failed to create network: %v", err)
	}

	// Store the Openstack network ID in provisioned network vars
	newVars := entProvisionedNetwork.Vars
	newVars["openstack_network_id"] = osNetwork.ID
	err = entProvisionedNetwork.Update().SetVars(newVars).Exec(ctxClosing)
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

	dnsServer := "1.1.1.1"
	envDnsServer, exists := entEnvironment.Config["master_dns_server"]
	if exists {
		dnsServer = envDnsServer
	}

	// Create openstack subnet on network
	osSubnet, err := subnets.Create(networkClient, subnets.CreateOpts{
		NetworkID:       osNetwork.ID,
		CIDR:            entNetwork.Cidr,
		Name:            builder.generateNetworkName(entEnvironment, entTeam, entNetwork, entBuild),
		Description:     fmt.Sprintf("%s@%d Subnet for  Network \"%s\" for Team %d", entEnvironment.Name, entBuild.EnvironmentRevision, entNetwork.Name, entTeam.TeamNumber),
		AllocationPools: []subnets.AllocationPool{},
		GatewayIP:       &routerAddress,
		IPVersion:       gophercloud.IPv4,
		DNSNameservers:  []string{dnsServer},
	}).Extract()
	if err != nil {
		return fmt.Errorf("failed to create subnet: %v", err)
	}

	// Store Openstack subnet ID in provisioned network vars
	newVars["openstack_subnet_id"] = osSubnet.ID
	err = entProvisionedNetwork.Update().SetVars(newVars).Exec(ctxClosing)
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
	err = entProvisionedNetwork.Update().SetVars(newVars).Exec(ctxClosing)
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
	err = entProvisionedNetwork.Update().SetVars(newVars).Exec(ctxClosing)
	if err != nil {
		return fmt.Errorf("failed to update provisioned network vars: %v", err)
	}
	return
}

func (builder OpenstackBuilder) DeployTeam(ctx context.Context, entTeam *ent.Team) (err error) {
	ctxClosing := context.Background()
	defer ctxClosing.Done()
	entBuild, err := entTeam.QueryTeamToBuild().Only(ctx)
	if err != nil {
		return fmt.Errorf("couldn't query build from team: %v", err)
	}
	entEnvironment, err := entTeam.QueryTeamToBuild().QueryBuildToEnvironment().Only(ctx)
	if err != nil {
		return fmt.Errorf("couldn't query environment from team: %v", err)
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
		Name:         builder.generateRouterName(entEnvironment, entTeam, entBuild),
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
	err = entTeam.Update().SetVars(newVars).Exec(ctxClosing)
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

	newVars := entProvisionedHost.Vars
	osServerId, instanceExists := entProvisionedHost.Vars["openstack_instance_id"]

	// ###########
	// Dissasociate floating IP
	// ###########
	floatingIP, exists := newVars["PublicIP"]
	if exists {
		if instanceExists {
			disassociateOpts := floatingips.DisassociateOpts{
				FloatingIP: floatingIP,
			}
			err := floatingips.DisassociateInstance(computeClient, osServerId, disassociateOpts).ExtractErr()
			if err != nil {
				return fmt.Errorf("failed to dissociate IP: %v", err)
			}
		}
		floatingIPID, exists := newVars["floatingip_id"]
		if exists {
			err := floatingips.Delete(computeClient, floatingIPID).ExtractErr()
			if err != nil {
				return fmt.Errorf("failed to delete floating IP: %v", err)
			}
		}
	}

	// #############
	// Teardown host
	// #############
	// Delete Openstack instance if it exists
	if instanceExists {
		err = servers.Delete(computeClient, osServerId).ExtractErr()
		if err != nil {
			return fmt.Errorf("failed to delete host: %v", err)
		}
		// Wait for the network to actually be deleted (Openstack queues actions asynchronously)
		waitForObjectTeardown(func() error {
			_, err := servers.Get(computeClient, osServerId).Extract()
			return err
		})
		// Remove from vars
		delete(newVars, "openstack_instance_id")
		err = entProvisionedHost.Update().SetVars(newVars).Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to update provisioned host vars: %v", err)
		}
	}

	// Delete Openstack security group if it exists
	osSecGroupId, exists := entProvisionedHost.Vars["openstack_secgroup_id"]
	if exists {
		err = secgroups.Delete(computeClient, osSecGroupId).ExtractErr()
		if err != nil {
			return fmt.Errorf("failed to delete security group: %v", err)
		}
		// Wait for the network to actually be deleted (Openstack queues actions asynchronously)
		waitForObjectTeardown(func() error {
			_, err := secgroups.Get(computeClient, osSecGroupId).Extract()
			return err
		})
		// Remove from vars
		delete(newVars, "openstack_secgroup_id")
		err = entProvisionedHost.Update().SetVars(newVars).Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to update provisioned host vars: %v", err)
		}
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
	newVars := entProvisionedNetwork.Vars
	// Delete Openstack router interface if it exists
	osRouterId, exists := entTeam.Vars["openstack_router_id"]
	if exists {
		osSubnetPortId, exists := entProvisionedNetwork.Vars["openstack_subnet_port_id"]
		if exists {
			_, err = routers.RemoveInterface(networkClient, osRouterId, routers.RemoveInterfaceOpts{
				PortID: osSubnetPortId,
			}).Extract()
			if err != nil {
				return fmt.Errorf("failed to delete router interface: %v", err)
			}
			// Wait for the router port to actually be deleted (Openstack queues actions asynchronously)
			waitForObjectTeardown(func() error {
				_, err := ports.Get(networkClient, osSubnetPortId).Extract()
				return err
			})
			// Remove from vars
			delete(newVars, "openstack_router_interface_id")
			delete(newVars, "openstack_subnet_port_id")
			err = entProvisionedNetwork.Update().SetVars(newVars).Exec(ctx)
			if err != nil {
				return fmt.Errorf("failed to update provisioned network vars: %v", err)
			}
		}
	}

	// Delete Openstack subnet if it exists
	osSubnetId, exists := entProvisionedNetwork.Vars["openstack_subnet_id"]
	if exists {
		err = subnets.Delete(networkClient, osSubnetId).ExtractErr()
		if err != nil {
			return fmt.Errorf("failed to delete subnet: %v", err)
		}
		// Wait for the subnet to actually be deleted (Openstack queues actions asynchronously)
		waitForObjectTeardown(func() error {
			_, err := subnets.Get(networkClient, osSubnetId).Extract()
			return err
		})
		// Remove from vars
		delete(newVars, "openstack_subnet_id")
		err = entProvisionedNetwork.Update().SetVars(newVars).Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to update provisioned network vars: %v", err)
		}
	}

	// Delete Openstack network if it exists
	osNetworkId, exists := entProvisionedNetwork.Vars["openstack_network_id"]
	if exists {
		err = networks.Delete(networkClient, osNetworkId).ExtractErr()
		if err != nil {
			return fmt.Errorf("failed to delete network: %v", err)
		}
		// Wait for the network to actually be deleted (Openstack queues actions asynchronously)
		waitForObjectTeardown(func() error {
			_, err := networks.Get(networkClient, osNetworkId).Extract()
			return err
		})
		// Remove from vars
		delete(newVars, "openstack_network_id")
		err = entProvisionedNetwork.Update().SetVars(newVars).Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to update provisioned network vars: %v", err)
		}
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
	newVars := entTeam.Vars
	// Delete Openstack router if it exists
	osRouterId, exists := entTeam.Vars["openstack_router_id"]
	if exists {
		err = routers.Delete(networkClient, osRouterId).ExtractErr()
		if err != nil {
			return fmt.Errorf("failed to delete router: %v", err)
		}
		// Wait for the network to actually be deleted (Openstack queues actions asynchronously)
		waitForObjectTeardown(func() error {
			_, err := routers.Get(networkClient, osRouterId).Extract()
			return err
		})
		// Remove from vars
		delete(newVars, "openstack_router_id")
		err = entTeam.Update().SetVars(newVars).Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to update team vars: %v", err)
		}
	}
	return
}
