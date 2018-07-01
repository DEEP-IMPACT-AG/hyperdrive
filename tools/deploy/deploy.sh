#!/usr/bin/env bash

set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null && pwd )"
VERSION=$(git describe --match "v*" --dirty=--DIRTY-- | sed 's:^.\(.*\)$:\1:')
ARTEFACTS_STACK_NAME=HyperdriveArtefacts

function init () {
    aws cloudformation deploy \
        --stack-name=${ARTEFACTS_STACK_NAME} \
        --template-file=${SCRIPT_DIR}/hyperdrive_artefacts.yaml \
        --parameter-override \
            BaseDomainName=$1 \
            Version=${VERSION}
}

function package () {
    cd ${SCRIPT_DIR}/../..
    goreleaser --snapshot --rm-dist
    cd ${SCRIPT_DIR}
    S3_BUCKET=$(aws cloudformation describe-stacks --stack-name ${ARTEFACTS_STACK_NAME} | jq -r '.Stacks[0].Outputs | map(select(.OutputKey=="ArtefactsBucketName"))[0].OutputValue')
    aws cloudformation package \
        --template-file=hyperdrive_core.yaml \
        --s3-bucket=${S3_BUCKET} \
        --s3-prefix=hyperdrive \
        --output-template=${SCRIPT_DIR}/../../dist/${VERSION}.yaml
}

function deploy () {
    cd ${SCRIPT_DIR}/../../dist
    local key_id=$(aws kms describe-key --key-id alias/aws/ssm | jq -r ".KeyMetadata.KeyId")
    aws cloudformation deploy \
        --capabilities CAPABILITY_IAM \
        --template-file ${VERSION}.yaml \
        --stack-name HyperdriveCore \
        --parameter-override \
            Version=${VERSION} \
            HyperdriveKmsKeyId=${key_id}
}

case $1 in
    "init") init ${2};;
	"package") package;;
	"deploy") deploy;;
	"package-deploy")
	    package
	    deploy
	    ;;
	*)
		echo "Unknown command"
		print_help;
		exit 1
		;;
esac