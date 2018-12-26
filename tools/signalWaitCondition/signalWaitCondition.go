package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/pkg/errors"
	"net/http"
	"os"
	"regexp"
)

var cf *cloudformation.CloudFormation

var chReg = regexp.MustCompile(".*/(.*)")

func main() {
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		panic(err)
	}
	cf = cloudformation.New(cfg)
	argsLen := len(os.Args)
	stackName := os.Args[argsLen-3]
	conditionName := os.Args[argsLen-2]
	status := os.Args[argsLen-1]
	stacks, err := cf.DescribeStacksRequest(&cloudformation.DescribeStacksInput{
		StackName: &stackName,
	}).Send()
	if err != nil {
		panic(err)
	}
	stack := stacks.Stacks[0]
	condition, err := findResourcePhysicalId(stack, conditionName)
	if err != nil {
		panic(err)
	}
	submatches := chReg.FindStringSubmatch(condition)
	conditionHandle, err := findResourcePhysicalId(stack, submatches[1])
	data := map[string]string{
		"Data":     "",
		"Reason":   "By Hand with signalWaitCondition",
		"Status":   status,
		"UniqueId": "1",
	}
	msg, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}
	request, err := http.NewRequest("PUT", conditionHandle, bytes.NewReader(msg))
	if err != nil {
		panic(err)
	}
	request.Header.Add("Content-Type", "")
	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		panic(err)
	}
	fmt.Println(resp.Status)
}

func findResourcePhysicalId(stack cloudformation.Stack, resource string) (string, error) {
	res, err := cf.DescribeStackResourceRequest(&cloudformation.DescribeStackResourceInput{
		StackName:         stack.StackId,
		LogicalResourceId: &resource,
	}).Send()
	if err != nil {
		return "", errors.Wrapf(err, "could not describe the resource %s in the stack %s", resource, *stack.StackId)
	}
	return *res.StackResourceDetail.PhysicalResourceId, nil
}
