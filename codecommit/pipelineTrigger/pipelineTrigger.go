package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stanislas/aws-lambda-go/events"
	"github.com/stanislas/aws-lambda-go/lambda"
	"log"
)

func main() {
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		log.Fatalf("could not get aws config: %+v\n", err)
	}
	s3 := awss3.New(cfg)
	lambda.Start(processEvent(s3))
}

func processEvent(s3 *awss3.S3) func(event events.CodeCommitEvent) (events.CodeCommitEvent, error) {
	return func(event events.CodeCommitEvent) (events.CodeCommitEvent, error) {
		fmt.Printf("event: %+v\n", event)
		fmt.Printf("custom data: %s\n", event.Records[0].CustomData)
		return event, nil
	}
}
