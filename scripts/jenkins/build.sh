#!/usr/bin/env bash
set -euxo pipefail

srcdir=`dirname $0`
test -z "$srcdir" && srcdir=.
. ${srcdir}/common.bash

jenkins_setup

go get -u -v golang.org/x/tools/cmd/goimports
go get -u -v golang.org/x/lint/golint

make install precheck
