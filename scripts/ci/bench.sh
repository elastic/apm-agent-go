#!/usr/bin/env bash
set -exo pipefail

## Buildkite specific configuration
if [ "$CI" == "true" ] ; then
	export GO111MODULE=on

	# If HOME is not set then use the current directory
	# that's normally happening when running in the CI
	# owned by Elastic.
	if [ -z "$HOME" ] ; then
		HOME=$(realpath ~)
		export HOME
	fi

	# Make sure gomod can be deleted automatically as part of the CI
	clean_up () {
	  ARG=$?
	  # see https://github.com/golang/go/issues/31481#issuecomment-485008558
	  chmod u+w -R $HOME/go/pkg
	  exit $ARG
	}
	trap clean_up EXIT
fi

## Fetch the latest stable goversion
export GO_VERSION=$(curl 'https://go.dev/VERSION?m=text' | grep 'go' | sed 's#go##g')

## Bench specific
set -u
source ./scripts/ci/setenv.sh

export GOFLAGS='-run=NONE -benchmem -bench=.'
export OUT_FILE="build/bench.out"
mkdir -p build

make test | tee ${OUT_FILE}

## Send data if running in the CI
if [ "$CI" == "true" ] ; then
	set +x
	set +u
	echo "Sending data with gobench"
	go run -modfile=scripts/ci/ci.go.mod github.com/elastic/gobench -index "benchmark-go" -es "${APM_AGENT_GO_CLOUD_SECRET}" < ${OUT_FILE}
fi
