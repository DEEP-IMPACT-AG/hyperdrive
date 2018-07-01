package main

import (
	"bytes"
	"context"
	"fmt"
	"github.com/DEEP-IMPACT-AG/hyperdrive/common"
	"github.com/aws/aws-lambda-go/cfn"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
)

var s3 *awss3.S3

// The lambda is started using the AWS lambda go sdk. The handler function
// does the actual work of creating the apikey. Cloudformation sends an
// event to signify that a resources must be created, updated or deleted.
func main() {
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		panic(err)
	}
	s3 = awss3.New(cfg)
	lambda.Start(cfn.LambdaWrap(processEvent))
}

type TemplateProperties struct {
	S3Bucket, S3Key string
	TemplateBody    string
}

func templateProperties(input map[string]interface{}) (TemplateProperties, error) {
	var properties TemplateProperties
	if err := mapstructure.Decode(input, &properties); err != nil {
		return properties, err
	}
	return properties, nil
}

func createS3Template(event cfn.Event, properties TemplateProperties) (string, map[string]interface{}, error) {
	s3Bucket := properties.S3Bucket
	s3Key := properties.S3Key
	id := fmt.Sprintf("s3://%s/%s", s3Bucket, s3Key)
	if _, err := s3.PutObjectRequest(&awss3.PutObjectInput{
		Bucket: &s3Bucket,
		Key:    &s3Key,
		Body:   bytes.NewReader([]byte(properties.TemplateBody)),
	}).Send(); err != nil {
		return common.FailurePhysicalResourceId(event), nil, errors.Wrapf(err, "could not create object in bucket %s with key %s", s3Bucket, s3Key)
	}
	return id, nil, nil
}

func processEvent(ctx context.Context, event cfn.Event) (string, map[string]interface{}, error) {
	properties, err := templateProperties(event.ResourceProperties);
	if err != nil {
		return "", nil, err
	}
	switch event.RequestType {
	case cfn.RequestDelete:
		if !common.IsFailurePhysicalResourceId(event.PhysicalResourceID) {
			s3Bucket := event.ResourceProperties["S3Bucket"].(string)
			s3Key := event.ResourceProperties["S3Key"].(string)
			var maxKeys int64 = 100
			versions, err := s3.ListObjectVersionsRequest(&awss3.ListObjectVersionsInput{
				Bucket:  &s3Bucket,
				Prefix:  &s3Key,
				MaxKeys: &maxKeys,
			}).Send()
			if err != nil {
				return event.PhysicalResourceID, nil, errors.Wrapf(err, "could not fetch versions for the s3 template %s", event.PhysicalResourceID)
			}
			for _, version := range versions.Versions {
				if _, err = s3.DeleteObjectRequest(&awss3.DeleteObjectInput{
					Bucket:    &s3Bucket,
					Key:       &s3Key,
					VersionId: version.VersionId,
				}).Send(); err != nil {
					return event.PhysicalResourceID, nil, errors.Wrapf(err, "could not delete the version %s of s3 template %s", *version.VersionId, event.PhysicalResourceID)
				}
			}
			if err != nil {
				return event.PhysicalResourceID, nil, errors.Wrapf(err, "could not delete the s3 template %s", event.PhysicalResourceID)
			}
		}
		return event.PhysicalResourceID, nil, nil
	case cfn.RequestUpdate:
		return createS3Template(event, properties)
	case cfn.RequestCreate:
		return createS3Template(event, properties)
	default:
		return event.PhysicalResourceID, nil, errors.Errorf("unknown request type %s", event.RequestType)
	}
}
