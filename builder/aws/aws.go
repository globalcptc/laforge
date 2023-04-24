package aws

import (
	"context"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/gen0cide/laforge/ent"
	"github.com/gen0cide/laforge/logging"
	"golang.org/x/sync/semaphore"
)

const (
	ID          = "aws"
	Name        = "AWS"
	Description = "Builder that interfaces with AWS"
	Author      = "Nicholas Graca <github.com/njg7716>"
	Version     = "1.0"
)

type AWSBuilder struct {
	Client             *ec2.Client
	Logger             *logging.Logger
	AMIConfig          map[string]AMIConfigStruct
	Config             AWSBuilderConfig
	DeployWorkerPool   *semaphore.Weighted
	TeardownWorkerPool *semaphore.Weighted
}

type AWSBuilderConfig struct {
	ServerUrl          string                     `json:"server_url"`
	AWSConfigFile      string                     `json:"aws_config_file"`
	Region             string                     `json:"region"`
	AMIConfig          map[string]AMIConfigStruct `json:"ami_configs"`
	MaxBuildWorkers    int                        `json:"max_build_workers"`
	MaxTeardownWorkers int                        `json:"max_teardown_workers"`
}

type AMIConfigStruct struct {
	Name               string `json:"name"`
	RootDeviceType     string `json:"root_device_type"`
	VirtualizationType string `json:"virtualization_type"`
	Architecture       string `json:"architecture"`
	Owner              string `json:"owner"`
}

func (builder AWSBuilder) ID() string {
	return ID
}

func (builder AWSBuilder) Name() string {
	return Name
}

func (builder AWSBuilder) Description() string {
	return Description
}

func (builder AWSBuilder) Author() string {
	return Author
}

func (builder AWSBuilder) Version() string {
	return Version
}

// EC2CreateInstanceAPI defines the interface for the RunInstances function.
type EC2CreateInstanceAPI interface {
	RunInstances(ctx context.Context,
		params *ec2.RunInstancesInput,
		optFns ...func(*ec2.Options)) (*ec2.RunInstancesOutput, error)
}

func (builder AWSBuilder) generateBuildID(entBuild *ent.Build) string {
	return string([]rune(entBuild.ID.String())[:8])
}

func (builder AWSBuilder) generateVmName(entEnvironment *ent.Environment, entTeam *ent.Team, entHost *ent.Host, entBuild *ent.Build, entProvisionedNetwork *ent.ProvisionedNetwork) string {
	return (entEnvironment.Name + "-team-" + fmt.Sprintf("%02d", entTeam.TeamNumber) + "-network-" + entProvisionedNetwork.Name + "-Host-" + entHost.Hostname + "-" + builder.generateBuildID(entBuild))
}

func (builder AWSBuilder) generateVPCName(entEnvironment *ent.Environment, entTeam *ent.Team, entBuild *ent.Build) string {
	return (entEnvironment.Name + "-team-" + fmt.Sprintf("%02d", entTeam.TeamNumber) + "-" + builder.generateBuildID(entBuild))
}

func (builder AWSBuilder) generateSubnetName(entEnvironment *ent.Environment, entTeam *ent.Team, entBuild *ent.Build, entProvisionedNetwork *ent.ProvisionedNetwork) string {
	return (entEnvironment.Name + "-team-" + fmt.Sprintf("%02d", entTeam.TeamNumber) + "-network-" + entProvisionedNetwork.Name + "-" + builder.generateBuildID(entBuild))
}
func (builder AWSBuilder) generatePublicSubnetName(entEnvironment *ent.Environment, entTeam *ent.Team, entBuild *ent.Build) string {
	return (entEnvironment.Name + "-team-" + fmt.Sprintf("%02d", entTeam.TeamNumber) + "-public_subnet-" + builder.generateBuildID(entBuild))
}

