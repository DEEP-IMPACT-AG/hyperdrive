#!/usr/bin/env bash

set -e

rm -fr ./build
mkdir build

GOOS=linux go build -o build/main

aws cloudformation package \
    --profile codesmith \
    --template-file template.yml \
    --s3-bucket lambdacfartifacts-artifactbucket-1jt4stxatm74x \
    --s3-prefix lambda \
    --output-template-file packaged-template.yml