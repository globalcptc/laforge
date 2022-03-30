package main

import (
	"context"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

var cidr = "10.0.0.0/24"
var Region = "us-east-1"

func deployTeam() string {
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(Region))
	if err != nil {
		println(err.Error())
		return string(err.Error())
	}

	// Describe the network with info from above and get ready to deploy
	client := ec2.NewFromConfig(cfg)
	input := &ec2.CreateVpcInput{
		CidrBlock: &cidr,
	}

	// Deploy Network
	results, err := client.CreateVpc(ctx, input)
	if err != nil {
		println(err.Error())
		os.Exit(1)
	}
	id := *results.Vpc.VpcId
	return id
}

func main() {
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(Region))
	if err != nil {
		println(err.Error())
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
		println(err.Error())
		return
	}
	println(results.GroupId)
	println("VPC ID: " + vpcID)
	return
}
