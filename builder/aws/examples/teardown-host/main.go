package main

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

const (
	AWS_Access_Key_Id     = "SECRET"
	AWS_Secret_Access_Key = "SECRET"
	AWS_Session_Token     = "SECRET"
	Region                = "us-east1"
	Instance_ID           = "INSTANCE_ID"
)

func main() {
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(AWS_Access_Key_Id, AWS_Secret_Access_Key, AWS_Session_Token)), config.WithRegion(Region))
	if err != nil {
		return
	}
	client := ec2.NewFromConfig(cfg)

	var instances []string
	instances[0] = Instance_ID
	input := &ec2.TerminateInstancesInput{
		InstanceIds: instances,
	}
	results, err := client.TerminateInstances(ctx, input)
	if err != nil {
		return
	}
	println(results)
}
