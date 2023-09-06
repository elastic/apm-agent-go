#!/usr/bin/env bash
set -euxo pipefail

source ./scripts/ci/setenv.sh

make precheck check-modules
