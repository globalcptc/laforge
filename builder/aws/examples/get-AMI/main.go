package main

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/davecgh/go-spew/spew"
)

var Region string = "us-east-1"
var name string = "ubuntu/images/hvm-ssd/ubuntu-focal-20.04-amd64-server-*"
var vt string = "hvm"
var rdt string = "ebs"
var arch string = "x86_64"
var owner string = "099720109477"

func main() {
	ctx := context.Background()
	// https://aws.github.io/aws-sdk-go-v2/docs/configuring-sdk/#specifying-credentials
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithSharedCredentialsFiles([]string{"../../../../configs/config"}),
		config.WithRegion(Region))
	if err != nil {
		println(err.Error())
		return
	}
	client := ec2.NewFromConfig(cfg)
	// Describe the host with info from above and get ready to deploy
	input := ec2.DescribeImagesInput{
		DryRun:          aws.Bool(false),
		ExecutableUsers: []string{"all"},
		Filters: []types.Filter{
			{Name: aws.String("name"), Values: []string{name}},
			{Name: aws.String("root-device-type"), Values: []string{rdt}},
			{Name: aws.String("virtualization-type"), Values: []string{vt}},
			{Name: aws.String("architecture"), Values: []string{arch}},
		},
		ImageIds:          []string{},
		IncludeDeprecated: aws.Bool(false),
		Owners:            []string{owner},
	}

	spew.Dump(input)
	output, err := client.DescribeImages(ctx, &input)
	spew.Dump(output)
	spew.Dump(err)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	if len(output.Images) > 0 {
		image := output.Images[0]
		fmt.Println("Image ID: " + *image.ImageId)
		return
	} else {
		fmt.Println("No image found")
		return
	}

}
