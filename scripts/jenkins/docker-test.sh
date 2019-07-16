#!/usr/bin/env bash
set -euxo pipefail

srcdir=`dirname $0`
test -z "$srcdir" && srcdir=.
. ${srcdir}/common.bash

jenkins_setup

./scripts/docker-test.sh
