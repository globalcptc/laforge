package main

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

const (
	Region       = "us-east-1"
	AMI          = "ami-04505e74c0741db8d"
	InstanceSize = "nano"
)

var numInstances int32 = 1
var instanceType types.InstanceType
var vmName string = "Test Ubuntu VM"
var ipAddress string = "10.0.0.1"
var secGroupID = ""
var subnetId string = ""
var vpcID string = ""

type EC2CreateInstanceAPI interface {
	RunInstances(ctx context.Context,
		params *ec2.RunInstancesInput,
		optFns ...func(*ec2.Options)) (*ec2.RunInstancesOutput, error)
}

func main() {
	ctx := context.Background()
	// https://aws.github.io/aws-sdk-go-v2/docs/configuring-sdk/#specifying-credentials
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(Region))
	if err != nil {
		println(err.Error())
		return
	}
	client := ec2.NewFromConfig(cfg)
	desc := "Example Security Group"
	input := &ec2.CreateSecurityGroupInput{
		Description: &desc,
		GroupName:   &desc,
		VpcId:       &vpcID,
	}

	// Deploy Network
	results, err := client.CreateSecurityGroup(ctx, input)
	if err != nil {
		println(err.Error())
		return
	}
	println("Security Group ID: " + *results.GroupId)

	switch InstanceSize {
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

	hostInput := &ec2.RunInstancesInput{
		ImageId:          aws.String(AMI),
		InstanceType:     instanceType,
		MinCount:         &numInstances,
		MaxCount:         &numInstances,
		SecurityGroupIds: []string{secGroupID},
		ClientToken:      &vmName,
		PrivateIpAddress: &ipAddress,
		SubnetId:         &subnetId,
	}

	result, err := client.RunInstances(ctx, hostInput)
	if err != nil {
		println(err.Error())
		return
	}
	//id := *result.Instances[0].InstanceId
	if err != nil {
		println(err.Error())
		return
	}
	println("Instance ID: " + *result.Instances[0].InstanceId)
}
