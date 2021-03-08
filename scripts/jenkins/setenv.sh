#!/usr/bin/env bash
set -euxo pipefail

# Install Go using the same travis approach
echo "Installing ${GO_VERSION} with gimme."
eval "$(curl -sL https://raw.githubusercontent.com/travis-ci/gimme/master/gimme | GIMME_GO_VERSION=${GO_VERSION} bash)"

# Install tools used only in CI using a local go.mod file when using Go 1.14+.
GO_GET_FLAGS=
if [ "$(go run ./scripts/mingoversion.go -print 1.14)" = "true" ]; then
  GO_GET_FLAGS="-modfile=$PWD/scripts/jenkins/jenkins.go.mod"
fi

go get $GO_GET_FLAGS -v github.com/jstemmer/go-junit-report
go get $GO_GET_FLAGS -v github.com/t-yuki/gocover-cobertura
