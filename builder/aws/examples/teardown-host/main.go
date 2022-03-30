package main

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

const (
	Region      = "us-east-1"
	Instance_ID = "i-06ee0742824c4b92e"
)

func main() {
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(Region))
	if err != nil {
		println(err.Error())
		return
	}
	client := ec2.NewFromConfig(cfg)

	var instances []string
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
}
