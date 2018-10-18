#!/usr/bin/env bash

set -e

for pkg in $(go list ./...); do
    go test -coverpkg=go.elastic.co/apm/... -coverprofile=profile.out -covermode=atomic $pkg 1>&2
    if [ -f profile.out ]; then
        cat profile.out
        rm profile.out
    fi
done
