#!/usr/bin/env bash

set -e

if [ -z ${CIRCLE_TAG} ]; then
	echo "No Tag = No Release"
else
	echo "Release: ${CIRCLE_TAG}"
	pushd make
	go build
	./make release
	rm -f ./make
	popd
fi
