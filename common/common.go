package common

import (
	"github.com/aws/aws-lambda-go/cfn"
	"regexp"
	"strings"
)

// The most important data in the error response to cloudformation is the
// `PhysicalResourceId` that is stored by cloudformation to identify the
// resources that have been created. The `PhysicalResourceId` must always
// be created even in case of creation failure as cloudformation generates
// a random one if non is supplied. If a resource needs many API calls to
// be completely generated, it is very important to send the correct
// `PhysicalResourceId` even in case of failure as cloudformation will send
// a delete event when the stack is deleted respectively rollbacked.
//
// It is important to create a failure id for graceful NOP when deleting a
// resource that has failed to create. To gracefully handle these cases, we
// define the following helper functions.
func FailurePhysicalResourceId(event cfn.Event) string {
	return "failure-" + event.LogicalResourceID
}

func IsFailurePhysicalResourceId(id string) bool {
	return strings.HasPrefix(id, "failure-")
}

// Some of the custom resources are created in a different region that the
// cloudformation stack. Since the region propery is optional, we need to
// use the arn of an existing resource to detect if a change of region has
// happen: the region could have been undefined previously and gets
// suddenly defined or the other way around.
var regionExtractor = regexp.MustCompile("arn:aws:(?:.*?):(.*?):")

func ArnRegion(arn string) string {
	return regionExtractor.FindStringSubmatch(arn)[1]
}

func IsSameRegion(event cfn.Event, oldRegion, region string) bool {
	// 1. if the new region is the same as the old one, they are the same.
	if oldRegion == region {
		return true
	}
	// 2. else, if both are defined, they are not the same.
	if len(oldRegion) > 0 && len(region) > 0 {
		return false
	}
	// 3. else, we have a complicate case where either the old or the new region are implicit from the
	// region of the cloudformation stack.
	sdkRegion := ArnRegion(event.StackID)
	if sdkRegion == oldRegion || sdkRegion == region {
		return true
	}
	return false
}

