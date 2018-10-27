package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/gobuffalo/packr"
	"testing"
)

func TestMap(t *testing.T) {
	box := packr.NewBox("./resources")
	fmt.Println(sslCertificateTemplate(box, "first-impact.io", "test3"))
}

func TestHostedZoneName(t *testing.T) {
	fmt.Println(hostedZoneStackName("oort.ch."))
}

func TestDescribeStacks(t *testing.T) {
	cfg, err := external.LoadDefaultAWSConfig(
		external.WithSharedConfigProfile("deepimpact-dev"),
		)
	if err != nil {
		t.Fatal(err)
	}
	cfs := cloudformation.New(cfg)
	stackName := "dummy"
	stacks, err := cfs.DescribeStacksRequest(&cloudformation.DescribeStacksInput{
		StackName: &stackName,
	}).Send()
	if err != nil {
		t.Fatalf("%t, %+v", err, err)
	}
	println(stacks)
}
