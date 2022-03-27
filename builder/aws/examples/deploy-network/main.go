package main

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
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
	cidr                  = "10.0.0.0/24"
)

func deployTeam() string {
	var cidr string
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(AWS_Access_Key_Id, AWS_Secret_Access_Key, AWS_Session_Token)), config.WithRegion(Region))
	if err != nil {
		return string(err.Error())
	}

	// Describe the network with info from above and get ready to deploy
	client := ec2.NewFromConfig(cfg)
	input := &ec2.CreateVpcInput{
		CidrBlock: aws.String(cidr),
	}

	// Deploy Network
	results, err := client.CreateVpc(ctx, input)
	if err != nil {
		return string(err.Error())
	}
	id := *results.Vpc.VpcId
	if err != nil {
		return string(err.Error())
	}
	return id
}

func main() {

	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(AWS_Access_Key_Id, AWS_Secret_Access_Key, AWS_Session_Token)), config.WithRegion(Region))
	if err != nil {
		return
	}

	// Describe the network with info from above and get ready to deploy
	client := ec2.NewFromConfig(cfg)
	desc := "Example Security Group"
	vpcID := deployTeam()

	input := &ec2.CreateSecurityGroupInput{
		Description: &desc,
		GroupName:   &desc,
		VpcId:       &vpcID,
	}

	// Deploy Network
	results, err := client.CreateSecurityGroup(ctx, input)
	if err != nil {
		return
	}
	if err != nil {
		return
	}
	println(results)
	return
}
