#!/usr/bin/env bash
set -euxo pipefail

srcdir=`dirname $0`
test -z "$srcdir" && srcdir=.
. ${srcdir}/common.bash

jenkins_setup

export GOPATH=$WORKSPACE
eval "$(gvm $GO_VERSION)"
go get -v -u github.com/jstemmer/go-junit-report
go get -v -u github.com/t-yuki/gocover-cobertura
go get -v -t ./...

export COV_FILE="build/coverage/coverage.cov"
export OUT_FILE="build/test-report.out"
mkdir -p build/coverage

go test -race ./... -v -coverprofile="${COV_FILE}" -coverpkg=go.elastic.co/apm/... 2>&1 | tee ${OUT_FILE}
cat ${OUT_FILE} | go-junit-report > build/junit-apm-agent-go.xml

go tool cover -html="${COV_FILE}" -o build/coverage/coverage-apm-agent-go-report.html
gocover-cobertura < "${COV_FILE}" > build/coverage/coverage-apm-agent-go-report.xml