package aws

import (
	"context"
	"fmt"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/gen0cide/laforge/ent"
)

const (
	ID          = "aws"
	Name        = "AWS"
	Description = "Builder that interfaces with AWS"
	Author      = "Nicholas Graca <github.com/njg7716>"
	Version     = "0.1"
)

type AWSBuilder struct {
	AWS_Access_Key_Id     string
	AWS_Secret_Access_Key string
	AWS_Session_Token     string
	Region                string
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
	return fmt.Sprintf("%s", buildId)
}

func (builder AWSBuilder) generateVmName(competition *ent.Competition, team *ent.Team, host *ent.Host, build *ent.Build) string {
	return (competition.HclID + "-Team-" + fmt.Sprintf("%02d", team.TeamNumber) + "-" + host.Hostname + "-" + builder.generateBuildID(build))
}
func (builder AWSBuilder) getAMI() string {
	//TODO: quick search function that you'd be able to use to get that info, specifically ones that take the filters for name, virtualization-type, root-device-type,  architecture, and owner
	var ami string = "test"
	return ami
}

func (builder AWSBuilder) DeployHost(ctx context.Context, provisionedHost *ent.ProvisionedHost) (err error) {

	// Get information about host from ENT
	host, err := provisionedHost.QueryProvisionedHostToHost().Only(ctx)
	if err != nil {
		return err
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

	entProNetwork, err := provisionedHost.QueryProvisionedHostToProvisionedNetwork().Only(ctx)

	//https://aws.github.io/aws-sdk-go-v2/docs/configuring-sdk/#specifying-credentials
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(builder.Region))
	if err != nil {
		return err
	}

	// Describe the host with info from above and get ready to deploy
	client := ec2.NewFromConfig(cfg)

	var numInstances int32 = 1
	var instanceType types.InstanceType

	switch host.InstanceSize {
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

	//Relate the OS to an AMI
	ami := builder.getAMI()

	vmName := builder.generateVmName(competition, team, host, build)
	vpcID, ok := entProNetwork.HCLProvisionedNetworkToTeam.Vars["VpcId"]
	if !ok {
		return fmt.Errorf("couldn't find vpcID in environment \"%v\"", entProNetwork.HCLProvisionedNetworkToTeam.TeamNumber)
	}
	// Before we can create a host, we need to create a Security Group for the host to be in
	var desc string = vmName + "'s Security Group"
	secGroupinput := &ec2.CreateSecurityGroupInput{
		Description: &desc,
		GroupName:   &desc,
		VpcId:       &vpcID,
	}

	// Deploy Security Group
	SecGroupResults, err := client.CreateSecurityGroup(ctx, secGroupinput)
	if err != nil {
		return err
	}
	// Save the Security Group ID so we can deploy the host and tear it down later
	sgID := SecGroupResults.GroupId
	var secGroupID []string
	secGroupID[0] = *SecGroupResults.GroupId

	// describe the host to deploy
	input := &ec2.RunInstancesInput{
		ImageId:          &ami,
		InstanceType:     instanceType,
		MinCount:         &numInstances,
		MaxCount:         &numInstances,
		SecurityGroupIds: secGroupID,
		ClientToken:      &vmName,
		PrivateIpAddress: &provisionedHost.SubnetIP,
		SubnetId:         &vpcID,
	}

	// Deploy Host
	result, err := client.RunInstances(ctx, input)
	if err != nil {
		return err
	}
	id := *result.Instances[0].InstanceId

	//Expose TCP ports both Egress and Ingress
	for i := range host.ExposedTCPPorts {
		port, ok := strconv.Atoi(host.ExposedTCPPorts[i])
		if ok != nil {
			return ok
		}
		egressinput := &ec2.AuthorizeSecurityGroupEgressInput{
			GroupId: aws.String(*sgID),
			IpPermissions: []types.IpPermission{
				{
					FromPort:   aws.Int32(int32(port)),
					IpProtocol: aws.String("tcp"),
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
			GroupId: aws.String(*sgID),
			IpPermissions: []types.IpPermission{
				{
					FromPort:   aws.Int32(int32(port)),
					IpProtocol: aws.String("tcp"),
					IpRanges: []types.IpRange{
						{
							CidrIp: aws.String(provisionedHost.SubnetIP),
						},
					},
					ToPort: aws.Int32(int32(port)),
				},
			},
		}
		egressResult, err := client.AuthorizeSecurityGroupEgress(ctx, egressinput)
		if err != nil {
			return err
		}
		_ = egressResult
		ingressResult, err := client.AuthorizeSecurityGroupIngress(ctx, ingressinput)
		if err != nil {
			return err
		}
		_ = ingressResult
	}
	// Expose UDP Ports both egress and ingress
	for i := range host.ExposedUDPPorts {
		port, ok := strconv.Atoi(host.ExposedUDPPorts[i])
		if ok != nil {
			return ok
		}
		egressinput := &ec2.AuthorizeSecurityGroupEgressInput{
			GroupId: aws.String(*sgID),
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
			GroupId: aws.String(*sgID),
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
		egressResult, err := client.AuthorizeSecurityGroupEgress(ctx, egressinput)
		if err != nil {
			return err
		}
		_ = egressResult
		ingressResult, err := client.AuthorizeSecurityGroupIngress(ctx, ingressinput)
		if err != nil {
			return err
		}
		_ = ingressResult
	}

	// Get InstanceId and store it in ENT to access later
	//TODO add a vars field to ProvisionedHost and update that instead of the vars field for the host object
	newVars := provisionedHost.HCLProvisionedHostToHost.Vars
	newVars["InstanceId"] = id
	newVars["SecGroupId"] = secGroupID[0]
	err = host.Update().SetVars(newVars).Exec(ctx)
	if err != nil {
		return err
	}

	return

}

// Deploys Subnets
func (builder AWSBuilder) DeployNetwork(ctx context.Context, provisionedNetwork *ent.ProvisionedNetwork) (err error) {

	// Get information about Network from ENT
	entTeam, err := provisionedNetwork.QueryProvisionedNetworkToTeam().Only(ctx)
	if err != nil {
		return fmt.Errorf("couldn't query build from team \"%d\": %v", entTeam.TeamNumber, err)
	}
	entNetworks, err := provisionedNetwork.QueryProvisionedNetworkToNetwork().All(ctx)
	if err != nil {
		return fmt.Errorf("couldn't query build from network \"%s\": %v", provisionedNetwork.Name, err)
	}

	//https://aws.github.io/aws-sdk-go-v2/docs/configuring-sdk/#specifying-credentials
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(builder.Region))
	if err != nil {
		return err
	}

	client := ec2.NewFromConfig(cfg)
	vpcID, ok := entTeam.Vars["VpcId"]
	if !ok {
		return fmt.Errorf("couldn't find vpc_cidr in environment \"%v\"", entTeam.TeamNumber)
	}
	// There can be multiple subnets per team so loop through them all and creat them
	for i := range entNetworks {
		//Describe subnet to create
		subnetInput := &ec2.CreateSubnetInput{
			VpcId:     &vpcID,
			CidrBlock: &entNetworks[i].Cidr,
		}
		result, err := client.CreateSubnet(ctx, subnetInput)
		if err != nil {
			return err
		}
		subnetID := *result.Subnet.SubnetId
		if err != nil {
			return err
		}
		// Store subnetID so that it can be used later and torn down
		// TODO add a vars field to ProvisionedNetwork and use that to hold vars for networks indtead of the network object
		newVars := entNetworks[i].Vars
		newVars["SubnetID"] = subnetID
		err = entNetworks[i].Update().SetVars(newVars).Exec(ctx)
		if err != nil {
			return err
		}
	}
	return
}

// Terminates a host and its security group
func (builder AWSBuilder) TeardownHost(ctx context.Context, provisionedHost *ent.ProvisionedHost) (err error) {
	host, err := provisionedHost.QueryProvisionedHostToHost().Only(ctx)

	//https://aws.github.io/aws-sdk-go-v2/docs/configuring-sdk/#specifying-credentials
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(builder.Region))
	if err != nil {
		return err
	}
	client := ec2.NewFromConfig(cfg)
	// Get instanceID to terminate before terminating the corresponding security group
	instance, ok := host.Vars["InstanceId"]
	if !ok {
		return fmt.Errorf("couldn't find InstanceID in environment \"%v\"", host.Hostname)
	}
	var instances []string
	instances[0] = instance

	input := &ec2.TerminateInstancesInput{
		InstanceIds: instances,
	}
	results, err := client.TerminateInstances(ctx, input)
	if err != nil {
		return err
	}
	_ = results
	//Get security group ID to terminate
	secGroupID, ok := host.Vars["SecGroupId"]
	if !ok {
		return fmt.Errorf("couldn't find SecGroupID in environment \"%v\"", host.Hostname)
	}
	secGroupInput := &ec2.DeleteSecurityGroupInput{
		GroupId: &secGroupID,
	}
	secGroupResults, err := client.DeleteSecurityGroup(ctx, secGroupInput)
	if err != nil {
		return err
	}
	_ = secGroupResults

	return
}

// DeletesSubnets
func (builder AWSBuilder) TeardownNetwork(ctx context.Context, provisionedNetwork *ent.ProvisionedNetwork) (err error) {
	entNetworks, err := provisionedNetwork.QueryProvisionedNetworkToNetwork().All(ctx)

	//https://aws.github.io/aws-sdk-go-v2/docs/configuring-sdk/#specifying-credentials
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(builder.Region))
	if err != nil {
		return err
	}
	// There can be multiple subnets per team, loop through them all and terminate them
	for i := range entNetworks {
		client := ec2.NewFromConfig(cfg)
		// Get subnet ID
		subnetID, ok := entNetworks[i].Vars["SubnetID"]
		if !ok {
			return fmt.Errorf("couldn't find SebnetID in environment \"%v\"", entNetworks[i].Name)
		}
		subnetInput := &ec2.DeleteSubnetInput{
			SubnetId: &subnetID,
		}
		subnetResults, err := client.DeleteSubnet(ctx, subnetInput)
		_ = subnetResults
		if err != nil {
			return err
		}
	}

	return
}

// Deploys VPC, one per team
func (builder AWSBuilder) DeployTeam(ctx context.Context, entTeam *ent.Team) (err error) {

	//https://aws.github.io/aws-sdk-go-v2/docs/configuring-sdk/#specifying-credentials
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(builder.Region))
	if err != nil {
		return err
	}
	// Describe the network with info from above and get ready to deploy
	client := ec2.NewFromConfig(cfg)
	input := &ec2.CreateVpcInput{
		CidrBlock: &entTeam.Cidr, //TODO Add VPC Cidr to team object
	}

	// Deploy Network
	results, err := client.CreateVpc(ctx, input)
	if err != nil {
		return err
	}
	id := *results.Vpc.VpcId
	//Store VPC ID so that it can be used and torn down later
	newVars := entTeam.Vars
	newVars["VpcId"] = id
	err = entTeam.Update().SetVars(newVars).Exec(ctx)

	return
}

// Terminates VPC
func (builder AWSBuilder) TeardownTeam(ctx context.Context, entTeam *ent.Team) (err error) {
	//https://aws.github.io/aws-sdk-go-v2/docs/configuring-sdk/#specifying-credentials
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(builder.Region))
	if err != nil {
		return err
	}
	vpcID, ok := entTeam.Vars["VpcId"]
	if !ok {
		return fmt.Errorf("couldn't find vpcID in environment \"%v\"", entTeam.TeamNumber)
	}

	client := ec2.NewFromConfig(cfg)

	input := &ec2.DeleteVpcInput{
		VpcId: &vpcID,
	}
	results, err := client.DeleteVpc(ctx, input)
	if err != nil {
		return err
	}
	_ = results
	return
}
