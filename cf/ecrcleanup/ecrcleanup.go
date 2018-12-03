// # ECR Cleanup
//
// The `ecrcleanup` custom resource is meant to be dependant to an ECR repository. It does
// nothing on creation and will remove all images of the ECR repository on deletion if and only if the stack to which
// it belongs is also in deletion status. It is therefore a relatively dangerous resource.
//
// The AWS ECR resource cannot be deleted if images still exsists. It is where the `ecrcleanup` custom resource
// plays its role by depending on the ECR repository. Its intended usage is for prototype stacks with short
// lived ECR repository when one wants to quickly create and delete ECR repositories.
//
// It is not dangerouse to delete the resource itself when updating the stack as the `ecrcleanup` custom resource only
// cleanup the content when the stack is getting deleted.
//
// ## Syntax
//
// To create an ecrcleanup resource, add the following resource to your cloudformation
// template (yaml notation, json is similar)
//
// ```yaml
// MyEcrCleanup:
//   Type: Custom::EcrCleanup
//   Properties:
//     ServiceToken:
//       Fn::ImportValue:
//         !Sub ${HyperdriveCore}-EcrCleanup
//     Repository: <repository arn>
// ```
//
// ## Properties
//
// `Repository`
//
// > The arn of the repository to clean when the resource is deleted while its stack
// > itself is deleted
// >
// > _Type_: ARN
// >
// > _Required_: Yes
// >
// > _Update Requires_: no interruption
package main

import (
	"context"

	common "github.com/DEEP-IMPACT-AG/hyperdrive/common"
	"github.com/aws/aws-lambda-go/cfn"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	awsecr "github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
)

var ecr *awsecr.ECR
var cf *cloudformation.CloudFormation

// The lambda is started using the AWS lambda go sdk. The handler function
// does the actual work of creating the log group. Cloudformation sends an
// event to signify that a resources must be created, updated or deleted.
func main() {
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		panic(err)
	}
	ecr = awsecr.New(cfg)
	cf = cloudformation.New(cfg)
	lambda.Start(cfn.LambdaWrap(processEvent))
}

// The EcrCleanupProperties is the main data structure for the ecrcleanup resource and
// is defined as a go struct. The struct mirrors the properties as defined above.
// We use the library [mapstructure](https://github.com/mitchellh/mapstructure) to
// decode the generic map from the cloudformation event to the struct.
type EcrCleanupProperties struct {
	Repository string
}

func ecrCleanupProperties(input map[string]interface{}) (EcrCleanupProperties, error) {
	var properties EcrCleanupProperties
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
//    2. the stack is being deleted: in that case, we delete all the images in the
//       repository.
//    3. the stack is not being delete: it is a NOP as well.
// 2. Create, Update: In that case, it is a NOP, the physical ID is simply
//    the ARN of the repository. Giving a new repository will replace the resource.
func processEvent(ctx context.Context, event cfn.Event) (string, map[string]interface{}, error) {
	properties, err := ecrCleanupProperties(event.ResourceProperties)
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
				if err = deleteImages(properties.Repository); err != nil {
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

// We delete all the images in batches.
func deleteImages(repositoryArn string) error {
	images, err := ecr.ListImagesRequest(&awsecr.ListImagesInput{
		RepositoryName: &repositoryArn,
	}).Send()
	if err != nil {
		return errors.Wrapf(err, "could not fetch images for the repository %s", repositoryArn)
	}
	for {
		if len(images.ImageIds) > 0 {
			_, err := ecr.BatchDeleteImageRequest(&awsecr.BatchDeleteImageInput{
				ImageIds:       images.ImageIds,
				RepositoryName: &repositoryArn,
			}).Send()
			if err != nil {
				return errors.Wrapf(err, "could not delete images from the repository %s", repositoryArn)
			}
		}
		if images.NextToken == nil {
			return nil
		}
		images, err = ecr.ListImagesRequest(&awsecr.ListImagesInput{
			RepositoryName: &repositoryArn,
			NextToken:      images.NextToken,
		}).Send()
		if err != nil {
			return errors.Wrapf(err, "could not fetch images for the repository %s", repositoryArn)
		}
	}
}
