#!/usr/bin/env bash
set -e

for dir in $(scripts/moduledirs.sh); do
    (cd $dir && go list -f '{{.Dir}}' ./... | grep -v vendor) | xargs go run golang.org/x/lint/golint -set_exit_status
done
