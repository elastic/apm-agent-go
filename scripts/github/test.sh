#!/usr/bin/env bash
set -euxo pipefail

source ./scripts/github/setenv.sh

# Run the tests
set +e
export OUT_FILE="build/test-report.out"
mkdir -p build
make test 2>&1 | tee ${OUT_FILE}
status=$?

go-junit-report > "build/junit-apm-agent-go.xml" < ${OUT_FILE}

exit ${status}
