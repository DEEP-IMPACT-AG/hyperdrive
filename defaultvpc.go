package main

import (
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"log"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/gobuffalo/packr"
)

func DefaultVpc(ec2s *ec2.EC2) ec2.Vpc {
	filterName := "isDefault"
	request := ec2.DescribeVpcsInput{
		Filters: []ec2.Filter{
			{
				Name: &filterName,
				Values: []string{"true"},
			},
		},
	}
	res, err := ec2s.DescribeVpcsRequest(&request).Send()
	if err != nil {
		log.Fatal(err)
	}
	return res.Vpcs[0];
}

func MakeDefaultVpcCF(resources packr.Box, ec2s *ec2.EC2, cfs *cloudformation.CloudFormation) error {
	return makeDummyCFT(resources, cfs, "DefaultVPC", KeyVal{Key: "VpcId", Val: *DefaultVpc(ec2s).VpcId})
}
