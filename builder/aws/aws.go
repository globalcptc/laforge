package aws

import (
	"context"
	"fmt"

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
	var ami string //TODO: Talk with Fred

	vmName := builder.generateVmName(competition, team, host, build)
	vpcID := entProNetwork.HCLProvisionedNetworkToNetwork.Vars["VpcId"]
	secGroupID := entProNetwork.HCLProvisionedNetworkToNetwork.Vars["SecGroupId"]

	input := &ec2.RunInstancesInput{
		ImageId:          aws.String(ami),
		InstanceType:     instanceType,
		MinCount:         &numInstances,
		MaxCount:         &numInstances,
		SecurityGroupIds: []string{secGroupID},
		ClientToken:      &vmName,
		PrivateIpAddress: &provisionedHost.SubnetIP,
		SubnetId:         &vpcID,
		//KeyName: ,
	}

	// Deploy Host
	result, err := client.RunInstances(ctx, input)
	if err != nil {
		return err
	}
	// Get InstanceId and store it in ENT to access later
	id := *result.Instances[0].InstanceId
	newVars := provisionedHost.HCLProvisionedHostToHost.Vars
	newVars["InstanceId"] = id
	err = host.Update().SetVars(newVars).Exec(ctx)
	if err != nil {
		return err
	}
	fmt.Println("Created tagged instance with ID " + id)

	return

}

func (builder AWSBuilder) DeployNetwork(ctx context.Context, provisionedNetwork *ent.ProvisionedNetwork) (err error) {

	// Get information about Network from ENT
	entNetwork, err := provisionedNetwork.QueryProvisionedNetworkToNetwork().Only(ctx)
	if err != nil {
		return fmt.Errorf("couldn't query build from network \"%s\": %v", provisionedNetwork.Name, err)
	}

	//https://aws.github.io/aws-sdk-go-v2/docs/configuring-sdk/#specifying-credentials
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(builder.Region))
	if err != nil {
		return err
	}

	// Describe the network with info from above and get ready to deploy
	client := ec2.NewFromConfig(cfg)

	teamName, err := provisionedNetwork.QueryProvisionedNetworkToTeam().Only(ctx)
	desc := "Team " + teamName.String() + "s Security Group"
	vpcID := entNetwork.Vars["VpcId"]

	input := &ec2.CreateSecurityGroupInput{
		Description: &desc,
		GroupName:   &desc,
		VpcId:       &vpcID,
	}

	// Deploy Network
	results, err := client.CreateSecurityGroup(ctx, input)
	if err != nil {
		return err
	}
	id := *results.GroupId
	newVars := entNetwork.Vars
	newVars["SecGroupId"] = id
	err = entNetwork.Update().SetVars(newVars).Exec(ctx)
	if err != nil {
		return err
	}
	return
}

func (builder AWSBuilder) TeardownHost(ctx context.Context, provisionedHost *ent.ProvisionedHost) (err error) {
	host, err := provisionedHost.QueryProvisionedHostToHost().Only(ctx)

	//https://aws.github.io/aws-sdk-go-v2/docs/configuring-sdk/#specifying-credentials
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(builder.Region))
	if err != nil {
		return err
	}
	client := ec2.NewFromConfig(cfg)

	var instances []string
	instances[0] = host.Vars["InstanceId"]
	input := &ec2.TerminateInstancesInput{
		InstanceIds: instances,
	}
	results, err := client.TerminateInstances(ctx, input)
	if err != nil {
		return err
	}
	_ = results
	return
}

func (builder AWSBuilder) TeardownNetwork(ctx context.Context, provisionedNetwork *ent.ProvisionedNetwork) (err error) {
	entNetwork, err := provisionedNetwork.QueryProvisionedNetworkToNetwork().Only(ctx)

	//https://aws.github.io/aws-sdk-go-v2/docs/configuring-sdk/#specifying-credentials
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(builder.Region))
	if err != nil {
		return err
	}
	secGroupID := entNetwork.Vars["SecGroupId"]
	client := ec2.NewFromConfig(cfg)
	input := &ec2.DeleteSecurityGroupInput{
		GroupId: &secGroupID,
	}
	results, err := client.DeleteSecurityGroup(ctx, input)
	if err != nil {
		return err
	}
	_ = results
	return
}

func (builder AWSBuilder) DeployTeam(ctx context.Context, entTeam *ent.Team) (err error) {
	provisionedNetwork, err := entTeam.QueryTeamToProvisionedNetwork().Only(ctx)
	entNetwork, err := provisionedNetwork.QueryProvisionedNetworkToNetwork().Only(ctx)
	if err != nil {
		return fmt.Errorf("couldn't query build from network \"%s\": %v", provisionedNetwork.Name, err)
	}

	//https://aws.github.io/aws-sdk-go-v2/docs/configuring-sdk/#specifying-credentials
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(builder.Region))
	if err != nil {
		return err
	}

	// Describe the network with info from above and get ready to deploy
	client := ec2.NewFromConfig(cfg)
	input := &ec2.CreateVpcInput{
		CidrBlock: &entNetwork.Cidr,
	}

	// Deploy Network
	results, err := client.CreateVpc(ctx, input)
	if err != nil {
		return err
	}
	id := *results.Vpc.VpcId

	subnetInput := &ec2.CreateSubnetInput{
		VpcId:     &id,
		CidrBlock: &entNetwork.Cidr,
	}
	result, err := client.CreateSubnet(ctx, subnetInput)
	if err != nil {
		return err
	}
	subnetID := *result.Subnet.SubnetId

	newVars := entNetwork.Vars
	newVars["VpcId"] = id
	newVars["SubnetID"] = subnetID
	err = entNetwork.Update().SetVars(newVars).Exec(ctx)
	if err != nil {
		return err
	}
	fmt.Println(results)
	return
}
func (builder AWSBuilder) TeardownTeam(ctx context.Context, entTeam *ent.Team) (err error) {
	provisionedNetwork, err := entTeam.QueryTeamToProvisionedNetwork().Only(ctx)
	entNetwork, err := provisionedNetwork.QueryProvisionedNetworkToNetwork().Only(ctx)

	//https://aws.github.io/aws-sdk-go-v2/docs/configuring-sdk/#specifying-credentials
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(builder.Region))
	if err != nil {
		return err
	}
	vpcID := entNetwork.Vars["VpcId"]
	subnetID := entNetwork.Vars["SubnetID"]
	client := ec2.NewFromConfig(cfg)
	subnetInput := &ec2.DeleteSubnetInput{
		SubnetId: &subnetID,
	}
	subnetResults, err := client.DeleteSubnet(ctx, subnetInput)
	if err != nil {
		println(err.Error())
		return err
	}
	input := &ec2.DeleteVpcInput{
		VpcId: &vpcID,
	}
	results, err := client.DeleteVpc(ctx, input)
	if err != nil {
		return err
	}
	_ = results
	_ = subnetResults
	return
}
