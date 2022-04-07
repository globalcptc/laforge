package main

import (
	"context"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

var Region = "us-east-1"
var vpcID = "vpc-0f84f412513823015"
var subnetId = "subnet-02c200476ee4d77f9"

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
	results, err := client.DeleteVpc(ctx, input)
	_ = results
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

	client := ec2.NewFromConfig(cfg)
	subnetInput := &ec2.DeleteSubnetInput{
		SubnetId: &subnetId,
	}
	result, err := client.DeleteSubnet(ctx, subnetInput)
	_ = result
	if err != nil {
		println(err.Error())
		os.Exit(1)
	}
	println("Subnet " + subnetId + " deleted successfully.")
	TeardownTeam()
}
