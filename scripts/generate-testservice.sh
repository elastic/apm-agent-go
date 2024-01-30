#!/bin/sh

TOOLS_DIR=$(dirname "$(readlink -f -- "$0")")

PATH="${TOOLS_DIR}/build/bin:${PATH}" protoc \
    --proto_path=./module/apmgrpc/internal/testservice/ \
    --go_out=./module/apmgrpc \
    --go-grpc_out=./module/apmgrpc \
    --go_opt=module=go.elastic.co/apm/module/apmgrpc/v2 \
    --go-grpc_opt=module=go.elastic.co/apm/module/apmgrpc/v2 \
    ./module/apmgrpc/internal/testservice/testservice.proto
