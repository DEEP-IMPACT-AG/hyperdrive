#!/usr/bin/env bash

set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null && pwd )"
VERSION=$(git describe --match "v*" --dirty=--DIRTY-- | sed 's:^.\(.*\)$:\1:')
CORE_STACK_NAME=HyperdriveCore

function init () {
    aws cloudformation deploy \
        --stack-name=${CORE_STACK_NAME} \
        --template-file=${SCRIPT_DIR}/hyperdrive_core.yaml \
        --capabilities CAPABILITY_NAMED_IAM \
        --parameter-override \
            BaseDomainName=$1 \
            Version=${VERSION}
}

function package () {
    cd ${SCRIPT_DIR}/../..
    goreleaser --snapshot --rm-dist
    cd ${SCRIPT_DIR}
    S3_BUCKET=$(aws cloudformation describe-stacks --stack-name ${CORE_STACK_NAME} | jq -r '.Stacks[0].Outputs | map(select(.OutputKey=="ArtifactsBucketName"))[0].OutputValue')
    aws cloudformation package \
        --template-file=hyperdrive_lambda.yaml \
        --s3-bucket=${S3_BUCKET} \
        --s3-prefix=.hyperdrive/${VERSION} \
        --output-template=${SCRIPT_DIR}/../../dist/${VERSION}.yaml
}

function deploy () {
    cd ${SCRIPT_DIR}/../../dist
    local key_id=$(aws kms describe-key --key-id alias/aws/ssm | jq -r ".KeyMetadata.KeyId")
    aws cloudformation deploy \
        --capabilities CAPABILITY_NAMED_IAM \
        --template-file ${VERSION}.yaml \
        --stack-name HyperdriveLambda \
        --parameter-override \
            Version=${VERSION} \
            HyperdriveKmsKeyId=${key_id}
}

case $1 in
    "core") init ${2};;
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