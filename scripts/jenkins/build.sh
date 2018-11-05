#!/bin/bash
set -e

srcdir=`dirname $0`
test -z "$srcdir" && srcdir=.
. ${srcdir}//common.bash

jenkins_setup

go get -u -v golang.org/x/tools/cmd/goimports

make install check
