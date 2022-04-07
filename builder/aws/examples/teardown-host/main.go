package main

import (
	"context"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

const (
	Region = "us-east-1"
)

var instances []string
var secGroupID string = ""
var Instance_ID = ""

func main() {
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(Region))
	if err != nil {
		println(err.Error())
		return
	}
	client := ec2.NewFromConfig(cfg)
	instances = append(instances, Instance_ID)
	input := &ec2.TerminateInstancesInput{
		InstanceIds: instances,
	}
	results, err := client.TerminateInstances(ctx, input)
	if err != nil {
		println(err.Error())
		return
	}
	println("Terminated Instance " + *results.TerminatingInstances[0].InstanceId)

	SecGroupinput := &ec2.DeleteSecurityGroupInput{
		GroupId: &secGroupID,
	}
	result, err := client.DeleteSecurityGroup(ctx, SecGroupinput)
	_ = result
	if err != nil {
		println(err.Error())
		os.Exit(1)
	}
	println("Security Group " + secGroupID + " deleted successfully.")
}
