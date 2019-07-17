#!/usr/bin/env bash
set -euxo pipefail

# Install Go using the same travis approach
eval "$(curl -sL https://raw.githubusercontent.com/travis-ci/gimme/master/gimme | GIMME_GO_VERSION=${GO_VERSION} bash)"

go get -u -v golang.org/x/tools/cmd/goimports
go get -u -v golang.org/x/lint/golint
go get -u -v github.com/elastic/go-licenser

make install check
