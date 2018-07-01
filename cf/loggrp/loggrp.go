// # Log Group
//
// The out-of-the box `AWS::Logs::LogGroup` does not allow to create log
// groups in different region. This can be useful in the case of
// lambda@edge functions that are always created in the us-east-1 region
// but that always log in the nearest region where they are used. To
// configure those log groups via cloudformation (retention, tags), we have
// the `loggrp` custom resource lambda.
//
// ## Syntax
// To create a new log group, add the following resource to your cloudformation
// template (yaml notation, json is similar)
//
// ```yaml
// MyLogGroup:
//   Type: Custom::LogGroup
//   Properties:
//     ServiceToken:
//       Fn::ImportValue:
//         !Sub ${HyperdriveCore}-LogGroup
//     LogGroupName: <log group name>
//     Region: <region of the loggrp>
//     RetentionInDays: <retention in days>
//     Tags:
//       <key>: <value>
//       ...
// ```
//
// ## Properties
//
// `ServiceToken`
//
// > The reference to the ARN of this lambda function; imported via the
// > hyperdrive core stack.
// >
// > _Type_: ARN
// >
// > _Required_: Yes
//
// `LogGroupName`
//
// > The name of the log group. It is also its ID.
// >
// > _Type_: String
// >
// > _Required: Yes
// >
// > _Update Requires_: Replacement
//
// `Region`
//
// > The region for the log group. This is mostly useful to create log
// > group outside the us-east-1 region for lamdba@edge functions.
// >
// > _Type_: Region (string)
// >
// > _Required_: No
// >
// > _Update Requires_: Replacement if different from the current region
//
// `RetentionInDays`
//
// > Period of retention of the logs.
// >
// > _Type_: Integer as String
// >
// > _Required_: No
// >
// > _Update Requires_: No interruption
//
// `Tags`
//
// > Tags to apply on the log group.
// >
// > _Type_: map of String to String. The keys are the names of the tags
// > and the values are the values of the tags.
// >
// > _Required_: No
// >
// > _Update Requires_: No interruption
//
// ## Return Values
//
// `Ref`
//
// The `Ref` intrinsic function gives the name of the log group.
//
//
// `Fn::GetAtt`
//
// The resource gives the ARN of the log group under the attribute `Arn`.
//
// ## Example
//
// The following example creates a log group with 90 days retention in the
// eu-west-1 region.
//
// ```yaml
// LambdaEdgeLogGroupEuWest1:
//   Type: Custom::LogGroup
//   Properties:
//     ServiceToken:
//       Fn::ImportValue:
//         !Sub ${HyperdriveCore}-LogGroup
//     LogGroupName: /aws/lambda/us-east-1.lambda_at_edge
//     Region: eu-west-1
//     RetentionInDays: "90"
// ```
//
// ## Implementation
//
// The implemention of the `loggrp` lambda uses the
// [AWS Lambda Go](https://github.com/aws/aws-lambda-go) library to
// simplify the integration. It is run in the `go1.x` runtime.
package main

import (
	"context"
	common "github.com/DEEP-IMPACT-AG/hyperdrive/common"
	"github.com/aws/aws-lambda-go/cfn"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"reflect"
	"strconv"
)

// The lambda is started using the AWS lambda go sdk. The handler function
// does the actual work of creating the log group. Cloudformation sends an
// event to signify that a resources must be created, updated or deleted.
func main() {
	lambda.Start(cfn.LambdaWrap(processEvent))
}

// The main data structure for the log group resource is defined as a go
// struct. The struct mirrors the properties as defined above. We use the
// library [mapstructure](https://github.com/mitchellh/mapstructure) to
// decode the generic map from the cloudformation event to the struct.
type LogGroupProperties struct {
	Region, LogGroupName string
	RetentionInDays      string
	Tags                 map[string]string
}

func logGroupProperties(input map[string]interface{}) (LogGroupProperties, error) {
	var properties LogGroupProperties
	if err := mapstructure.Decode(input, &properties); err != nil {
		return properties, err
	}
	return properties, nil
}

