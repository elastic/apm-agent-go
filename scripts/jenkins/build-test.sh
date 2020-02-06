#!/usr/bin/env bash
set -euxo pipefail

# Install Go using the same travis approach
echo "Installing ${GO_VERSION} with gimme."
eval "$(curl -sL https://raw.githubusercontent.com/travis-ci/gimme/master/gimme | GIMME_GO_VERSION=${GO_VERSION} bash)"

export OUT_FILE="build/test-report.out"
mkdir -p build
make install check 2> >(tee ${OUT_FILE} 1>&2)

go get -v -u github.com/jstemmer/go-junit-report
go-junit-report > "build/junit-apm-agent-go-${GO_VERSION}.xml" < ${OUT_FILE}
