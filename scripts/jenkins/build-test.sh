#!/usr/bin/env bash
set -euxo pipefail

./scripts/jenkins/build.sh
./scripts/jenkins/test.sh
