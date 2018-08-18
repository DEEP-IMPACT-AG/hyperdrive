// # Build/release program for the HyperdriveCore
//
// This program is used to build and deploy the HyperdriveCore of the
// [hyperdrive](https://hyperdrive.sh) project.
//
// The core consists of the following parts:
//
// 1. Lambda functions to create custom resources.
// 2. Cloudformation template to install those lambda functions
// 3. A Cloudformation template to install a cloudfront folder rewriting
//    lambda functions (only for in the us-east-1 due to restrictions on
//    Lambda@Edge).
//
// The main difficulty is to publish AWS Lambda code artifacts and
// Cloudformation templates: due to restrictions of Cloudformation when
// creating lambda functions in a stack, packaged code for lambda functions
// must be placed on a bucket that is in the same region as the stack
// containing them. Consequently, we have 1 S3 bucket per region
// `<region-name>.hyperdrive.sh` and copy artifacts on all the buckets.
//
package main

import (
	"fmt"
	"github.com/DEEP-IMPACT-AG/hyperdrive/make/build"
	"github.com/DEEP-IMPACT-AG/hyperdrive/make/release"
	"github.com/DEEP-IMPACT-AG/hyperdrive/make/test"
	"log"
	"os"
)

// ## Main function
//
// The main function is the entry point. The make program is meant to be
// called from the shell and expects one argument which can be one of the
// following:
//
// - `build` : in this case, the program will build all the lambda
//   functions defined above and package them, namely zip them, as
//   artifacts for a lambda cloudformation resource.
// - `release`: in this case, the program will upload artifacts built via
//   the previous command to all hyperdrive s3 buckets; also, for each
//   region, it will generate cloudformation templates for the installation
//   of core; then, it will create a special us-east-1 only template to
//   install the cloudfront redirect lambda@edge function; finally, the
//   releasing should be done on a commit with a release tag;
// - `test-release`: this case is used to deploy a test version to the
//   eu-west-1 and us-east-1 regions.
// - `version`: prints out the version that would be released (or
//   "test-released").
//
func main() {
	var err error
	switch os.Args[len(os.Args)-1] {
	case "build":
		err = build.Build()
	case "release":
		err = release.Release()
	case "test-release":
		err = release.TestRelease()
	case "integration-test":
		test.IntegrationTest()
	case "version":
		var version string
		version, err = release.HyperdriveVersion()
		if err == nil {
			fmt.Println("Version:", version)
		}
	}
	if err != nil {
		log.Fatalf("fatal error: %+v\n", err)
	}
}
