package main

import (
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/gobuffalo/packr"
	"github.com/google/uuid"
	"log"
	"time"
)

func resource(resources packr.Box, name string) map[string]interface{} {
	var result map[string]interface{}
	dum := resources.Bytes(name)
	err := json.Unmarshal(dum, &result)
	if err != nil {
		panic(err)
	}
	return result
}

func dummyResource(resources packr.Box) map[string]interface{} {
	return resource(resources, "dummy-resource.json")
}

func hyperdriveTemplate(resources packr.Box) map[string]interface{} {
	return resource(resources, "hyperdrive-template.json")
}

type KeyVal = struct {
	Key, Val string
}

func makeDummyCFT(resources packr.Box, cfs *cloudformation.CloudFormation, stackName string, outs ...KeyVal) error {
	cft, err := dummyCFT(resources, outs...)
	if err != nil {
		return err
	}
	return deployCFT(cfs, stackName, cft)
}

func dummyCFT(resources packr.Box, outs ...KeyVal) (map[string]interface{}, error) {
	cft := dummyResource(resources)
	cft["Description"] = "A template to access the default VPC resources as if they were created by a CF template"
	out := make(map[string]interface{}, len(outs))
	for _, el := range outs {
		accOutput(out, el.Key, el.Val)
	}
	cft["Outputs"] = out
	return cft, nil
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

func deployCFT(cfs *cloudformation.CloudFormation, stackName string, template map[string]interface{}, keyvals ...KeyVal) error {
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
	if len(keyvals) > 0 {
		parameters := make([]cloudformation.Parameter, len(keyvals))
		for i, el := range keyvals {
			parameters[i] = cloudformation.Parameter{
				ParameterKey:   &el.Key,
				ParameterValue: &el.Val,
			}
		}
		input.Parameters = parameters
	}
	cs, err := cfs.CreateChangeSetRequest(&input).Send()
	if err != nil {
		return err
	}
	return executeChangeset(cfs, cs)
}

func executeChangeset(cfs *cloudformation.CloudFormation, cs *cloudformation.CreateChangeSetOutput) error {
	waitForChangeSet(cfs, cs)
	_, err := cfs.ExecuteChangeSetRequest(&cloudformation.ExecuteChangeSetInput{
		ChangeSetName: cs.Id,
	}).Send()
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
