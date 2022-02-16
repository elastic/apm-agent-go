#!/usr/bin/env bash
set -euxo pipefail

source ./scripts/jenkins/debug.sh

# Install tools used only in CI using a local go.mod file.
GO_GET_FLAGS="-modfile=$PWD/scripts/jenkins/jenkins.go.mod"

go get $GO_GET_FLAGS -v github.com/jstemmer/go-junit-report
go get $GO_GET_FLAGS -v github.com/t-yuki/gocover-cobertura
