#!/usr/bin/env bash

DESCRIBE=$(git describe --match "v*" --abbrev=4 --tags --dirty=--DIRTY--)
VERSION=$(echo $DESCRIBE | sed  'v/\(.*\)_\1_')

echo ${VERSION}
