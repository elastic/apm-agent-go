#!/usr/bin/env bash
set -euxo pipefail

# Install Go using the same travis approach
echo "Installing ${GO_VERSION} with gimme."
eval "$(curl -sL https://raw.githubusercontent.com/travis-ci/gimme/master/gimme | GIMME_GO_VERSION=${GO_VERSION} bash)"

go get -v -u github.com/jstemmer/go-junit-report

export GOFLAGS='-run=NONE -benchmem -bench=.'
export OUT_FILE="build/bench.out"
mkdir -p build

make install test | tee ${OUT_FILE}
go-junit-report < ${OUT_FILE} > build/junit-apm-agent-go-bench.xml