// To process an event, we first decode the resource properties, create a
// AWS cloudwatch logs client and analyse the event itself. We have 2 cases.
//
// 1. Delete: The delete case it self has 2 sub cases: if the physical
//    resource id is a failure id, then this is a NOP, otherwise we delete
//    the log group.
// 2. Create, Update: In that case, we proceed to create the log group with
//    the given retention period and the given tags.
//    > As of 2018-07-11, any change to the properties of the log group
//    > will trigger a replacement. It is buggy to change the tags or the
//    > retention period.
func processEvent(ctx context.Context, event cfn.Event) (string, map[string]interface{}, error) {
	properties, err := logGroupProperties(event.ResourceProperties)
	if err != nil {
		return "", nil, err
	}
	logs, err := logService(properties)
	if err != nil {
		return "", nil, err
	}
	switch event.RequestType {
	case cfn.RequestDelete:
		if !common.IsFailurePhysicalResourceId(event.PhysicalResourceID) {
			_, err := logs.DeleteLogGroupRequest(&cloudwatchlogs.DeleteLogGroupInput{
				LogGroupName: &event.PhysicalResourceID,
			}).Send()
			if err != nil {
				return event.PhysicalResourceID, nil, errors.Wrapf(err, "could not delete log group %s", event.PhysicalResourceID)
			}
		}
		return event.PhysicalResourceID, nil, nil
	case cfn.RequestCreate:
		return createLogGroup(logs, properties)
	case cfn.RequestUpdate:
		oldProperties, err := logGroupProperties(event.OldResourceProperties)
		if err != nil {
			return "", nil, err
		}
		if oldProperties.LogGroupName != properties.LogGroupName || !common.IsSameRegion(event, oldProperties.Region, properties.Region) {
			return createLogGroup(logs, properties)
		}
		data, err := updateLogGroup(logs, event, oldProperties, properties)
		return event.PhysicalResourceID, data, err
	default:
		return event.PhysicalResourceID, nil, errors.Errorf("unknown request type %s", event.RequestType)
	}
}

// ### Create
//
// Creating is a straightforward multi-step process since not all
// properties can be written at the same time. We first create the log
// group with its tags. We then put the retention policy in place, if
// applicable. Finally, we need to fetch the log group arn separately to
// give is as attribute to the resources.
func createLogGroup(logs *cloudwatchlogs.CloudWatchLogs, properties LogGroupProperties) (string, map[string]interface{}, error) {
	// 1. Create the log group
	_, err := logs.CreateLogGroupRequest(&cloudwatchlogs.CreateLogGroupInput{
		LogGroupName: &properties.LogGroupName,
		Tags:         properties.Tags,
	}).Send()
	if err != nil {
		return "", nil, err
	}

	// 2. Modify the retention period if applicable
	if len(properties.RetentionInDays) > 0 {
		if err := putRetentionPolicy(logs, properties.LogGroupName, properties.RetentionInDays); err != nil {
			return properties.LogGroupName, nil, err
		}
	}

	// 3. Fetch the log group to get the arn
	arn, err := fetchLogGroupArn(logs, properties.LogGroupName)
	if err != nil {
		return properties.LogGroupName, nil, err
	}

	// 4. Construct the response to cloudformation.
	return properties.LogGroupName, map[string]interface{}{"Arn": *arn}, nil
}

func putRetentionPolicy(logs *cloudwatchlogs.CloudWatchLogs, logGroupName string, retention string) error {
	r, err := strconv.ParseInt(retention, 10, 64)
	if err != nil {
		return errors.Wrapf(err, "could not parse retention in days %s for group %s", retention, logGroupName)
	}
	_, err = logs.PutRetentionPolicyRequest(&cloudwatchlogs.PutRetentionPolicyInput{
		LogGroupName:    &logGroupName,
		RetentionInDays: &r,
	}).Send()
	if err != nil {
		return errors.Wrapf(err, "could not put retention policy for log group %s", logGroupName)
	}
	return nil
}

