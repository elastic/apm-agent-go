#!/usr/bin/env bash
set -exo pipefail

# If HOME is not set then use the current directory
# that's normally happening when running in the CI
# owned by Elastic.
if [ -z "$HOME" ] ; then
	HOME=$(realpath ~)
	export HOME
fi

set -u
source ./scripts/jenkins/setenv.sh

export GOFLAGS='-run=NONE -benchmem -bench=.'
export OUT_FILE="build/bench.out"
mkdir -p build

make test | tee ${OUT_FILE}
go-junit-report < ${OUT_FILE} > build/junit-apm-agent-go-bench.xml
