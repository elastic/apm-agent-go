#!/usr/bin/env bash
set -euxo pipefail

# Install tools used only in CI using a local go.mod file.
GO_INSTALL_FLAGS="-modfile=$PWD/scripts/ci/ci.go.mod"

go install $GO_INSTALL_FLAGS -v github.com/t-yuki/gocover-cobertura
