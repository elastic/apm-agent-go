#!/bin/sh
set -e
$(which go-licenser) || echo "downloading go-licenser" && go get -u github.com/elastic/go-licenser
cd "$(dirname "$(readlink -f "$0")")"
go run go.elastic.co/fastjson/cmd/generate-fastjson -f -o marshal_fastjson.go .
exec go-licenser marshal_fastjson.go
