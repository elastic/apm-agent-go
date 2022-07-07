#!/usr/bin/env bash
set -euxo pipefail

source ./scripts/jenkins/setenv.sh

export GOFLAGS='-run=NONE -benchmem -bench=.'
export OUT_FILE="bench.out"
mkdir -p build

make test | tee ${OUT_FILE}
go-junit-report < ${OUT_FILE} > build/junit-apm-agent-go-bench.xml
