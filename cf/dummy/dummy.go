package main

import (
	"context"

	"github.com/aws/aws-lambda-go/cfn"
	"github.com/aws/aws-lambda-go/lambda"
)

func main() {
	lambda.Start(cfn.LambdaWrap(processEvent))
}

func processEvent(ctx context.Context, event cfn.Event) (string, map[string]interface{}, error) {
	return event.LogicalResourceID, nil, nil;
}
