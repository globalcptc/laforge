package main

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

const (
	AWS_Access_Key_Id     = "SECRET"
	AWS_Secret_Access_Key = "SECRET"
	AWS_Session_Token     = "SECRET"
	Region                = "us-east1"
	Security_Group_Id     = "Security_Group_Id"
	vmName                = "VMNAME"
	VpcId                 = "VPC_ID"
	AMI                   = "AMI"
	ipAddress             = "IP"
	secGroupID            = "SEC_GROUP_ID"
	vpcID                 = "VPC_ID"
	InstanceSize          = "nano"
)

type EC2CreateInstanceAPI interface {
	RunInstances(ctx context.Context,
		params *ec2.RunInstancesInput,
		optFns ...func(*ec2.Options)) (*ec2.RunInstancesOutput, error)
}

func main() {
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(AWS_Access_Key_Id, AWS_Secret_Access_Key, AWS_Session_Token)), config.WithRegion(Region))
	var numInstances int32 = 1
	var instanceType types.InstanceType
	var vmName string = vmName
	var ipAddress string = ipAddress
	var vpcID string = vpcID

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
		SubnetId:         &vpcID,
	}
	client := ec2.NewFromConfig(cfg)

	result, err := client.RunInstances(ctx, input)
	if err != nil {
		return
	}
	//id := *result.Instances[0].InstanceId
	if err != nil {
		return
	}
	println(result)

	return
}
