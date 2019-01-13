#!/usr/bin/env bash

set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null && pwd )"
VERSION=$(git describe --match "v*" --dirty=--DIRTY-- | sed 's:^.\(.*\)$:\1:')
CORE_STACK_NAME=HyperdriveCore


function package () {
    cd ${SCRIPT_DIR}/../../codecommit/pipelineTrigger
    GOOS=linux GOARCH=amd64 go build
    cd ${SCRIPT_DIR}
    S3_BUCKET=$(aws cloudformation describe-stacks --stack-name ${CORE_STACK_NAME} | jq -r '.Stacks[0].Outputs | map(select(.OutputKey=="ArtifactsBucketName"))[0].OutputValue')
    aws cloudformation package \
        --template-file=hd_lambda_test.yaml \
        --s3-bucket=${S3_BUCKET} \
        --s3-prefix=.hyperdrivetest \
        --output-template=packaged.yaml
}

function deploy () {
    cd ${SCRIPT_DIR}
    local key_id=$(aws kms describe-key --key-id alias/aws/ssm | jq -r ".KeyMetadata.KeyId")
    aws cloudformation deploy \
        --capabilities CAPABILITY_NAMED_IAM \
        --template-file packaged.yaml \
        --stack-name HyperdriveLambdaTest \
        --parameter-override \
            HyperdriveKmsKeyId=${key_id}
}

case $1 in
	"lambda-package") package;;
	"lambda-deploy") deploy;;
	"lambda-package-deploy")
	    package
	    deploy
	    ;;
	*)
		echo "Unknown command"
		print_help;
		exit 1
		;;
esac