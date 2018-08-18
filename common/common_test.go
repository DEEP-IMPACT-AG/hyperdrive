package common

import (
	"github.com/aws/aws-lambda-go/cfn"
	"log"
	"testing"
)

func TestIsSameRegion(t *testing.T) {
	// 1. Correctness
	for i, test := range []struct {
		event             cfn.Event
		oldRegion, region string
	}{
		// test 0
		{oldRegion: "", region: ""},
		// test 1
		{oldRegion: "us-east-1", region: "us-east-1"},
		// test 2
		{event: cfn.Event{StackID: "arn:aws:cloudformation:us-west-2:123456789012:stack/teststack/51af3dc0-da77-11e4-872e-1234567db123"},
			oldRegion: "", region: "us-west-2"},
		// test 3
		{event: cfn.Event{StackID: "arn:aws:cloudformation:us-west-2:123456789012:stack/teststack/51af3dc0-da77-11e4-872e-1234567db123"},
			oldRegion: "us-west-2", region: ""},
	} {
		if !IsSameRegion(test.event, test.oldRegion, test.region) {
			log.Fatalf("Test case %d, %+v, %s, %s", i, test.event, test.oldRegion, test.region)
		}
	}

	// 2. Completeness
	for i, test := range []struct {
		event             cfn.Event
		oldRegion, region string
	}{
		// test 0
		{oldRegion: "us-east-1", region: "us-east-2"},
		// test 1
		{event: cfn.Event{StackID: "arn:aws:cloudformation:us-west-2:123456789012:stack/teststack/51af3dc0-da77-11e4-872e-1234567db123"},
			oldRegion: "", region: "us-west-1"},
		// test 2
		{event: cfn.Event{StackID: "arn:aws:cloudformation:us-west-2:123456789012:stack/teststack/51af3dc0-da77-11e4-872e-1234567db123"},
			oldRegion: "us-west-1", region: ""},
	} {
		if IsSameRegion(test.event, test.oldRegion, test.region) {
			log.Fatalf("Test case %d, %+v, %s, %s", i, test.event, test.oldRegion, test.region)
		}
	}
}
