#!/bin/bash
set -e

srcdir=`dirname $0`
test -z "$srcdir" && srcdir=.
. ${srcdir}//common.bash

jenkins_setup

make install check
