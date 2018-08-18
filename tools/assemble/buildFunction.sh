#!/usr/bin/env bash

set -e

go test
go vet
GOOS=linux GOARCH=amd64 go build
