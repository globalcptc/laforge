package main

import (
	"context"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

var Region = "us-east-1"
var Security_Group_Id = ""
var vpcID = ""

func TeardownTeam() {
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(Region))
	if err != nil {
		println(err.Error())
		os.Exit(1)
	}
	vpcID := vpcID
	client := ec2.NewFromConfig(cfg)
	input := &ec2.DeleteVpcInput{
		VpcId: &vpcID,
	}
	result, err := client.DeleteVpc(ctx, input)
	if err != nil {
		println(err.Error())
		os.Exit(1)
	}
	println("VPC " + vpcID + " deleted successfully.")
	return
}

func main() {
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(Region))
	if err != nil {
		println(err.Error())
		os.Exit(1)
	}
	secGroupID := Security_Group_Id
	client := ec2.NewFromConfig(cfg)
	input := &ec2.DeleteSecurityGroupInput{
		GroupId: &secGroupID,
	}
	results, err := client.DeleteSecurityGroup(ctx, input)
	if err != nil {
		println(err.Error())
		os.Exit(1)
	}
	TeardownTeam()
	print("Security Group " + secGroupID + " deleted successfully.")
}
