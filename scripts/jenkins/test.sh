#!/usr/bin/env bash
set -euxo pipefail

srcdir=`dirname $0`
test -z "$srcdir" && srcdir=.
. ${srcdir}/common.bash

jenkins_setup

go get -v -u github.com/jstemmer/go-junit-report
go get -v -t ./...

export OUT_FILE="build/test-report.out"
mkdir -p build

(go test -race ./... -v 2>&1 | tee ${OUT_FILE}) || echo -e "\033[31;49mTests FAILED\033[0m"
cat ${OUT_FILE} | go-junit-report > build/junit-apm-agent-go.xml
