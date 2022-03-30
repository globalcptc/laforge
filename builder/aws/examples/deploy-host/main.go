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
var ipAddress string = "10.0.0.6"
var secGroupID = "sg-0823517c8484680f9"
var subnetId string = "subnet-02c200476ee4d77f9"

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

	input := &ec2.RunInstancesInput{
		ImageId:          aws.String(AMI),
		InstanceType:     instanceType,
		MinCount:         &numInstances,
		MaxCount:         &numInstances,
		SecurityGroupIds: []string{secGroupID},
		ClientToken:      &vmName,
		PrivateIpAddress: &ipAddress,
		SubnetId:         &subnetId,
	}
	client := ec2.NewFromConfig(cfg)

	result, err := client.RunInstances(ctx, input)
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
