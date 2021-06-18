#!/bin/sh

dirs=$(find . -maxdepth 1 -type d \! \( -name '.*' -or -name vendor \))
exec go run golang.org/x/tools/cmd/goimports $GOIMPORTSFLAGS -local go.elastic.co,github.com/elastic *.go $dirs
