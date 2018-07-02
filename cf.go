package main

import (
	"encoding/json"
	"github.com/google/uuid"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"fmt"
	"time"
	"log"
)

func dummyResource() map[string]interface{} {
	var result map[string]interface{}
	dum := Resources.Bytes("dummy-resource.json")
	json.Unmarshal(dum, &result)
	return result
}

type dummyOutput = struct {
	Key, Val string
}

func makeDummyCFT(cfs *cloudformation.CloudFormation, stackName string, outs ...dummyOutput) error {
	cft := dummyCFT(outs...)
	deployCFT(cfs, stackName, cft)
	return nil
}

func dummyCFT(outs ...dummyOutput) map[string]interface{} {
	cft := dummyResource()
	cft["Description"] = "A template to access the default VPC resources as if they were created by a CF template"
	out := make(map[string]interface{}, len(outs))
	for _, el := range outs {
		accOutput(out, el.Key, el.Val)
	}
	cft["Outputs"] = out
	return cft
}

func accOutput(m map[string]interface{}, key, val string) {
	m[key] = map[string]interface{}{
		"Value": val,
		"Export": map[string]interface{}{
			"Name": map[string]interface{}{
				"Fn::Sub": "${AWS::StackName}-" + key,
			},
		},
	}
}

func deployCFT(cfs *cloudformation.CloudFormation, stackName string, template map[string]interface{}) error {
	cfeb, err := json.Marshal(template)
	if err != nil {
		return err
	}
	cfes := string(cfeb)
	postFix, err := uuid.NewRandom()
	if err != nil {
		return err
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
		return err
	}
	waitForChangeSet(cfs, cs)
	_, err = cfs.ExecuteChangeSetRequest(&in2).Send()
	if err != nil {
		return err
	}
	return nil
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