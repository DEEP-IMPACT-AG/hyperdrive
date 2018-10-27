#!/usr/bin/env bash

set -e

go test

pushd assemble

go build
./assemble build
rm -f ./assemble

popd