#!/usr/bin/env bash

set -e

echo "mode: atomic"

for pkg in $(go list ./...); do
    go test -coverpkg=go.elastic.co/apm/... -coverprofile=profile.out -covermode=atomic $pkg 1>&2
    if [ -f profile.out ]; then
        grep -v "mode: atomic" profile.out || true
        rm profile.out
    fi
done
