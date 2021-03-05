#!/usr/bin/env bash
set -euxo pipefail

source ./scripts/jenkins/setenv.sh

# Run the tests
set +e
export OUT_FILE="build/test-report.out"
mkdir -p build
make test 2>&1 | tee ${OUT_FILE}
status=$?

go-junit-report > "build/junit-apm-agent-go-${GO_VERSION}.xml" < ${OUT_FILE}

exit ${status}
