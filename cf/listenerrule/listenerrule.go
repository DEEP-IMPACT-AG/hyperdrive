package main

import (
	"context"
	"github.com/DEEP-IMPACT-AG/hyperdrive/common"
	"github.com/aws/aws-lambda-go/cfn"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/elbv2"
	"github.com/pkg/errors"
)

var elb *elbv2.ELBV2

func main() {
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		panic(err)
	}
	elb = elbv2.New(cfg)
	lambda.Start(cfn.LambdaWrap(processEvent))
}

type RuleProperties struct {

}


func processEvent(ctx context.Context, event cfn.Event) (string, map[string]interface{}, error) {
	properties, err := domainProperties(event.ResourceProperties);
	if err != nil {
		return "", nil, err
	}
	switch event.RequestType {
	case cfn.RequestDelete:
		if !common.IsFailurePhysicalResourceId(event.PhysicalResourceID) {
			userPoolId := event.ResourceProperties["UserPoolId"].(string)
			_, err := idp.DeleteUserPoolDomainRequest(&cognitoidentityprovider.DeleteUserPoolDomainInput{
				Domain:     &event.PhysicalResourceID,
				UserPoolId: &userPoolId,
			}).Send()
			if err != nil {
				return event.PhysicalResourceID, nil, errors.Wrapf(err, "could not delete the UserPoolDomain %s", event.PhysicalResourceID)
			}
		}
		return event.PhysicalResourceID, nil, nil
	case cfn.RequestUpdate:
		return createDomain(event, properties)
	case cfn.RequestCreate:
		return createDomain(event, properties)
	default:
		return event.PhysicalResourceID, nil, errors.Errorf("unknown request type %s", event.RequestType)
	}
}