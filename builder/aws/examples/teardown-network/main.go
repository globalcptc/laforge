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
	Security_Group_Id     = "Security_Group_Id"
)

func main() {
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(AWS_Access_Key_Id, AWS_Secret_Access_Key, AWS_Session_Token)), config.WithRegion(Region))
	if err != nil {
		return
	}
	secGroupID := Security_Group_Id
	client := ec2.NewFromConfig(cfg)
	input := &ec2.DeleteSecurityGroupInput{
		GroupId: &secGroupID,
	}
	results, err := client.DeleteSecurityGroup(ctx, input)
	if err != nil {
		return
	}
	println(results)
}