func (builder AWSBuilder) getAMI(ctx context.Context, name, vt, rdt, arch, owner string) (string, error) {

	// Describe the host with info from above and get ready to deploy
	input := ec2.DescribeImagesInput{
		DryRun:          aws.Bool(false),
		ExecutableUsers: []string{"all"},
		Filters: []types.Filter{
			{Name: aws.String("name"), Values: []string{name}},
			{Name: aws.String("root-device-type"), Values: []string{rdt}},
			{Name: aws.String("virtualization-type"), Values: []string{vt}},
			{Name: aws.String("architecture"), Values: []string{arch}},
		},
		ImageIds:          []string{},
		IncludeDeprecated: aws.Bool(false),
		Owners:            []string{owner},
	}
	output, err := builder.Client.DescribeImages(ctx, &input)
	if err != nil {
		return "", err
	}
	if len(output.Images) > 0 {
		image := output.Images[0]
		return *image.ImageId, nil
	} else {
		return "", fmt.Errorf("no images found")
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

func (builder AWSBuilder) DeployHost(ctx context.Context, provisionedHost *ent.ProvisionedHost) (err error) {
	ctxClosing := context.Background()
	defer ctxClosing.Done()
	// Get information about host from ENT
	entHost, err := provisionedHost.QueryProvisionedHostToHost().Only(ctx)
	if err != nil {
		return fmt.Errorf("couldn't query host from provisioned host \"%v\": %v", entHost.Hostname, err)
	}

	entBuild, err := provisionedHost.QueryProvisionedHostToPlan().QueryBuild().Only(ctx)
	if err != nil {
		return fmt.Errorf("couldn't query build from provisioned host \"%v\": %v", entHost.Hostname, err)
	}
	entCompetition, err := entBuild.QueryCompetition().Only(ctx)
	if err != nil {
		return fmt.Errorf("couldn't query competition from build \"%v\": %v", entBuild.ID, err)
	}
	entTeam, err := provisionedHost.QueryProvisionedHostToProvisionedNetwork().QueryProvisionedNetworkToTeam().Only(ctx)
	if err != nil {
		return fmt.Errorf("couldn't query team from provisioned host \"%v\": %v", entHost.Hostname, err)
	}
	agentFile, err := provisionedHost.QueryProvisionedHostToGinFileMiddleware().First(ctx)
	if err != nil {
		return fmt.Errorf("error while querying gin file middleware from provisioned host: %v", err)
	}

	entProNetwork, err := provisionedHost.QueryProvisionedHostToProvisionedNetwork().Only(ctx)
	if err != nil {
		return fmt.Errorf("couldn't query provisioned network from provisioned host \"%v\": %v", entHost.Hostname, err)
	}

	entNetwork, err := entProNetwork.QueryProvisionedNetworkToNetwork().Only(ctx)
	if err != nil {
		return fmt.Errorf("couldn't query network from provisioned network \"%v\": %v", entProNetwork.Name, err)
	}

	entEnvironment, err := entBuild.QueryEnvironment().Only(ctx)
	if err != nil {
		return fmt.Errorf("couldn't query environment from competition \"%v\": %v", entCompetition.ID, err)
	}
	// Get InstanceId and store it in ENT to access later
	newVars := provisionedHost.Vars

	// Describe the host with info from above and get ready to deploy

	var numInstances int32 = 1
	var instanceType types.InstanceType

	switch entHost.InstanceSize {
	case "nano":
		instanceType = types.InstanceTypeT2Nano
	case "micro":
		instanceType = types.InstanceTypeT2Micro
	case "small":
		instanceType = types.InstanceTypeT2Small
	case "medium":
		instanceType = types.InstanceTypeT2Medium
	case "large":
		instanceType = types.InstanceTypeT2Large
	case "xlarge":
		instanceType = types.InstanceTypeT2Xlarge
	}

	amiConfig, ok := builder.AMIConfig[entHost.OS]
	if !ok {
		return fmt.Errorf("no AMI config found for %s", entHost.OS)
	}

	//Relate the OS to an AMI
	ami, err := builder.getAMI(ctx,
		amiConfig.Name,
		amiConfig.VirtualizationType,
		amiConfig.RootDeviceType,
		amiConfig.Architecture,
		amiConfig.Owner,
	)

	vmName := builder.generateVmName(entEnvironment, entTeam, entHost, entBuild, entProNetwork)
	vpcID, ok := entTeam.Vars["VpcId"]
	if !ok {
		return fmt.Errorf("couldn't find vpcID in ProNetwork \"%v\"", entProNetwork.ID)
	}
	subnetID, ok := entProNetwork.Vars["SubnetID"]
	if !ok {
		return fmt.Errorf("couldn't find subnetID in ProNetwork \"%v\"", entProNetwork.ID)
	}

	// ###################
	// Wait on open thread
	// ###################
	err = builder.DeployWorkerPool.Acquire(ctx, int64(1))
	if err != nil {
		return fmt.Errorf("failed to acquire thread: %v", err)
	}
	defer builder.DeployWorkerPool.Release(int64(1))

	// Before we can create a host, we need to create a Security Group for the host to be in
	desc := vmName + "_sec_group"
	secGroupinput := &ec2.CreateSecurityGroupInput{
		Description: &desc,
		GroupName:   &desc,
		TagSpecifications: []types.TagSpecification{{
			ResourceType: "security-group",
			Tags:         []types.Tag{{Key: aws.String("Name"), Value: aws.String(desc)}},
		}},
		VpcId: &vpcID,
	}

	// Deploy Security Group
	SecGroupResults, err := builder.Client.CreateSecurityGroup(ctx, secGroupinput)
	if err != nil {
		return fmt.Errorf("error creating : %v", err)
	}
	// Save the Security Group ID so we can deploy the host and tear it down later
	sgID := *SecGroupResults.GroupId
	newVars["SecGroupId"] = sgID
	err = provisionedHost.Update().SetVars(newVars).Exec(ctxClosing)
	if err != nil {
		return fmt.Errorf("error updating host vars with Instance and SecGroup IDs %v", err)
	}

	agentUrl := fmt.Sprintf("%s/api/download/%s", builder.Config.ServerUrl, agentFile.URLID)
	var code string
	builder.Logger.Log.Debugf("Deploying HostOS : %s", entHost.OS)
	if strings.HasPrefix(entHost.OS, "w2k") {
		code = fmt.Sprintf(`<script>
powershell -Command mkdir $env:PROGRAMDATA\Laforge -Force
powershell -Command do{	$test = Test-Connection 1.1.1.1 -Quiet; Start-Sleep -s 5}until($test)
powershell -Command Invoke-WebRequest %s -OutFile $env:PROGRAMDATA\Laforge\laforge.exe
powershell -Command %%PROGRAMDATA%%\Laforge\laforge.exe -service install
powershell -Command %%PROGRAMDATA%%\Laforge\laforge.exe -service start
powershell -Command logoff
</script>`, agentUrl)
	} else {
		var linuxPassword string
		if len(entHost.OverridePassword) > 0 {
			linuxPassword = entHost.OverridePassword
		} else {
			linuxPassword = entCompetition.RootPassword
		}
		code = fmt.Sprintf(`#!/bin/bash
		while [ ! -f "/laforge.bin" ]
		do
		curl -sL -o /laforge.bin %s
		sleep 10
		done
		chmod +x /laforge.bin
		cd /
		./laforge.bin -service install
		./laforge.bin -service start
		usermod --password $(echo %s | openssl passwd -1 -stdin) laforge
		`, agentUrl, linuxPassword)
	}
	userdata := base64.StdEncoding.EncodeToString([]byte(code))
	// describe the host to deploy
	input := &ec2.RunInstancesInput{
		ImageId:          &ami,
		InstanceType:     instanceType,
		MinCount:         &numInstances,
		MaxCount:         &numInstances,
		SecurityGroupIds: []string{sgID},
		PrivateIpAddress: &provisionedHost.SubnetIP,
		SubnetId:         &subnetID,
		UserData:         &userdata,
		TagSpecifications: []types.TagSpecification{{
			ResourceType: "instance",
			Tags:         []types.Tag{{Key: aws.String("Name"), Value: aws.String(vmName)}},
		}},
	}

	// Deploy Host
	result, err := builder.Client.RunInstances(ctx, input)
	if err != nil {
		for i := 0; i < 5; i++ {
			builder.Logger.Log.Errorf("Error deploying host %v, retrying %v time. Err: %v", provisionedHost.ID, i, err)
			result, err = builder.Client.RunInstances(ctx, input)
			if err == nil {
				break
			}
			time.Sleep(time.Second * 3)
		}
		return fmt.Errorf("error deploying host %v: %v", provisionedHost.ID, err)
	}
	id := *result.Instances[0].InstanceId

	newVars["InstanceId"] = id
	err = provisionedHost.Update().SetVars(newVars).Exec(ctxClosing)
	if err != nil {
		return fmt.Errorf("error updating host vars with Instance and SecGroup IDs %v", err)
	}
	builder.Logger.Log.Debugf("Deployed Host: %s", provisionedHost.ID)

	err = waitForObject(func() (bool, error) {
		builder.Logger.Log.Debugf("Waiting for host %v to be running", provisionedHost.ID)
		// check instance state
		instanceStateResults, err := builder.Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
			InstanceIds: []string{id},
		})
		if err == nil {
			instanceState := instanceStateResults.Reservations[0].Instances[0].State.Name
			if instanceState == "running" {
				return true, nil
			}
		}
		return false, nil
	})
	if err != nil {
		return fmt.Errorf("error waiting for instance to be running %v", err)
	}

	// Deploy host with public IP
	if entNetwork.Vars["public_net"] == "true" {
		// Allocate a public IP
		allocateInput := &ec2.AllocateAddressInput{
			Domain: types.DomainTypeStandard,
		}
		allocateResult, err := builder.Client.AllocateAddress(ctx, allocateInput)
		if err != nil {
			return fmt.Errorf("error allocating IP %v", err)
		}
		allocateID := *allocateResult.AllocationId
		publicIP := *allocateResult.PublicIp
		newVars["AllocationID"] = allocateID
		newVars["PublicIP"] = publicIP
		err = provisionedHost.Update().SetVars(newVars).Exec(ctxClosing)

		associateInput := &ec2.AssociateAddressInput{
			InstanceId:   aws.String(id),
			AllocationId: aws.String(allocateID),
		}
		_, err = builder.Client.AssociateAddress(ctx, associateInput)
		if err != nil {
			return fmt.Errorf("error associating public IP %v : %v", entHost.ID, err)
		}
		builder.Logger.Log.Debugf("Deployed Host: %s with Public IP: %s", provisionedHost.ID, publicIP)
	}

	//Expose TCP ports both Egress and Ingress
	for _, ports := range entHost.ExposedTCPPorts {
		fromPort := 0
		toPort := 0
		input_cidr := entProNetwork.Cidr
		if entNetwork.VdiVisible {
			vpcCidr, ok := entEnvironment.Config["vpc_cidr"]
			if !ok {
				return fmt.Errorf("couldn't find vpc_cidr in environment \"%v\"", entEnvironment.Name)
			}
			input_cidr = vpcCidr
		}
		if strings.HasSuffix(entProNetwork.Name, "vdi") {
			vdiWhitelist, ok := entEnvironment.Config["vdi_whitelist"]
			if !ok {
				return fmt.Errorf("couldn't find vdi_whitelist in environment \"%v\"", entEnvironment.Name)
			}
			input_cidr = vdiWhitelist
		}
		if entNetwork.Vars["public_net"] == "true" {
			input_cidr = "0.0.0.0/0"
		}
		portList := strings.Split(ports, "-")
		if len(portList) == 2 {
			fromPort, err = strconv.Atoi(portList[0])
			if err != nil {
				return fmt.Errorf("error converting Fromport %v from string to int : %v", portList[0], err)
			}
			toPort, err = strconv.Atoi(portList[1])
			if err != nil {
				return fmt.Errorf("error converting Tooport %v from string to int : %v", portList[1], err)
			}
		} else if len(portList) == 1 {
			fromPort, err = strconv.Atoi(portList[0])
			if err != nil {
				return fmt.Errorf("error converting Fromport %v from string to int : %v", portList[0], err)
			}
			toPort = fromPort
		} else {
			return fmt.Errorf("ports not right")
		}
		ingressinput := &ec2.AuthorizeSecurityGroupIngressInput{
			GroupId: aws.String(sgID),
			IpPermissions: []types.IpPermission{
				{
					FromPort:   aws.Int32(int32(fromPort)),
					IpProtocol: aws.String("tcp"),
					IpRanges: []types.IpRange{
						{
							CidrIp: aws.String(input_cidr),
						},
					},
					ToPort: aws.Int32(int32(toPort)),
				},
			},
		}
		_, err = builder.Client.AuthorizeSecurityGroupIngress(ctx, ingressinput)
		if err != nil {
			return fmt.Errorf("error creating ingress rule %v", err)
		}
	}
	builder.Logger.Log.Debugf("Deployed Host: %s with Exposed TCP Ports: %s", provisionedHost.ID, entHost.ExposedTCPPorts)
	// Expose UDP Ports both egress and ingress
	for _, ports := range entHost.ExposedUDPPorts {
		fromPort := 0
		toPort := 0
		input_cidr := entProNetwork.Cidr
		if entProNetwork.Name == "vdi" {
			vdiWhitelist, ok := entEnvironment.Config["vdi_whitelist"]
			if !ok {
				return fmt.Errorf("couldn't find vdi_whitelist in environment \"%v\"", entEnvironment.Name)
			}
			input_cidr = vdiWhitelist
		}
		if entNetwork.Vars["public_net"] == "true" {
			input_cidr = "0.0.0.0/0"
		}
		portList := strings.Split(ports, "-")
		if len(portList) == 2 {
			fromPort, err = strconv.Atoi(portList[0])
			if err != nil {
				return fmt.Errorf("error converting Fromport %v from string to int : %v", portList[0], err)
			}
			toPort, err = strconv.Atoi(portList[1])
			if err != nil {
				return fmt.Errorf("error converting Tooport %v from string to int : %v", portList[1], err)
			}
		} else if len(portList) == 1 {
			fromPort, err = strconv.Atoi(portList[0])
			if err != nil {
				return fmt.Errorf("error converting Fromport %v from string to int : %v", portList[0], err)
			}
			toPort = fromPort
		} else {
			return fmt.Errorf("ports not right")
		}
		ingressinput := &ec2.AuthorizeSecurityGroupIngressInput{
			GroupId: aws.String(sgID),
			IpPermissions: []types.IpPermission{
				{
					FromPort:   aws.Int32(int32(fromPort)),
					IpProtocol: aws.String("udp"),
					IpRanges: []types.IpRange{
						{
							CidrIp: aws.String(input_cidr),
						},
					},
					ToPort: aws.Int32(int32(toPort)),
				},
			},
		}
		_, err = builder.Client.AuthorizeSecurityGroupIngress(ctx, ingressinput)
		if err != nil {
			return fmt.Errorf("error creating ingress rule %v", err)
		}
	}
	builder.Logger.Log.Debugf("Deployed Host: %s with Exposed UDP Ports: %s", provisionedHost.ID, entHost.ExposedUDPPorts)
	return

}

