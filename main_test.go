package main

import (
	"testing"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"log"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
)

func TestMap(t *testing.T) {
	cfg, err := external.LoadDefaultAWSConfig(
		external.WithSharedConfigProfile("libra-dev"),
	)
	if err != nil {
		log.Fatal(err)
	}

	cfs := cloudformation.New(cfg)
	exist, err := DefaultVpcCFSExist(cfs)
	if err != nil {
		t.Fatal(err)
	}
	println(exist)
}
