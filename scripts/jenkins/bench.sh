#!/usr/bin/env bash
set -exo pipefail

## Buildkite specific configuration
export GO111MODULE=on

# If HOME is not set then use the current directory
# that's normally happening when running in the CI
# owned by Elastic.
if [ -z "$HOME" ] ; then
	HOME=$(realpath ~)
	export HOME
fi

# Make sure gomod can be deleted automatically as part of the CI
clean_up () {
  ARG=$?
  # see https://github.com/golang/go/issues/31481#issuecomment-485008558
  chmod u+w -R $GOPATH/pkg/mod
  exit $ARG
}
trap clean_up EXIT

## Bench specific
set -u
source ./scripts/jenkins/setenv.sh

export GOFLAGS='-run=NONE -benchmem -bench=.'
export OUT_FILE="build/bench.out"
mkdir -p build

make test | tee ${OUT_FILE}
go-junit-report < ${OUT_FILE} > build/junit-apm-agent-go-bench.xml
