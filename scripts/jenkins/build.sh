#!/usr/bin/env bash
set -euxo pipefail

source ./scripts/jenkins/setenv.sh

make precheck check-modules
