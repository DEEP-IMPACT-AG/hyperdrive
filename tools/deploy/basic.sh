#!/usr/bin/env bash

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null && pwd )"
VERSION=$(git describe --match v* --dirty=--DIRTY-- | sed 's:^.\(.*\)$:\1:')

