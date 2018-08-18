package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"math/rand"
)

const hyperdriveCoreVersion = "v0.0.0-52-g1c0b"

func InstallHyperdrive() {
	stackName := "HyperdriveCore"
	for _, region := range []string{"eu-central-1"} {
		cfg, err := external.LoadDefaultAWSConfig(
			external.WithRegion(region),
		)
		if err != nil {
			panic(err)
		}
		cfs := cloudformation.New(cfg)
		csn := fmt.Sprintf("hyperdrive-%d", rand.Int())
		key := "hyperdrive-origin"
		value := "hyperdrive-core"
		template := fmt.Sprintf("https://s3-%[1]s.amazonaws.com/%[1]s.hyperdrive.sh/cf/hyperdrive/%[2]s/hyperdriveCore.yaml", region, hyperdriveCoreVersion)
		cs, err := cfs.CreateChangeSetRequest(&cloudformation.CreateChangeSetInput{
			Capabilities:  []cloudformation.Capability{cloudformation.CapabilityCapabilityIam},
			ChangeSetName: &csn,
			ChangeSetType: cloudformation.ChangeSetTypeCreate,
			StackName:     &stackName,
			Tags:          []cloudformation.Tag{{Key: &key, Value: &value}},
			TemplateURL:   &template,
		}).Send()
		if err != nil {
			panic(err)
		}
		if err = executeChangeset(cfs, cs); err != nil {
			panic(err)
		}
	}
}
