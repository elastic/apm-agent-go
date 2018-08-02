#!/usr/bin/env bash

set -e

for pkg in $(go list ./...); do
    go test -coverpkg=github.com/elastic/apm-agent-go/... -coverprofile=profile.out -covermode=atomic $pkg 1>&2
    if [ -f profile.out ]; then
        cat profile.out
        rm profile.out
    fi
done
