// # S3 Cleanup
//
// The `s3cleanup` custom resource is helping cleaning up S3 buckets when deleting a Cloudformation
// stack. Possible uses are cleaning up artificats from an artificat S3 bucket when deleting a project
// or cleaning up a S3 bucket before deleting a S3 Bucket resource as the AWS S3 Bucket resource cannot
// be deleted when it is not empty.
//
// The `s3cleanup` custom resource only deletes object when the stack itself is being deleted
// It also safe to remove the resource from an existing stack.
//
// ## Syntax
//
// To create an `s3cleanup` resource, add the following resource to your cloudformation
// template (yaml notation, json is similar)
//
// ```yaml
// MyS3Cleanup:
//   Type: Custom::S3Cleanup
//   Properties:
//     ServiceToken:
//       Fn::ImportValue:
//         !Sub ${HyperdriveCore}-S3Cleanup
//     Bucket: <bucket arn>
//     Prefix: <prefix>
// ```
//
// ## Properties
//
// `Bucket`
//
// > The arn of the S3 Bucket to cleanup when the `s3cleanup` resource is deleted while its stack
// > itself is deleted.
// >
// > _Type_: ARN
// >
// > _Required_: Yes
// >
// > _Update Requires_: no interruption
//
// `Prefix`
//
// > A prefix to delete objects. If the prefix is omitted or is empty, then all objects are deleted.
// >
// > _Type_: String
// >
// > _Required_: No
// >
// > _Update Requires_: nothing (?)
package main

import (
	"context"

	common "github.com/DEEP-IMPACT-AG/hyperdrive/common"
	"github.com/aws/aws-lambda-go/cfn"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
)

var s3 *awss3.S3
var cf *cloudformation.CloudFormation

func main() {
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		panic(err)
	}
	s3 = awss3.New(cfg)
	cf = cloudformation.New(cfg)
	lambda.Start(cfn.LambdaWrap(processEvent))
}

// The S3CleanupProperties is the main data structure for the s3bucket resource and
// is defined as a go struct. The struct mirrors the properties as defined above.
// We use the library [mapstructure](https://github.com/mitchellh/mapstructure) to
// decode the generic map from the cloudformation event to the struct.
type S3CleanupProperties struct {
	Bucket, Prefix string
}

func s3CleanupProperties(input map[string]interface{}) (S3CleanupProperties, error) {
	var properties S3CleanupProperties
	if err := mapstructure.Decode(input, &properties); err != nil {
		return properties, err
	}
	return properties, nil
}

// To process an event, we first decode the resource properties and analyse
// the event. We have 2 cases.
//
// 1. Delete: The delete case it self has 3 sub cases:
//    1. the physical resource id is a failure id, then this is a NOP;
//    2. the stack is being deleted: in that case, we delete all the objects with the given
//       path prefix from the S3 bucket or, if the path prefix is not defined, we delete
//       all the resources.
//    3. the stack is not being delete: it is a NOP as well.
// 2. Create, Update: In that case, it is a NOP, the physical ID is simply
//    the ARN of the s3 bucket concatenated with the path.
//    Giving a new repository will replace the resource. ???
func processEvent(ctx context.Context, event cfn.Event) (string, map[string]interface{}, error) {
	properties, err := s3CleanupProperties(event.ResourceProperties)
	if err != nil {
		return "", nil, err
	}
	switch event.RequestType {
	case cfn.RequestDelete:
		if !common.IsFailurePhysicalResourceId(event.PhysicalResourceID) {
			stacks, err := cf.DescribeStacksRequest(&cloudformation.DescribeStacksInput{
				StackName: &event.StackID,
			}).Send()
			if err != nil {
				return event.PhysicalResourceID, nil, errors.Wrapf(err, "could not fetch the stack for the resource %s", event.PhysicalResourceID)
			}
			stackStatus := stacks.Stacks[0].StackStatus
			if stackStatus == cloudformation.StackStatusDeleteInProgress {
				if err = deleteObjects(properties); err != nil {
					return event.PhysicalResourceID, nil, errors.Wrapf(err, "could not delete the images of the repository %s", event.PhysicalResourceID)
				}
			}
		}
		return event.PhysicalResourceID, nil, nil
	case cfn.RequestCreate:
		return event.LogicalResourceID, nil, nil
	case cfn.RequestUpdate:
		return event.PhysicalResourceID, nil, nil
	default:
		return event.LogicalResourceID, nil, errors.Errorf("unknown request type %s", event.RequestType)
	}
}

func deleteObjects(properties S3CleanupProperties) error {
	versions, err := s3.ListObjectVersionsRequest(&awss3.ListObjectVersionsInput{
		Bucket: &properties.Bucket,
		Prefix: &properties.Prefix,
	}).Send()
	if err != nil {
		return errors.Wrapf(err, "could not fetch versions for the bucket %s", properties.Bucket)
	}
	quiet := true

	for {
		versionsLength := len(versions.Versions)
		if versionsLength > 0 {
			objects := make([]awss3.ObjectIdentifier, versionsLength)
			for i, version := range versions.Versions {
				objects[i] = awss3.ObjectIdentifier{
					Key:       version.Key,
					VersionId: version.VersionId,
				}
			}
			_, err = s3.DeleteObjectsRequest(&awss3.DeleteObjectsInput{
				Bucket: &properties.Bucket,
				Delete: &awss3.Delete{
					Objects: objects,
					Quiet:   &quiet,
				},
			}).Send()
			if err != nil {
				return errors.Wrapf(err, "could not delete objects from the s3 bucket %s", properties.Bucket)
			}
		}
		if *versions.IsTruncated {
			versions, err = s3.ListObjectVersionsRequest(&awss3.ListObjectVersionsInput{
				Bucket:          &properties.Bucket,
				Prefix:          &properties.Prefix,
				KeyMarker:       versions.NextKeyMarker,
				VersionIdMarker: versions.NextVersionIdMarker,
			}).Send()
			if err != nil {
				return errors.Wrapf(err, "could not fetch versions for the bucket %s", properties.Bucket)
			}
		} else {
			return nil
		}
	}
}
