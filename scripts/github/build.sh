#!/usr/bin/env bash
set -euxo pipefail

source ./scripts/github/setenv.sh

make precheck check-modules
