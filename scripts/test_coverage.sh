#!/usr/bin/env bash

set -e

profile_out=$(mktemp)
function on_exit {
    rm -f $profile_out
}
trap on_exit EXIT

echo "mode: atomic"
for dir in $(scripts/moduledirs.sh); do
    (
    cd $dir
    for pkg in $(go list ./...); do
        go test -coverpkg=go.elastic.co/apm/... -coverprofile=$profile_out -covermode=atomic $pkg 1>&2
        if [ -f $profile_out ]; then
            grep -v "mode: atomic" $profile_out || true
            rm $profile_out
        fi
    done
    )
done
