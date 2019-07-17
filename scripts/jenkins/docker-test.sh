#!/usr/bin/env bash
set -euxo pipefail

# Install Go using the same travis approach
eval "$(curl -sL https://raw.githubusercontent.com/travis-ci/gimme/master/gimme | GIMME_GO_VERSION=${GO_VERSION} bash)"

./scripts/docker-test.sh
