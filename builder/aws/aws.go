package aws

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/gen0cide/laforge/ent"
	"github.com/gen0cide/laforge/logging"
)

const (
	ID          = "aws"
	Name        = "AWS"
	Description = "Builder that interfaces with AWS"
	Author      = "Nicholas Graca <github.com/njg7716>"
	Version     = "0.1"
)

type AWSBuilder struct {
	Client    *ec2.Client
	Logger    *logging.Logger
	AMIConfig map[string]AMIConfigStruct
	Config    AWSBuilderConfig
}

type AWSBuilderConfig struct {
	AWSConfigFile string                     `json:"aws_config_file"`
	Region        string                     `json:"region"`
	AMIConfig     map[string]AMIConfigStruct `json:"ami_configs"`
}

type AMIConfigStruct struct {
	Name               string `json:"name"`
	RootDeviceType     string `json:"root_device_type"`
	VirtualizationType string `json:"virtualization_type"`
	Architecture       string `json:"architecture"`
	Owner              string `json:"owner"`
}

// EC2CreateInstanceAPI defines the interface for the RunInstances function.
type EC2CreateInstanceAPI interface {
	RunInstances(ctx context.Context,
		params *ec2.RunInstancesInput,
		optFns ...func(*ec2.Options)) (*ec2.RunInstancesOutput, error)
}

func (builder AWSBuilder) generateBuildID(build *ent.Build) string {
	buildId, err := build.ID.MarshalText()
	if err != nil {
		buildId = []byte(fmt.Sprint(build.Revision))
	}
	return fmt.Sprintf("%v", buildId)
}

