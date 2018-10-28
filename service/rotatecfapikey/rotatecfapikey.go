// # RotateCfApiKey
//
// This AWS lambda function is meant to be use in conjunction with the CfApiKey custom resource and as target
// of a cloudwatch rule. To integrate into your cloudformation template, use a similar snippet.
//
// ```yaml
//  ApiKeyRotationRule:
//    Type: "AWS::Events::Rule"
//    Properties:
//      Description:
//        Fn::Sub: |
//          {
//            "StackId": "${AWS::StackId}",
//            "OrdinalParameterName": "ApiKeyOrdinal"
//          }
//      ScheduleExpression: "cron(0 6 ? * SUN *)"
//      Targets:
//      - Fn::ImportValue:
//          !Sub ${HyperdriveCore}-CfApiKeyRotateCfApiKeyLambdaArn
// ```
package main

import (
	"context"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchevents"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"strconv"
)

var cf *cloudformation.CloudFormation
var cwe *cloudwatchevents.CloudWatchEvents

// The lambda is started using the AWS lambda go sdk. The handler function
// does the actual work of creating the apikey. Cloudformation sends an
// event to signify that a resources must be created, updated or deleted.
func main() {
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		panic(err)
	}
	cf = cloudformation.New(cfg)
	cwe = cloudwatchevents.New(cfg)
	lambda.Start(processEvent)
}

type KeyRotationProperties struct {
	StackId, OrdinalParameterName string
}

func keyRotationProperties(input string) (KeyRotationProperties, error) {
	return KeyRotationProperties{}, nil
}

func parametersSpecification(p []cloudformation.Parameter, ordinal string) ([]cloudformation.Parameter, error) {
	p2 := make([]cloudformation.Parameter, len(p), len(p))
	usePreviousValue := true
	for i, parameter := range p {
		if *parameter.ParameterKey == ordinal {
			current, err := strconv.ParseUint(*parameter.ParameterValue, 10, 64)
			if err != nil {
				return nil, errors.Wrapf(err, "Ordinal value is not a uint: %s", *parameter.ParameterValue)
			}
			ordinal := strconv.FormatUint(current + 1, 10)
			p2[i] = cloudformation.Parameter{
				ParameterKey: parameter.ParameterKey,
				ParameterValue: &ordinal,
			}
		} else {
			p2[i] = cloudformation.Parameter{
				ParameterKey: parameter.ParameterKey,
				UsePreviousValue: &usePreviousValue,
			}
		}
	}
	return p2, nil;
}

func processEvent(ctx context.Context, event events.CloudWatchEvent) error {
	fmt.Printf("event: %+v\n", event)
	ruleArn := event.Resources[0]
	rule, err := cwe.DescribeRuleRequest(&cloudwatchevents.DescribeRuleInput{
		Name: &ruleArn,
	}).Send()
	if err != nil {
		return errors.Wrapf(err, "Could not fetch the rule %s", ruleArn)
	}
	properties, err := keyRotationProperties(*rule.Description)
	if err != nil {
		return errors.Wrapf(err,"Invalid Description (must be Json metadata): %s", *rule.Description)
	}
	stacks, err := cf.DescribeStacksRequest(&cloudformation.DescribeStacksInput{
		StackName: &properties.StackId,
	}).Send()
	if err != nil {
		return errors.Wrapf(err, "Could not describe the stack %s", properties.StackId)
	}
	stack := stacks.Stacks[0]
	parameters, err := parametersSpecification(stack.Parameters, properties.OrdinalParameterName)
	if err != nil {
		return err
	}
	rand, err := uuid.NewRandom()
	if err != nil {
		return errors.Wrapf(err, "Cound not create a name for the change set for the stack %s", *stack.StackId)
	}
	randomName := fmt.Sprintf(*stack.StackName+"-%s", rand)
	usePreviousTemplate := true
	cs, err := cf.CreateChangeSetRequest(&cloudformation.CreateChangeSetInput{
		Capabilities: stack.Capabilities,
		ChangeSetName: &randomName,
		ChangeSetType: cloudformation.ChangeSetTypeUpdate,
		StackName: &properties.StackId,
		Parameters: parameters,
		UsePreviousTemplate: &usePreviousTemplate,
	}).Send()
	if err != nil {
		return errors.Wrapf(err, "Could not create the change set for the stack %s", *stack.StackId)
	}
	_, err = cf.ExecuteChangeSetRequest(&cloudformation.ExecuteChangeSetInput{
		ChangeSetName: cs.Id,
	}).Send()
	return err
}