// DeployNetwork creates a subnet
func (builder AWSBuilder) DeployNetwork(ctx context.Context, provisionedNetwork *ent.ProvisionedNetwork) (err error) {
	ctxClosing := context.Background()
	defer ctxClosing.Done()

	// Get information about Network from ENT
	entTeam, err := provisionedNetwork.QueryProvisionedNetworkToTeam().Only(ctx)
	if err != nil {
		return fmt.Errorf("couldn't query build from team \"%d\": %v", entTeam.TeamNumber, err)
	}

	entBuild, err := entTeam.QueryTeamToBuild().Only(ctx)
	if err != nil {
		return fmt.Errorf("couldn't query build from team \"%d\": %v", entTeam.TeamNumber, err)
	}

	entEnvironment, err := entBuild.QueryEnvironment().Only(ctx)
	if err != nil {
		return fmt.Errorf("couldn't query environment from team \"%d\": %v", entTeam.TeamNumber, err)
	}

	entNetwork, err := provisionedNetwork.QueryProvisionedNetworkToNetwork().Only(ctx)
	if err != nil {
		return fmt.Errorf("couldn't query network from team \"%d\": %v", entTeam.TeamNumber, err)
	}

	vpcID, ok := entTeam.Vars["VpcId"]
	if !ok {
		return fmt.Errorf("couldn't find vpc_cidr in team \"%v\"", entTeam.TeamNumber)
	}
	natGatewayID, ok := entTeam.Vars["NatGatewayID"]
	if !ok {
		return fmt.Errorf("couldn't find nat_gateway_id in team \"%v\"", entTeam.TeamNumber)
	}
	newVars := provisionedNetwork.Vars

	// ###################
	// Wait on open thread
	// ###################
	err = builder.DeployWorkerPool.Acquire(ctx, int64(1))
	if err != nil {
		return fmt.Errorf("failed to acquire thread: %v", err)
	}
	defer builder.DeployWorkerPool.Release(int64(1))

	subnetName := builder.generateSubnetName(entEnvironment, entTeam, entBuild, provisionedNetwork)

	//Describe subnet to create
	subnetInput := &ec2.CreateSubnetInput{
		VpcId:     &vpcID,
		CidrBlock: &provisionedNetwork.Cidr,
		TagSpecifications: []types.TagSpecification{{
			ResourceType: "subnet",
			Tags:         []types.Tag{{Key: aws.String("Name"), Value: aws.String(subnetName)}},
		}},
	}
	result, err := builder.Client.CreateSubnet(ctx, subnetInput)
	if err != nil {
		return fmt.Errorf("error creating subnet %v", err)
	}
	subnetID := *result.Subnet.SubnetId
	if err != nil {
		return fmt.Errorf("error getting subnetID from subnet result : %v", err)
	}
	newVars["SubnetID"] = subnetID
	err = provisionedNetwork.Update().SetVars(newVars).Exec(ctxClosing)
	builder.Logger.Log.Debugf("Created Subnet: %s", provisionedNetwork.ID)
	if entNetwork.Vars["public_net"] == "true" {
		builder.Logger.Log.Debug("Not Creating Custom route table for public network")
	} else {
		//Create route table
		routeTableInput := &ec2.CreateRouteTableInput{
			VpcId: &vpcID,
			TagSpecifications: []types.TagSpecification{{
				ResourceType: "route-table",
				Tags:         []types.Tag{{Key: aws.String("Name"), Value: aws.String(subnetName)}},
			}},
		}
		routeTableResult, err := builder.Client.CreateRouteTable(ctx, routeTableInput)
		if err != nil {
			return fmt.Errorf("error creating route table %v", err)
		}
		routeTableID := *routeTableResult.RouteTable.RouteTableId
		newVars["RouteTableID"] = routeTableID
		err = provisionedNetwork.Update().SetVars(newVars).Exec(ctxClosing)
		builder.Logger.Log.Debugf("Created Route Table: %s", provisionedNetwork.ID)
		associateInput := ec2.AssociateRouteTableInput{
			RouteTableId: &routeTableID,
			SubnetId:     &subnetID,
		}
		_, err = builder.Client.AssociateRouteTable(ctx, &associateInput)
		if err != nil {
			return fmt.Errorf("error associating route table %v", err)
		}
		builder.Logger.Log.Debugf("Associated Route Table: %s", provisionedNetwork.ID)
		//Create route
		routeInput := ec2.CreateRouteInput{
			RouteTableId:         &routeTableID,
			DestinationCidrBlock: aws.String("0.0.0.0/0"),
			NatGatewayId:         aws.String(natGatewayID),
		}
		_, err = builder.Client.CreateRoute(ctx, &routeInput)
		if err != nil {
			return fmt.Errorf("error creating route %v", err)
		}
		builder.Logger.Log.Debugf("Created Route: %s", provisionedNetwork.ID)
	}

	if err != nil {
		return fmt.Errorf("error updating network vars with subnetID %v", err)
	}
	return
}

