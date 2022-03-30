package main

import (
	"context"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

var Region = "us-east-1"
var Security_Group_Id = "sg-0823517c8484680f9"
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
	secGroupID := Security_Group_Id
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
	input := &ec2.DeleteSecurityGroupInput{
		GroupId: &secGroupID,
	}
	results, err := client.DeleteSecurityGroup(ctx, input)
	_ = results
	if err != nil {
		println(err.Error())
		os.Exit(1)
	}
	TeardownTeam()
	println("Security Group " + secGroupID + " deleted successfully.")
}
