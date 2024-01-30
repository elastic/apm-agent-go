#!/bin/sh
set -e
cd "$(dirname "$(readlink -f "$0")")"

GOMOD=$(pwd)/../tools/go.mod
go run -modfile=$GOMOD go.elastic.co/fastjson/cmd/generate-fastjson -f -o marshal_fastjson.go .
exec go run -modfile=$GOMOD github.com/elastic/go-licenser marshal_fastjson.go
