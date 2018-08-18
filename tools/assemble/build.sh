#!/usr/bin/env bash

set -e

go test

pushd make

go build
./make build
rm -f ./make

popd