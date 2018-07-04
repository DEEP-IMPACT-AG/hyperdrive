package main

import (
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"log"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/gobuffalo/packr"
	"strings"
	"fmt"
)

var DefaultVPCStackName = "DefaultVPC"
var notExistsError = fmt.Sprintf("Stack with id %s does not exist", DefaultVPCStackName)

func DefaultVpc(ec2s *ec2.EC2) ec2.Vpc {
	filterName := "isDefault"
	request := ec2.DescribeVpcsInput{
		Filters: []ec2.Filter{
			{
				Name:   &filterName,
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

func DefaultVpcCFSExist(cfs *cloudformation.CloudFormation) (bool, error) {
	_, err := cfs.DescribeStacksRequest(&cloudformation.DescribeStacksInput{
		StackName: &DefaultVPCStackName,
	}).Send();
	if err != nil {
		if strings.Contains(err.Error(), notExistsError) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func MakeDefaultVpcCF(resources packr.Box, ec2s *ec2.EC2, cfs *cloudformation.CloudFormation) error {
	return makeDummyCFT(resources, cfs, DefaultVPCStackName, KeyVal{Key: "VpcId", Val: *DefaultVpc(ec2s).VpcId})
}
