#!/bin/sh
set -e

#GENERATED_FILES=\
#  model/marshal_fastjson.go \
#  stacktrace/library.go

out=$(go run github.com/jcchavezs/porto/cmd/porto -l .)
out=$(echo $out | (xargs grep -L "Code generated by" || true))

if [ -n "$out" ]; then
  out=$(echo $out | sed 's/ /\n - /')
  printf "Vanity imports are not up to date:\n - $out\n" >&2
  exit 1
fi