func fetchLogGroupArn(logs *cloudwatchlogs.CloudWatchLogs, logGroupName string) (*string, error) {
	data, err := logs.DescribeLogGroupsRequest(&cloudwatchlogs.DescribeLogGroupsInput{
		LogGroupNamePrefix: &logGroupName,
	}).Send()
	if err != nil {
		return nil, errors.Wrapf(err, "could not fetch log groups with prefix %s", logGroupName)
	}
	var arn *string
	for _, group := range data.LogGroups {
		if *group.LogGroupName == logGroupName {
			arn = group.Arn
			break
		}
	}
	if arn == nil {
		return nil, errors.Errorf("log group not found for name %s", logGroupName)
	}
	return arn, nil
}

// ### Update
// Only the RetentionInDays and the Tags can be updated without replacing
// the resource.
//
// 1. Retention: if the retention has been dropped, we delete the
//    corresponding RetentionPolicy, otherwise, we update/create the
//    Retention policy.
// 2. If the tags have changed, we first delete the old tags and add the
//    new tags in a second step.
func updateLogGroup(logs *cloudwatchlogs.CloudWatchLogs, event cfn.Event, oldProperties LogGroupProperties, properties LogGroupProperties) (map[string]interface{}, error) {
	if oldProperties.RetentionInDays != properties.RetentionInDays {
		if len(oldProperties.RetentionInDays) > 0 && len(properties.RetentionInDays) == 0 {
			_, err := logs.DeleteRetentionPolicyRequest(&cloudwatchlogs.DeleteRetentionPolicyInput{
				LogGroupName: &event.PhysicalResourceID,
			}).Send()
			if err != nil {
				return nil, errors.Wrapf(err, "could not delete retention policy for log group %s", event.PhysicalResourceID)
			}
		} else {
			err := putRetentionPolicy(logs, event.PhysicalResourceID, properties.RetentionInDays)
			if err != nil {
				return nil, err
			}
		}
	}
	if !reflect.DeepEqual(oldProperties.Tags, properties.Tags) {
		tags, err := logs.ListTagsLogGroupRequest(&cloudwatchlogs.ListTagsLogGroupInput{
			LogGroupName: &event.PhysicalResourceID,
		}).Send()
		if err != nil {
			return nil, errors.Wrapf(err, "could not list the tags for log group %s", event.PhysicalResourceID)
		}
		t := make([]string, 0, len(tags.Tags))
		for k := range tags.Tags {
			t = append(t, k)
		}
		_, err = logs.UntagLogGroupRequest(&cloudwatchlogs.UntagLogGroupInput{
			LogGroupName: &event.PhysicalResourceID,
			Tags:         t,
		}).Send()
		if err != nil {
			return nil, errors.Wrapf(err, "could untag the log group %s", event.PhysicalResourceID)
		}
		_, err = logs.TagLogGroupRequest(&cloudwatchlogs.TagLogGroupInput{
			LogGroupName: &event.PhysicalResourceID,
			Tags:         properties.Tags,
		}).Send()
		if err != nil {
			return nil, errors.Wrapf(err, "could tag the log group %s", event.PhysicalResourceID)
		}
	}
	arn, err := fetchLogGroupArn(logs, event.PhysicalResourceID)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"Arn": *arn}, nil
}

// ### SDK client
//
// We use the
// [cloudwatchlogs sdk v2](https://github.com/aws/aws-sdk-go-v2/tree/master/service/cloudwatchlogs)
// to create the certificate. The client is created with the default
// credential chain loader, if need be with the supplied region.
func logService(properties LogGroupProperties) (*cloudwatchlogs.CloudWatchLogs, error) {
	cfg, err := external.LoadDefaultAWSConfig(
		external.WithRegion(properties.Region),
	)
	if err != nil {
		return nil, errors.Wrapf(err, "could not create aws config with region %s", properties.Region)
	}
	return cloudwatchlogs.New(cfg), nil
}