// DeployTeam Deploys VPC for a team
func (builder AWSBuilder) DeployTeam(ctx context.Context, entTeam *ent.Team) (err error) {
	ctxClosing := context.Background()
	defer ctxClosing.Done()

	entBuild, err := entTeam.QueryTeamToBuild().Only(ctx)
	if err != nil {
		return fmt.Errorf("couldn't query build from team \"%v\": %v", entTeam.TeamNumber, err)
	}
	entEnvironment, err := entBuild.QueryEnvironment().Only(ctx)
	if err != nil {
		return fmt.Errorf("couldn't query environment from build \"%v\": %v", entTeam.TeamNumber, err)
	}

	vpcCidr, ok := entEnvironment.Config["vpc_cidr"]
	if !ok {
		return fmt.Errorf("couldn't find vpc_cidr in environment \"%v\"", entEnvironment.Name)
	}
	newVars := entTeam.Vars

	publicCidr, ok := entEnvironment.Config["public_cidr"]
	if !ok {
		return fmt.Errorf("couldn't find public_cidr in environment \"%v\"", entEnvironment.Name)
	}

	// ###################
	// Wait on open thread
	// ###################
	err = builder.DeployWorkerPool.Acquire(ctx, int64(1))
	if err != nil {
		return fmt.Errorf("failed to acquire thread: %v", err)
	}
	defer builder.DeployWorkerPool.Release(int64(1))

	VPCName := builder.generateVPCName(entEnvironment, entTeam, entBuild)

	// Describe the vpc with info from above and get ready to deploy
	input := &ec2.CreateVpcInput{
		CidrBlock: &vpcCidr,
		TagSpecifications: []types.TagSpecification{{
			ResourceType: "vpc",
			Tags:         []types.Tag{{Key: aws.String("Name"), Value: aws.String(VPCName)}},
		}},
	}

	// Deploy VPC
	vpcResults, err := builder.Client.CreateVpc(ctx, input)
	if err != nil {
		return fmt.Errorf("error creating vpc %v", err)
	}
	vpcID := *vpcResults.Vpc.VpcId
	newVars["VpcId"] = vpcID
	err = entTeam.Update().SetVars(newVars).Exec(ctxClosing)
	if err != nil {
		return fmt.Errorf("error updating team vars with VpcID %v", err)
	}

	builder.Logger.Log.Debugf("Created VPC: %s", entTeam.ID)

	err = waitForObject(func() (bool, error) {
		builder.Logger.Log.Debugf("Waiting for VPC %s to be available", vpcID)
		// check vpc state
		vpcInput := &ec2.DescribeVpcsInput{
			VpcIds: []string{vpcID},
		}
		vpcResult, err := builder.Client.DescribeVpcs(ctx, vpcInput)
		if err != nil {
			return false, fmt.Errorf("error describing vpc %v", err)
		}
		state := vpcResult.Vpcs[0].State
		if state == "available" {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return fmt.Errorf("error waiting for vpc to be available %v", err)
	}

	// Create Internet Gateway
	gatewayInput := &ec2.CreateInternetGatewayInput{
		TagSpecifications: []types.TagSpecification{{ResourceType: "internet-gateway", Tags: []types.Tag{{Key: aws.String("Name"), Value: aws.String(VPCName)}}}},
	}

	gatewayResuts, err := builder.Client.CreateInternetGateway(ctx, gatewayInput)
	if err != nil {
		return fmt.Errorf("error creating internet gateway %v", err)
	}
	gatewayID := *gatewayResuts.InternetGateway.InternetGatewayId
	newVars["GatewayId"] = gatewayID
	err = entTeam.Update().SetVars(newVars).Exec(ctxClosing)
	if err != nil {
		return fmt.Errorf("error updating team vars with Internet Gateway ID %v", err)
	}
	builder.Logger.Log.Debugf("Created Internet Gateway: %s", entTeam.ID)
	// Attach internet gateway to VPC
	attachGatewayInput := &ec2.AttachInternetGatewayInput{
		InternetGatewayId: &gatewayID,
		VpcId:             &vpcID,
	}
	_, err = builder.Client.AttachInternetGateway(ctx, attachGatewayInput)
	if err != nil {
		return fmt.Errorf("error attaching internet gateway %v", err)
	}
	builder.Logger.Log.Debugf("Attached Internet Gateway: %s", entTeam.ID)

	subnetName := builder.generatePublicSubnetName(entEnvironment, entTeam, entBuild)

	//Describe subnet to create
	subnetInput := &ec2.CreateSubnetInput{
		VpcId:     &vpcID,
		CidrBlock: &publicCidr,
		TagSpecifications: []types.TagSpecification{{
			ResourceType: "subnet",
			Tags:         []types.Tag{{Key: aws.String("Name"), Value: aws.String(subnetName)}},
		}},
	}
	result, err := builder.Client.CreateSubnet(ctx, subnetInput)
	if err != nil {
		return fmt.Errorf("error creating subnet %v", err)
	}
	publicSubnetID := *result.Subnet.SubnetId
	newVars["SubnetID"] = publicSubnetID
	err = entTeam.Update().SetVars(newVars).Exec(ctxClosing)
	if err != nil {
		return fmt.Errorf("error updating team vars with Public Subnet ID %v", err)
	}
	builder.Logger.Log.Debugf("Created Public Subnet: %s", entTeam.ID)
	err = waitForObject(func() (bool, error) {
		builder.Logger.Log.Debugf("Waiting for Public Subnet %s to be available", publicSubnetID)
		// check subnet state
		subnetInput := &ec2.DescribeSubnetsInput{
			SubnetIds: []string{publicSubnetID},
		}
		subnetResult, err := builder.Client.DescribeSubnets(ctx, subnetInput)
		if err != nil {
			return false, fmt.Errorf("error describing subnet %v", err)
		}
		state := subnetResult.Subnets[0].State
		if state == "available" {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return fmt.Errorf("error waiting for subnet to be available %v", err)
	}

	// Allocate an Elastic IP address
	allocateIPInput := &ec2.AllocateAddressInput{}

	allocateResult, err := builder.Client.AllocateAddress(ctx, allocateIPInput)
	if err != nil {
		return fmt.Errorf("error allocating IP %v", err)
	}
	allocateID := *allocateResult.AllocationId
	newVars["AllocationID"] = allocateID
	err = entTeam.Update().SetVars(newVars).Exec(ctxClosing)
	if err != nil {
		return fmt.Errorf("error updating team vars with AllocationID %v", err)
	}
	builder.Logger.Log.Debugf("Created AllocationID: %s", entTeam.ID)

	// create NAT gateway
	natGatewayInput := &ec2.CreateNatGatewayInput{
		SubnetId:         &publicSubnetID,
		AllocationId:     &allocateID,
		ConnectivityType: types.ConnectivityType("public"),
		TagSpecifications: []types.TagSpecification{{
			ResourceType: "natgateway",
			Tags:         []types.Tag{{Key: aws.String("Name"), Value: aws.String(subnetName)}}},
		},
	}
	natGatewayResults, err := builder.Client.CreateNatGateway(ctx, natGatewayInput)
	if err != nil {
		return fmt.Errorf("error creating nat gateway %v", err)
	}
	natGatewayID := *natGatewayResults.NatGateway.NatGatewayId
	newVars["NatGatewayID"] = natGatewayID
	err = entTeam.Update().SetVars(newVars).Exec(ctxClosing)
	if err != nil {
		return fmt.Errorf("error updating team vars with NatGatewayID %v", err)
	}
	builder.Logger.Log.Debugf("Created NatGateway: %s", entTeam.ID)

	err = waitForObject(func() (bool, error) {
		builder.Logger.Log.Debugf("Waiting for NatGateway %s to be available", natGatewayID)
		//check status of NAT gateway
		natGatewayStatusInput := &ec2.DescribeNatGatewaysInput{
			NatGatewayIds: []string{natGatewayID},
		}
		natGatewayStatusResults, err := builder.Client.DescribeNatGateways(ctx, natGatewayStatusInput)
		if err != nil {
			builder.Logger.Log.Errorf("error describing nat gateway %v", err)
			return false, err
		}
		natGatewayState := natGatewayStatusResults.NatGateways[0].State
		if natGatewayState == "available" {
			builder.Logger.Log.Debugf("Team %v's nat gateway state is %v", entTeam.TeamNumber, natGatewayState)
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return fmt.Errorf("error waiting for nat gateway to be available %v", err)
	}
	// get default route table
	routeTableInput := &ec2.DescribeRouteTablesInput{
		Filters: []types.Filter{{Name: aws.String("vpc-id"), Values: []string{vpcID}}},
	}
	routeTableResults, err := builder.Client.DescribeRouteTables(ctx, routeTableInput)
	if err != nil {
		return fmt.Errorf("error describing route tables %v", err)
	}
	routeTableID := *routeTableResults.RouteTables[0].RouteTableId
	newVars["RouteTableId"] = routeTableID
	err = entTeam.Update().SetVars(newVars).Exec(ctxClosing)
	if err != nil {
		return fmt.Errorf("error updating team vars with RouteTableID %v", err)
	}
	builder.Logger.Log.Debugf("Created RouteTable: %s", entTeam.ID)

	associateInput := ec2.AssociateRouteTableInput{
		RouteTableId: &routeTableID,
		SubnetId:     &publicSubnetID,
	}
	_, err = builder.Client.AssociateRouteTable(ctx, &associateInput)
	if err != nil {
		return fmt.Errorf("error associating route table %v", err)
	}
	builder.Logger.Log.Debugf("Associated RouteTable: %s", entTeam.ID)
	//create defult route
	defaultRouteInput := &ec2.CreateRouteInput{
		RouteTableId:         &routeTableID,
		DestinationCidrBlock: aws.String("0.0.0.0/0"),
		GatewayId:            &gatewayID,
	}
	_, err = builder.Client.CreateRoute(ctx, defaultRouteInput)
	if err != nil {
		return fmt.Errorf("error creating default route %v", err)
	}
	builder.Logger.Log.Debugf("Created default route: %s", entTeam.ID)

	return nil
}

// TeardownHost Terminates a host and its security group
func (builder AWSBuilder) TeardownHost(ctx context.Context, provisionedHost *ent.ProvisionedHost) (err error) {
	err = builder.TeardownWorkerPool.Acquire(ctx, int64(1))
	if err != nil {
		return fmt.Errorf("failed to acquire thread: %v", err)
	}
	defer builder.TeardownWorkerPool.Release(int64(1))
	newVars := provisionedHost.Vars

	// Get instanceID to terminate before terminating the corresponding security group
	instance, ok := provisionedHost.Vars["InstanceId"]
	if ok {
		instances := []string{instance}

		input := &ec2.TerminateInstancesInput{
			InstanceIds: instances,
		}
		_, err = builder.Client.TerminateInstances(ctx, input)
		if err != nil {
			return fmt.Errorf("error terminating instance %v", err)
		}
		builder.Logger.Log.Debugf("Terminated instance: %s", provisionedHost.ID)
		err = waitForObject(func() (bool, error) {
			builder.Logger.Log.Debugf("Waiting for instance %s to be terminated", provisionedHost.ID)
			// check instance state
			instanceStateResults, err := builder.Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
				InstanceIds: []string{instance},
			})
			if err != nil {
				builder.Logger.Log.Errorf("error describing instance %v : %v", instance, err)
				return false, err
			}
			instanceState := instanceStateResults.Reservations[0].Instances[0].State.Name
			if instanceState == "terminated" {
				return true, nil
			}
			return false, nil
		})
		if err != nil {
			return fmt.Errorf("error waiting for instance to be terminated %v", err)
		}
		// Remove from vars
		delete(newVars, "InstanceId")
		err = provisionedHost.Update().SetVars(newVars).Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to update provisioned host vars: %v", err)
		}
	} else {
		builder.Logger.Log.Debugf("No instance id found for host %v", provisionedHost)
	}

	//Get security group ID to terminate
	secGroupID, ok := provisionedHost.Vars["SecGroupId"]
	if ok {
		secGroupInput := &ec2.DeleteSecurityGroupInput{
			GroupId: &secGroupID,
		}
		_, err = builder.Client.DeleteSecurityGroup(ctx, secGroupInput)
		if err != nil {
			return fmt.Errorf("error deleting security group %v", err)
		}
		builder.Logger.Log.Debugf("Deleted security group: %s", provisionedHost.ID)
		// Remove from vars
		delete(newVars, "SecGroupId")
		err = provisionedHost.Update().SetVars(newVars).Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to update provisioned host vars: %v", err)
		}
	} else {
		builder.Logger.Log.Debugf("No security group id found for host %v", provisionedHost)
	}

	allocateID, ok := provisionedHost.Vars["AllocationID"]
	if ok {
		allocateInput := &ec2.ReleaseAddressInput{
			AllocationId: &allocateID,
		}
		_, err = builder.Client.ReleaseAddress(ctx, allocateInput)
		if err != nil {
			return fmt.Errorf("error releasing address %v", err)
		}
		builder.Logger.Log.Debugf("Deleted allocation %v", allocateID)
		// Remove from vars
		delete(newVars, "AllocationID")
		err = provisionedHost.Update().SetVars(newVars).Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to update provisioned host vars: %v", err)
		}
	} else {
		builder.Logger.Log.Debugf("No allocation id found for host %v", provisionedHost)
	}

	return nil
}

// TeardownNetwork deletes a subnet
func (builder AWSBuilder) TeardownNetwork(ctx context.Context, provisionedNetwork *ent.ProvisionedNetwork) (err error) {
	// ###################
	// Wait on open thread
	// ###################
	err = builder.TeardownWorkerPool.Acquire(ctx, int64(1))
	if err != nil {
		return fmt.Errorf("failed to acquire thread: %v", err)
	}
	defer builder.TeardownWorkerPool.Release(int64(1))

	newVars := provisionedNetwork.Vars

	subnetID, ok := provisionedNetwork.Vars["SubnetID"]
	if ok {
		subnetInput := &ec2.DeleteSubnetInput{
			SubnetId: &subnetID,
		}
		_, err = builder.Client.DeleteSubnet(ctx, subnetInput)
		if err != nil {
			return fmt.Errorf("error deleting subnet %v", err)
		}
		builder.Logger.Log.Debugf("Deleted subnet: %s", provisionedNetwork.ID)
		delete(newVars, "SubnetID")
		err = provisionedNetwork.Update().SetVars(newVars).Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to update provisioned network vars: %v", err)
		}
	} else {
		builder.Logger.Log.Debugf("No subnet id found for network %v", provisionedNetwork)
	}

	routeTableID, ok := provisionedNetwork.Vars["RouteTableID"]
	if ok {
		routeTableInput := &ec2.DeleteRouteTableInput{
			RouteTableId: &routeTableID,
		}
		_, err = builder.Client.DeleteRouteTable(ctx, routeTableInput)
		if err != nil {
			return fmt.Errorf("error deleting route table %v", err)
		}
		builder.Logger.Log.Debugf("Deleted route table: %s", provisionedNetwork.ID)
		delete(newVars, "RouteTableID")
		err = provisionedNetwork.Update().SetVars(newVars).Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to update provisioned network vars: %v", err)
		}
	} else {
		builder.Logger.Log.Debugf("No route table id found for network %v", provisionedNetwork)
	}

	return nil
}

