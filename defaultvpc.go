package main

import (
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"encoding/json"
	"log"
	"github.com/google/uuid"
	"fmt"
	"time"
)

func DefaultVpcCF() map[string]interface{} {
	cf := dummyResource()
	cf["Description"] = "A template to access the default VPC resources as if they were created by a CF template"
	out := make(map[string]interface{})
	accOutput(out, "VpcId", "???")
	cf["Outputs"] = out
	return cf
}

func MakeDefaultVpcCF(cfc *cloudformation.CloudFormation) {
	cf := DefaultVpcCF()
	cfb, err := json.Marshal(cf)
	if err != nil {
		log.Fatal(err)
	}
	cfs := string(cfb)
	stackName := "HyperdriveDefaultVPC"
	postFix, err := uuid.NewRandom()
	if err != nil {
		log.Fatal(err)
	}
	csName := stackName + postFix.String()
	input := cloudformation.CreateChangeSetInput{
		ChangeSetName: &csName,
		ChangeSetType: "CREATE",
		TemplateBody:  &cfs,
		StackName:     &stackName,
	}
	cs, err := cfc.CreateChangeSetRequest(&input).Send()
	in2 := cloudformation.ExecuteChangeSetInput{
		ChangeSetName: cs.Id,
	}
	if err != nil {
		log.Fatal(err)
	}
	waitForChangeSet(cfc, cs)
	_, err = cfc.ExecuteChangeSetRequest(&in2).Send()
	if err != nil {
		log.Fatal(err)
	}
}

func waitForChangeSet(cfc *cloudformation.CloudFormation, cs *cloudformation.CreateChangeSetOutput) {
	for i := 0; i < 10; i++ {
		request := cloudformation.DescribeChangeSetInput{
			ChangeSetName: cs.Id,
		}
		res, err := cfc.DescribeChangeSetRequest(&request).Send()
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
