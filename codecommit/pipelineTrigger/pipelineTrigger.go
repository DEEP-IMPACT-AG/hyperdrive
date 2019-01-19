package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/pkg/errors"
	"github.com/stanislas/aws-lambda-go/events"
	"github.com/stanislas/aws-lambda-go/lambda"
	"log"
	"os"
	"strings"
)

const EventsBucketName = "EVENTS_BUCKET_NAME"
const Buildspec = `version: 0.2

phases:
  build:
    commands:
      - bash checkout.sh
artifacts:
  type: zip
  files:
    - "**/*"
  base-directory: repo`

const CheckoutSh = `#!/bin/bash

git config --global credential.helper '!aws codecommit credential-helper $@'
git config --global credential.UseHttpPath true

git clone --shallow-submodules https://git-codecommit.%s.amazonaws.com/v1/repos/%s repo
cd repo
%s
cd
`

func main() {
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		log.Fatalf("could not get aws config: %+v\n", err)
	}
	s3 := awss3.New(cfg)
	lambda.Start(processEvent(s3))
}

type Settings struct {
	Pipeline string `json:"pipeline"`
	OnTag    bool   `json:"onTag,omitempty"`
	OnCommit bool   `json:"onCommit,omitempty"`
}

func settings(input string) (Settings, error) {
	var settings Settings
	if err := json.Unmarshal([]byte(input), &settings); err != nil {
		return settings, errors.Wrapf(err, "could not unmarshall settings: %s", input)
	}
	return settings, nil
}

func processEvent(s3 *awss3.S3) func(event events.CodeCommitEvent) (events.CodeCommitEvent, error) {
	return func(event events.CodeCommitEvent) (events.CodeCommitEvent, error) {
		commit := event.Records[0]
		settings, err := settings(commit.CustomData)
		if err != nil {
			return event, err
		}
		awsRegion := commit.AWSRegion
		repository := extractRepository(commit)
		ref := commit.CodeCommit.References[0]
		if isCommit(ref) && settings.OnCommit {
			if err := triggerPipeline(s3, awsRegion, repository, settings.Pipeline, "git checkout "+ref.Commit); err != nil {
				return event, err
			}
		}
		if isTag(ref) && settings.OnTag {
			if err := triggerPipeline(s3, awsRegion, repository, settings.Pipeline, "git checkout "+tag(ref)); err != nil {
				return event, err
			}
		}
		return event, nil
	}
}

func extractRepository(commit events.CodeCommitRecord) string {
	idx := strings.LastIndex(commit.EventSourceARN, ":")
	return commit.EventSourceARN[idx+1:]
}

func isTag(ref events.CodeCommitReference) bool {
	return strings.HasPrefix(ref.Ref, "refs/tags/")
}

func tag(ref events.CodeCommitReference) string {
	return ref.Ref[10:len(ref.Ref)]
}

func isCommit(ref events.CodeCommitReference) bool {
	return strings.HasPrefix(ref.Ref, "refs/heads/")
}

func triggerPipeline(s3 *awss3.S3, awsRegion, repository, pipeline, gitCheckoutCommand string) error {
	buf := new(bytes.Buffer)
	writer := zip.NewWriter(buf)
	var files = []struct {
		Name, Body string
	}{
		{"buildspec.yaml", Buildspec},
		{"checkout.sh", fmt.Sprintf(CheckoutSh, awsRegion, repository, gitCheckoutCommand)},
	}
	for _, file := range files {
		zipFile, err := writer.Create(file.Name)
		if err != nil {
			return errors.Wrapf(err, "unable to create the file %s in the zip archive", file.Name)
		}
		_, err = zipFile.Write([]byte(file.Body))
		if err != nil {
			return errors.Wrapf(err, "unable to write the content %s of the file %s in the zip archive", file.Body, file.Name)
		}
	}
	err := writer.Close()
	if err != nil {
		return errors.Wrap(err, "unable to close the zip archive")
	}
	bucket := os.Getenv(EventsBucketName)
	key := pipeline + "/trigger.zip"
	_, err = s3.PutObjectRequest(&awss3.PutObjectInput{
		Bucket: &bucket,
		Key:    &key,
		Body:   bytes.NewReader(buf.Bytes()),
	}).Send()
	if err != nil {
		return errors.Wrapf(err, "could not put the object %s on the bucket %s", key, bucket)
	}
	return nil
}
