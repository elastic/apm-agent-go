#!/usr/bin/env bash
set -euxo pipefail

source ./scripts/ci/setenv.sh

export GOFLAGS='-run=NONE -benchmem -bench=.'

make test
