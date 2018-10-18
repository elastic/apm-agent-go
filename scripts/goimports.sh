#!/bin/sh

dirs=$(find . -maxdepth 1 -type d \! \( -name '.*' -or -name vendor \))
exec goimports $GOIMPORTSFLAGS -local go.elastic.co,github.com/elastic *.go $dirs
