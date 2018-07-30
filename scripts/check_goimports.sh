#!/bin/sh

dirs=$(find . -maxdepth 1 -type d \! \( -name '.*' -or -name vendor \))
out=$(goimports -l -local github.com/elastic *.go $dirs)
if [ -n "$out" ]; then
  out=$(echo $out | sed 's/ /\n - /')
  printf "goimports differs:\n - $out\n" >&2
  exit 1
fi
