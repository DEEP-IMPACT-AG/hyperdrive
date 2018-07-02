package main

import (
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"encoding/json"
	"log"
	"github.com/google/uuid"
	"fmt"
	"time"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
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

func DefaultVpcCF(ec2s *ec2.EC2) map[string]interface{} {
	cf := dummyResource()
	cf["Description"] = "A template to access the default VPC resources as if they were created by a CF template"
	out := make(map[string]interface{})
	accOutput(out, "VpcId", *DefaultVpc(ec2s).VpcId)
	cf["Outputs"] = out
	return cf
}

func MakeDefaultVpcCF(ec2s *ec2.EC2, cfs *cloudformation.CloudFormation) {
	cf := DefaultVpcCF(ec2s)
	cfeb, err := json.Marshal(cf)
	if err != nil {
		log.Fatal(err)
	}
	cfes := string(cfeb)
	stackName := "HyperdriveDefaultVPC"
	postFix, err := uuid.NewRandom()
	if err != nil {
		log.Fatal(err)
	}
	csName := stackName + postFix.String()
	input := cloudformation.CreateChangeSetInput{
		ChangeSetName: &csName,
		ChangeSetType: "CREATE",
		TemplateBody:  &cfes,
		StackName:     &stackName,
	}
	cs, err := cfs.CreateChangeSetRequest(&input).Send()
	in2 := cloudformation.ExecuteChangeSetInput{
		ChangeSetName: cs.Id,
	}
	if err != nil {
		log.Fatal(err)
	}
	waitForChangeSet(cfs, cs)
	_, err = cfs.ExecuteChangeSetRequest(&in2).Send()
	if err != nil {
		log.Fatal(err)
	}
}

func waitForChangeSet(cfs *cloudformation.CloudFormation, cs *cloudformation.CreateChangeSetOutput) {
	for i := 0; i < 10; i++ {
		request := cloudformation.DescribeChangeSetInput{
			ChangeSetName: cs.Id,
		}
		res, err := cfs.DescribeChangeSetRequest(&request).Send()
		if err != nil {
			log.Fatal(err)
		}
		if res.ExecutionStatus == "AVAILABLE" {
			return
		}
		fmt.Printf("Loop %v\n", i)
		time.Sleep(500 * time.Millisecond)
	}
	panic("CS not ready!")
}
