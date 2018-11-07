#!/bin/sh
set -e

out=$(GOIMPORTSFLAGS=-l ./scripts/goimports.sh)
if [ -n "$out" ]; then
  out=$(echo $out | sed 's/ /\n - /')
  printf "goimports differs:\n - $out\n" >&2
  exit 1
fi
