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
}

// EC2CreateInstanceAPI defines the interface for the RunInstances function.
type EC2CreateInstanceAPI interface {
	RunInstances(ctx context.Context,
		params *ec2.RunInstancesInput,
		optFns ...func(*ec2.Options)) (*ec2.RunInstancesOutput, error)
}

func (builder AWSBuilder) DeployHost(ctx context.Context, provisionedHost *ent.ProvisionedHost) (err error) {

	host, err := provisionedHost.QueryProvisionedHostToHost().Only(ctx)
	if err != nil {
		return err
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("us-east-2"))
	if err != nil {
		return err
	}

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

	//Determine the Team so I can determine the VPC to deploy to
	var vpc []string

	input := &ec2.RunInstancesInput{
		ImageId:          aws.String(ami),
		InstanceType:     instanceType,
		MinCount:         &numInstances,
		MaxCount:         &numInstances,
		SecurityGroupIds: vpc,
	}

	result, err := client.RunInstances(ctx, input)
	if err != nil {
		return err
	}
	id := *result.Instances[0].InstanceId
	fmt.Println("Created tagged instance with ID " + id)

	return

}

func (builder AWSBuilder) DeployNetwork(ctx context.Context, provisionedNetwork *ent.ProvisionedNetwork) (err error) {
	entBuild, err := provisionedNetwork.QueryProvisionedNetworkToBuild().Only(ctx)
	if err != nil {
		return fmt.Errorf("couldn't query build from network \"%s\": %v", provisionedNetwork.Name, err)
	}
	entEnvironment, err := provisionedNetwork.QueryProvisionedNetworkToBuild().QueryBuildToEnvironment().Only(ctx)
	if err != nil {
		return fmt.Errorf("couldn't query build from network \"%s\": %v", provisionedNetwork.Name, err)
	}
	entCompetition, err := entEnvironment.EnvironmentToCompetition(ctx)
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

	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("us-east-2"))
	if err != nil {
		return err
	}

	client := ec2.NewFromConfig(cfg)
	input := &ec2.CreateVpcInput{
		CidrBlock: aws.String(entNetwork.Cidr),
	}
	results, err := client.CreateVpc(ctx, input)
	if err != nil {
		return err
	}
	fmt.Println(results)
	return
}

func (builder AWSBuilder) TeardownHost(ctx context.Context, provisionedHost *ent.ProvisionedHost) (err error) {
	return
}

func (builder AWSBuilder) TeardownNetwork(ctx context.Context, provisionedHost *ent.ProvisionedHost) (err error) {
	return
}