func (builder AWSBuilder) generateVmName(competition *ent.Competition, team *ent.Team, host *ent.Host, build *ent.Build) string {
	return (competition.HclID + "-Team-" + fmt.Sprintf("%02d", team.TeamNumber) + "-" + host.Hostname + "-" + builder.generateBuildID(build))
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

func (builder AWSBuilder) DeployHost(ctx context.Context, provisionedHost *ent.ProvisionedHost) (err error) {

	// Get information about host from ENT
	entHost, err := provisionedHost.QueryProvisionedHostToHost().Only(ctx)
	if err != nil {
		return err
	}

	entBuild, err := provisionedHost.QueryProvisionedHostToPlan().QueryPlanToBuild().Only(ctx)
	if err != nil {
		return
	}
	entCompetition, err := entBuild.QueryBuildToCompetition().Only(ctx)
	if err != nil {
		return
	}
	entTeam, err := provisionedHost.QueryProvisionedHostToProvisionedNetwork().QueryProvisionedNetworkToTeam().Only(ctx)
	if err != nil {
		return
	}

	entProNetwork, err := provisionedHost.QueryProvisionedHostToProvisionedNetwork().Only(ctx)

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

	vmName := builder.generateVmName(entCompetition, entTeam, entHost, entBuild)
	vpcID, ok := entTeam.Vars["VpcId"]
	if !ok {
		return fmt.Errorf("couldn't find vpcID in ProNetwork \"%v\"", entProNetwork.ID)
	}
	subnetID, ok := entProNetwork.Vars["SubnetID"]
	if !ok {
		return fmt.Errorf("couldn't find subnetID in ProNetwork \"%v\"", entProNetwork.ID)
	}
	// Before we can create a host, we need to create a Security Group for the host to be in
	desc := vmName + "'s Security Group"
	secGroupinput := &ec2.CreateSecurityGroupInput{
		Description: &desc,
		GroupName:   &desc,
		VpcId:       &vpcID,
	}

	// Deploy Security Group
	SecGroupResults, err := builder.Client.CreateSecurityGroup(ctx, secGroupinput)
	if err != nil {
		return err
	}
	// Save the Security Group ID so we can deploy the host and tear it down later
	sgID := *SecGroupResults.GroupId

	// describe the host to deploy
	input := &ec2.RunInstancesInput{
		ImageId:          &ami,
		InstanceType:     instanceType,
		MinCount:         &numInstances,
		MaxCount:         &numInstances,
		SecurityGroupIds: []string{sgID},
		ClientToken:      &vmName,
		PrivateIpAddress: &provisionedHost.SubnetIP,
		SubnetId:         &subnetID,
	}

	// Deploy Host
	result, err := builder.Client.RunInstances(ctx, input)
	if err != nil {
		return err
	}
	id := *result.Instances[0].InstanceId

	//Expose TCP ports both Egress and Ingress
	//TODO Egress allow all, Ingress is to and from
	for _, ports := range entHost.ExposedTCPPorts {
		fromPort := 0
		toPort := 0
		portList := strings.Split(ports, "-")
		if len(portList) == 2 {
			fromPort, err = strconv.Atoi(portList[0])
			if err != nil {
				return err
			}
			toPort, err = strconv.Atoi(portList[1])
			if err != nil {
				return err
			}
		} else if len(portList) == 1 {
			fromPort, err = strconv.Atoi(portList[0])
			if err != nil {
				return err
			}
			toPort = fromPort
		} else {
			return fmt.Errorf("ports not right")
		}
		egressinput := &ec2.AuthorizeSecurityGroupEgressInput{
			GroupId: &sgID,
			IpPermissions: []types.IpPermission{
				{
					FromPort:   aws.Int32(int32(fromPort)),
					IpProtocol: aws.String("tcp"),
					IpRanges: []types.IpRange{
						{
							CidrIp: aws.String(provisionedHost.SubnetIP),
						},
					},
					ToPort: aws.Int32(int32(toPort)),
				},
			},
		}
		ingressinput := &ec2.AuthorizeSecurityGroupIngressInput{
			GroupId: aws.String(sgID),
			IpPermissions: []types.IpPermission{
				{
					FromPort:   aws.Int32(int32(fromPort)),
					IpProtocol: aws.String("tcp"),
					IpRanges: []types.IpRange{
						{
							CidrIp: aws.String(provisionedHost.SubnetIP),
						},
					},
					ToPort: aws.Int32(int32(toPort)),
				},
			},
		}
		_, err = builder.Client.AuthorizeSecurityGroupEgress(ctx, egressinput)
		if err != nil {
			return err
		}
		_, err = builder.Client.AuthorizeSecurityGroupIngress(ctx, ingressinput)
		if err != nil {
			return err
		}
	}
	// Expose UDP Ports both egress and ingress
	//TODO ALL ALLOW INGRESS keep Egress the same
	for i := range entHost.ExposedUDPPorts {
		port, ok := strconv.Atoi(entHost.ExposedUDPPorts[i])
		if ok != nil {
			return ok
		}
		egressinput := &ec2.AuthorizeSecurityGroupEgressInput{
			GroupId: aws.String(sgID),
			IpPermissions: []types.IpPermission{
				{
					FromPort:   aws.Int32(int32(port)),
					IpProtocol: aws.String("udp"),
					IpRanges: []types.IpRange{
						{
							CidrIp: aws.String(provisionedHost.SubnetIP),
						},
					},
					ToPort: aws.Int32(int32(port)),
				},
			},
		}
		ingressinput := &ec2.AuthorizeSecurityGroupIngressInput{
			GroupId: aws.String(sgID),
			IpPermissions: []types.IpPermission{
				{
					FromPort:   aws.Int32(int32(port)),
					IpProtocol: aws.String("udp"),
					IpRanges: []types.IpRange{
						{
							CidrIp: aws.String(provisionedHost.SubnetIP),
						},
					},
					ToPort: aws.Int32(int32(port)),
				},
			},
		}
		_, err = builder.Client.AuthorizeSecurityGroupEgress(ctx, egressinput)
		if err != nil {
			return err
		}
		_, err = builder.Client.AuthorizeSecurityGroupIngress(ctx, ingressinput)
		if err != nil {
			return err
		}
	}

	// Get InstanceId and store it in ENT to access later
	//TODO add a vars field to ProvisionedHost and update that instead of the vars field for the host object
	newVars := provisionedHost.Vars
	newVars["InstanceId"] = id
	newVars["SecGroupId"] = sgID
	err = provisionedHost.Update().SetVars(newVars).Exec(ctx)
	if err != nil {
		return err
	}

	return

}

// DeployNetwork creates a subnet
func (builder AWSBuilder) DeployNetwork(ctx context.Context, provisionedNetwork *ent.ProvisionedNetwork) (err error) {

	// Get information about Network from ENT
	entTeam, err := provisionedNetwork.QueryProvisionedNetworkToTeam().Only(ctx)
	if err != nil {
		return fmt.Errorf("couldn't query build from team \"%d\": %v", entTeam.TeamNumber, err)
	}

	vpcID, ok := entTeam.Vars["VpcId"]
	if !ok {
		return fmt.Errorf("couldn't find vpc_cidr in environment \"%v\"", entTeam.TeamNumber)
	}
	//Describe subnet to create
	subnetInput := &ec2.CreateSubnetInput{
		VpcId:     &vpcID,
		CidrBlock: &provisionedNetwork.Cidr,
	}
	result, err := builder.Client.CreateSubnet(ctx, subnetInput)
	if err != nil {
		return err
	}
	subnetID := *result.Subnet.SubnetId
	if err != nil {
		return err
	}
	// Store subnetID so that it can be used later and torn down
	newVars := provisionedNetwork.Vars
	newVars["SubnetID"] = subnetID
	err = provisionedNetwork.Update().SetVars(newVars).Exec(ctx)
	if err != nil {
		return err
	}
	return
}

//TeardownHost Terminates a host and its security group
func (builder AWSBuilder) TeardownHost(ctx context.Context, provisionedHost *ent.ProvisionedHost) (err error) {

	// Get instanceID to terminate before terminating the corresponding security group
	instance, ok := provisionedHost.Vars["InstanceId"]
	if !ok {
		return fmt.Errorf("couldn't find InstanceID in environment \"%v\"", provisionedHost.ID)
	}
	var instances []string
	instances[0] = instance

	input := &ec2.TerminateInstancesInput{
		InstanceIds: instances,
	}
	_, err = builder.Client.TerminateInstances(ctx, input)
	if err != nil {
		return err
	}
	//Get security group ID to terminate
	secGroupID, ok := provisionedHost.Vars["SecGroupId"]
	if !ok {
		return fmt.Errorf("couldn't find SecGroupID in environment \"%v\"", provisionedHost.ID)
	}
	secGroupInput := &ec2.DeleteSecurityGroupInput{
		GroupId: &secGroupID,
	}
	_, err = builder.Client.DeleteSecurityGroup(ctx, secGroupInput)
	if err != nil {
		return err
	}
	return nil
}

// TeardownNetwork deletes a subnet
func (builder AWSBuilder) TeardownNetwork(ctx context.Context, provisionedNetwork *ent.ProvisionedNetwork) (err error) {

	subnetID, ok := provisionedNetwork.Vars["SubnetID"]
	if !ok {
		return fmt.Errorf("couldn't find Subent in PN \"%v\"", provisionedNetwork.Name)
	}
	subnetInput := &ec2.DeleteSubnetInput{
		SubnetId: &subnetID,
	}
	_, err = builder.Client.DeleteSubnet(ctx, subnetInput)
	if err != nil {
		return err
	}
	return nil
}

//DeployTeam Deploys VPC for a team
func (builder AWSBuilder) DeployTeam(ctx context.Context, entTeam *ent.Team) (err error) {

	entEnvironment, err := entTeam.QueryTeamToBuild().QueryBuildToEnvironment().Only(ctx)
	if err != nil {
		return fmt.Errorf("couldn't query enviroment from team \"%v\": %v", entTeam.TeamNumber, err)
	}

	vpcCidr, ok := entEnvironment.Config["vpc_cidr"]
	if !ok {
		return fmt.Errorf("couldn't find vpc_cidr in environment \"%v\"", entEnvironment.Name)
	}

	// Describe the vpc with info from above and get ready to deploy
	input := &ec2.CreateVpcInput{
		CidrBlock: &vpcCidr,
	}

	// Deploy VPC
	results, err := builder.Client.CreateVpc(ctx, input)
	if err != nil {
		return err
	}
	id := *results.Vpc.VpcId

	newVars := entTeam.Vars
	newVars["VpcId"] = id
	err = entTeam.Update().SetVars(newVars).Exec(ctx)
	if err != nil {
		return err
	}
	return nil
}

//TeardownTeam Terminates VPC
func (builder AWSBuilder) TeardownTeam(ctx context.Context, entTeam *ent.Team) (err error) {
	//https://aws.github.io/aws-sdk-go-v2/docs/configuring-sdk/#specifying-credentials

	vpcID, ok := entTeam.Vars["VpcId"]
	if !ok {
		return fmt.Errorf("couldn't find vpcID in environment \"%v\"", entTeam.TeamNumber)
	}

	input := &ec2.DeleteVpcInput{
		VpcId: &vpcID,
	}
	_, err = builder.Client.DeleteVpc(ctx, input)
	if err != nil {
		return err
	}
	return
}
