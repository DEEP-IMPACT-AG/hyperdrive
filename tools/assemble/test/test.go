// # Integration Test
//
// This section contains code to test the different custom resources. It
// can take some time to run as resources on AWS have to stabilise even
// after a cloudformation stack has been created. The test always proceeds
// by installing the hyperdrive core template in the eu-west-1 zone and
// creates/updates/deletes different stacks. At the moment, it can only be
// run manually with the `test-hyperdriveCore.yaml` template
package test

import (
	"fmt"
	"github.com/DEEP-IMPACT-AG/hyperdrive/make/test/cf"
	"github.com/aws/aws-sdk-go-v2/aws/awserr"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/pkg/errors"
	"log"
)

var hyperdriveCoreTestStackName = "HyperdriveCoreTest"

const version = "v0.0.0-52-g38a3"

func IntegrationTest() {
	err := RemoveHyperdrive()
	if err != nil {
		log.Fatal("could not remove the hyperdrive", err)
	}
	err = InstallHyperdrive(version)
	if err != nil {
		log.Fatalf("could not install the hyperdrive: %+v\n", err)
	}
	/*
	err = loggrp.TestLogGroup()
	if err != nil {
		log.Printf("error during log group test: %+v\n", err)
	}
	err = dnscert.TestAcmCertificate()
	if err != nil {
		log.Printf("error during acm test: %+v\n", err)
	}*/

}

func InstallHyperdrive(hyperdriveCoreVersion string) error {
	// 1. we start by creating the hyperdrive test stacks.
	log.Printf("Installing hyperdrive core version %s\n", hyperdriveCoreVersion)
	stackMap := make(map[string]string, 2)
	for _, region := range []string{"eu-west-1", "us-east-1"} {
		log.Printf(".. region: %s\n", region)
		cfg, err := external.LoadDefaultAWSConfig(
			external.WithRegion(region),
		)
		if err != nil {
			return err
		}
		cfs := cloudformation.New(cfg)
		csn := cf.GenCSN()
		key := "hyperdrive-origin"
		value := "hyperdrive-core"
		templateUrl := fmt.Sprintf("https://s3.amazonaws.com/%[1]s.test.hyperdrive.sh/cf/hyperdrive/%[2]s/hyperdriveCore.yaml", region, hyperdriveCoreVersion)
		log.Printf(".. templateUrl: %s\n", templateUrl)
		cs, err := cfs.CreateChangeSetRequest(&cloudformation.CreateChangeSetInput{
			Capabilities:  []cloudformation.Capability{cloudformation.CapabilityCapabilityIam},
			ChangeSetName: &csn,
			ChangeSetType: cloudformation.ChangeSetTypeCreate,
			StackName:     &hyperdriveCoreTestStackName,
			Tags:          []cloudformation.Tag{{Key: &key, Value: &value}},
			TemplateURL:   &templateUrl,
		}).Send()
		if err != nil {
			return err
		}
		if err = cf.ExecuteChangeset(cfs, cs); err != nil {
			return err
		}
		log.Printf(".. StackId: %s", *cs.StackId)
		stackMap[region] = *cs.StackId
	}
	// 2. and by waiting for them to stabilise.
	log.Println("Wait for stabilistaion")
	for region, stackId := range stackMap {
		cfg, err := external.LoadDefaultAWSConfig(
			external.WithRegion(region),
		)
		if err != nil {
			return err
		}
		log.Printf(".. region: %s\n", region)
		log.Printf(".. StackId: %s\n", stackId)
		cfs := cloudformation.New(cfg)
		if err = cf.WaitForStableStack(cfs, stackId); err != nil {
			return err
		}
	}
	return nil
}

func RemoveHyperdrive() error {
	// 1. trigger all deletions.
	log.Println("Removing hyperdrive core")
	stackMap := make(map[string]string, 2)
	for _, region := range []string{"eu-west-1", "us-east-1"} {
		cfg, err := external.LoadDefaultAWSConfig(
			external.WithRegion(region),
		)
		if err != nil {
			return err
		}
		cfs := cloudformation.New(cfg)
		stack, err := cf.FetchStack(cfs, hyperdriveCoreTestStackName)
		if err != nil {
			vErr, ok := errors.Cause(err).(awserr.RequestFailure)
			if ok && vErr.StatusCode() == 400 {
				log.Printf("The hyperdrive test stack has not been installed on %s.\n", region)
				continue
			}
			return err
		}
		stackMap[region] = *stack.StackId
		log.Printf(".. region: %s\n", region)
		log.Printf(".. StackId: %s\n", *stack.StackId)
		_, err = cfs.DeleteStackRequest(&cloudformation.DeleteStackInput{
			StackName: stack.StackId,
		}).Send()
		if err != nil {
			return err
		}
	}
	// 2. and by waiting for them to stabilise.
	log.Println("Wait for stabilistaion")
	for region, stackId := range stackMap {
		cfg, err := external.LoadDefaultAWSConfig(
			external.WithRegion(region),
		)
		if err != nil {
			return err
		}
		cfs := cloudformation.New(cfg)
		log.Printf(".. region: %s\n", region)
		log.Printf(".. StackId: %s\n", stackId)
		if err = cf.WaitForStableStack(cfs, stackId); err != nil {
			return err
		}
	}
	return nil
}
