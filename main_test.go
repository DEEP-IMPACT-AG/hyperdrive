package main

import (
	"testing"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"log"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

func TestMap(t *testing.T) {
	cfg, err := external.LoadDefaultAWSConfig(
		external.WithSharedConfigProfile("???"),
	)
	if err != nil {
		log.Fatal(err)
	}
	ec2c := ec2.New(cfg)
	filterName := "isDefault"
	request := ec2.DescribeVpcsInput{
		Filters: []ec2.Filter{
			{
				Name: &filterName,
				Values: []string{"true"},
			},
		},
	}
	res, err := ec2c.DescribeVpcsRequest(&request).Send()
	println(*res.Vpcs[0].VpcId)
}
