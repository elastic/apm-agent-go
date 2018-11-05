#!/bin/bash
set -exo pipefail

srcdir=`dirname $0`
test -z "$srcdir" && srcdir=.
. ${srcdir}/common.bash

jenkins_setup

export GOPATH=$WORKSPACE
eval "$(gvm $GO_VERSION)"
go get -v -u github.com/jstemmer/go-junit-report
go get -v -t ./...

export OUT_FILE="build/bench.out"
mkdir -p build
go test -run=NONE -benchmem -bench=. ./... -v > ${OUT_FILE} 2>&1 
cat ${OUT_FILE} | go-junit-report > build/junit-apm-agent-go-bench.xml