// TeardownTeam Terminates VPC
func (builder AWSBuilder) TeardownTeam(ctx context.Context, entTeam *ent.Team) (err error) {
	// ###################
	// Wait on open thread
	// ###################
	err = builder.TeardownWorkerPool.Acquire(ctx, int64(1))
	if err != nil {
		return fmt.Errorf("failed to acquire thread: %v", err)
	}
	defer builder.TeardownWorkerPool.Release(int64(1))

	newVars := entTeam.Vars

	natGatewayID, ok := entTeam.Vars["NatGatewayID"]
	if ok {
		natGatewayInput := &ec2.DeleteNatGatewayInput{
			NatGatewayId: &natGatewayID,
		}
		_, err = builder.Client.DeleteNatGateway(ctx, natGatewayInput)
		if err != nil {
			return fmt.Errorf("error deleting nat gateway %v", err)
		}
		builder.Logger.Log.Debugf("Deleted nat gateway: %s", entTeam.ID)

		err = waitForObject(func() (bool, error) {
			builder.Logger.Log.Debugf("Waiting for nat gateway to be deleted: %s", natGatewayID)
			natGatewayStatusInput := &ec2.DescribeNatGatewaysInput{
				NatGatewayIds: []string{natGatewayID},
			}
			natGatewayStatusResults, err := builder.Client.DescribeNatGateways(ctx, natGatewayStatusInput)
			if err != nil {
				builder.Logger.Log.Errorf("error describing nat gateway %v", err)
				return false, err
			}
			natGatewayState := natGatewayStatusResults.NatGateways[0].State
			if natGatewayState == "deleted" {
				builder.Logger.Log.Debugf("Team %v's nat gateway state is %v", entTeam.TeamNumber, natGatewayState)
				return true, nil
			}
			return false, nil
		})
		if err != nil {
			return fmt.Errorf("failed to wait for gatway to become detached: %v", err)
		}
		delete(newVars, "NatGatewayID")
		err = entTeam.Update().SetVars(newVars).Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to update team vars: %v", err)
		}
	} else {
		builder.Logger.Log.Debugf("No nat gateway found in Team %v", entTeam.TeamNumber)
		return
	}

	subnetID, ok := entTeam.Vars["SubnetID"]
	if ok {
		subnetInput := &ec2.DeleteSubnetInput{
			SubnetId: &subnetID,
		}
		_, err = builder.Client.DeleteSubnet(ctx, subnetInput)
		if err != nil {
			return fmt.Errorf("error deleting subnet %v", err)
		}
		builder.Logger.Log.Debugf("Deleted subnet: %s", entTeam.ID)

		builder.Logger.Log.Debugf("Deleted subnet %v", subnetID)
		delete(newVars, "SubnetID")
		err = entTeam.Update().SetVars(newVars).Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to update team vars: %v", err)
		}
	} else {
		builder.Logger.Log.Debugf("No subnet found in Team %v", entTeam.TeamNumber)
		return
	}

	allocateID, ok := entTeam.Vars["AllocationID"]
	if ok {
		allocateInput := &ec2.ReleaseAddressInput{
			AllocationId: &allocateID,
		}
		_, err = builder.Client.ReleaseAddress(ctx, allocateInput)
		if err != nil {
			return fmt.Errorf("error releasing address %v", err)
		}
		builder.Logger.Log.Debugf("Deleted allocation %v", allocateID)
		delete(newVars, "AllocationID")
		err = entTeam.Update().SetVars(newVars).Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to update team vars: %v", err)
		}
	} else {
		builder.Logger.Log.Debugf("No allocation found in Team %v", entTeam.TeamNumber)
		return
	}

	vpcID, ok := entTeam.Vars["VpcId"]
	if ok {
		gatewayID, ok := entTeam.Vars["GatewayId"]
		if ok {
			// detach internet gateway
			detachInput := &ec2.DetachInternetGatewayInput{
				InternetGatewayId: &gatewayID,
				VpcId:             &vpcID,
			}
			_, err = builder.Client.DetachInternetGateway(ctx, detachInput)
			if err != nil {
				return fmt.Errorf("error detaching internet gateway %v", err)
			}
			builder.Logger.Log.Debugf("Detached internet gateway %v", gatewayID)

			// time.Sleep(time.Second * 30)
			// err = waitForObject(func() (bool, error) {
			// 	// describe internet gateway
			// 	gatewayDescribeInput := &ec2.DescribeInternetGatewaysInput{
			// 		InternetGatewayIds: []string{gatewayID},
			// 	}
			// 	gatewayDescribeResults, err := builder.Client.DescribeInternetGateways(ctx, gatewayDescribeInput)
			// 	if err != nil {
			// 		return false, fmt.Errorf("error describing internet gateway %v", err)
			// 	}
			// 	gatewayState := gatewayDescribeResults.InternetGateways[0].Attachments[0].State
			// 	if gatewayState == "detached" {
			// 		return true, nil
			// 	}
			// 	return false, nil
			// })
			// if err != nil {
			// 	return fmt.Errorf("failed to wait for gatway to become detached: %v", err)
			// }

			gatewayInput := &ec2.DeleteInternetGatewayInput{
				InternetGatewayId: &gatewayID,
			}
			_, err = builder.Client.DeleteInternetGateway(ctx, gatewayInput)
			if err != nil {
				return fmt.Errorf("error deleting gateway %v", err)
			}

			builder.Logger.Log.Debugf("Deleted internet gateway %v", gatewayID)
			delete(newVars, "GatewayId")
			err = entTeam.Update().SetVars(newVars).Exec(ctx)
			if err != nil {
				return fmt.Errorf("failed to update team vars: %v", err)
			}

		} else {
			builder.Logger.Log.Debugf("No internet gateway found in Team %v", entTeam.TeamNumber)
			return
		}
		input := &ec2.DeleteVpcInput{
			VpcId: &vpcID,
		}
		_, err = builder.Client.DeleteVpc(ctx, input)
		if err != nil {
			return fmt.Errorf("error deleting vpc %v", err)
		}
		builder.Logger.Log.Debugf("Deleted vpc %v", vpcID)
		delete(newVars, "VpcId")
		err = entTeam.Update().SetVars(newVars).Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to update team vars: %v", err)
		}

	} else {
		builder.Logger.Log.Debugf("No vpc found in Team %v", entTeam.TeamNumber)
		return
	}

	return
}
