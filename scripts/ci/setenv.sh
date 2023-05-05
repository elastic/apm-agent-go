#!/usr/bin/env bash
set -euxo pipefail

# Install Go using the same travis approach
echo "Installing ${GO_VERSION} with gimme."
eval "$(curl -sL https://raw.githubusercontent.com/travis-ci/gimme/master/gimme | GIMME_GO_VERSION=${GO_VERSION} bash)"

# Install tools used only in CI using a local go.mod file.
GO_INSTALL_FLAGS="-modfile=$PWD/scripts/ci/ci.go.mod"

go install $GO_INSTALL_FLAGS -v github.com/jstemmer/go-junit-report
go install $GO_INSTALL_FLAGS -v github.com/t-yuki/gocover-cobertura
go install $GO_INSTALL_FLAGS -v github.com/elastic/gobench
