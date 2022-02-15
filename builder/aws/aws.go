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

func (builder AWSBuilder) generateRouterName(competition *ent.Competition, team *ent.Team, build *ent.Build) string {
	return (competition.HclID + "-Team-" + fmt.Sprintf("%02d", team.TeamNumber) + "-" + builder.generateBuildID(build))
}

func (builder AWSBuilder) generateNetworkName(competition *ent.Competition, team *ent.Team, network *ent.Network, build *ent.Build) string {
	return (competition.HclID + "-Team-" + fmt.Sprintf("%02d", team.TeamNumber) + "-" + network.Name + "-" + builder.generateBuildID(build))
}

func (builder AWSBuilder) DeployHost(ctx context.Context, provisionedHost *ent.ProvisionedHost) (err error) {

	// Get information about host from ENT
	host, err := provisionedHost.QueryProvisionedHostToHost().Only(ctx)
	if err != nil {
		return err
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(builder.Region))
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
	var ami string
	switch host.OS {
	case "ubuntu":
		ami = ""
	}
	vmName := builder.generateVmName(competition, team, host, build)
	//Determine the Team so I can determine the VPC to deploy to
	var vpc []string //TODO: Find out how to get the VPC ID

	input := &ec2.RunInstancesInput{
		ImageId:          aws.String(ami),
		InstanceType:     instanceType,
		MinCount:         &numInstances,
		MaxCount:         &numInstances,
		SecurityGroupIds: vpc,
		ClientToken:      &vmName,
		//PrivateIpAddress: ,
		//SecurityGroups: ,
		//SecurityGroupIds: ,
		//SubnetId: ,
		//NetworkInterfaces: ,
		//KeyName: ,
	}

	// Deploy Host
	result, err := client.RunInstances(ctx, input)
	if err != nil {
		return err
	}

	id := *result.Instances[0].InstanceId
	fmt.Println("Created tagged instance with ID " + id)

	return

}

func (builder AWSBuilder) DeployNetwork(ctx context.Context, provisionedNetwork *ent.ProvisionedNetwork) (err error) {

	// Get information about Network from ENT
	entNetwork, err := provisionedNetwork.QueryProvisionedNetworkToNetwork().Only(ctx)
	if err != nil {
		return fmt.Errorf("couldn't query build from network \"%s\": %v", provisionedNetwork.Name, err)
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(builder.Region))
	if err != nil {
		return err
	}

	// Describe the network with info from above and get ready to deploy
	client := ec2.NewFromConfig(cfg)
	input := &ec2.CreateVpcInput{
		CidrBlock: aws.String(entNetwork.Cidr),
	}

	// Deploy Network
	results, err := client.CreateVpc(ctx, input)
	if err != nil {
		return err
	}
	fmt.Println(results)
	return
}

func (builder AWSBuilder) TeardownHost(ctx context.Context, provisionedHost *ent.ProvisionedHost) (err error) {
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

	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(builder.Region))
	if err != nil {
		return err
	}
	client := ec2.NewFromConfig(cfg)

	vmName := builder.generateVmName(competition, team, host, build)
	var instances []string
	instances[0] = vmName
	input := &ec2.TerminateInstancesInput{
		InstanceIds: instances,
	}
	results, err := client.TerminateInstances(ctx, input)
	if err != nil {
		return err
	}
	fmt.Println(results)
	return
}

func (builder AWSBuilder) TeardownNetwork(ctx context.Context, provisionedHost *ent.ProvisionedHost) (err error) {
	return
}
