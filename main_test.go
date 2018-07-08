package main

import (
	"testing"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"log"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
)

func TestMap(t *testing.T) {
	cfg, err := external.LoadDefaultAWSConfig(
		external.WithSharedConfigProfile("hyperdrive"),
	)
	if err != nil {
		log.Fatal(err)
	}

	cfs := cloudformation.New(cfg)
	stackSetName := "hyperdriveS3Buckets"
	res, err := cfs.ListStackInstancesRequest(&cloudformation.ListStackInstancesInput{
		StackSetName: &stackSetName,
	}).Send()
	if err != nil {
		panic(err)
	}
	regions := make([]string, len(res.Summaries))
	for i, sum := range res.Summaries {
		regions[i] = *sum.Region
	}
	fmt.Printf("%v\n", regions)
}
