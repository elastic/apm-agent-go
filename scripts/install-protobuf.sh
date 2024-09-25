#!/bin/sh

PLATFORM=$(go env GOOS)
PROTOBUF_VERSION="24.2"
PROTOC_GO_VERSION="v1.31.0"
PROTOC_GO_GRPC_VERSION="v1.3.0"


if [ "${PLATFORM}" = "darwin" ]; then
	PROTOBUF_URL="https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOBUF_VERSION}/protoc-${PROTOBUF_VERSION}-osx-x86_64.zip"
elif [ "${PLATFORM}" = "linux" ]; then 
	PROTOBUF_URL="https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOBUF_VERSION}/protoc-${PROTOBUF_VERSION}-linux-x86_64.zip"
elif [ "${PLATFORM}" = "windows" ]; then 
	PROTOBUF_URL="https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOBUF_VERSION}/protoc-${PROTOBUF_VERSION}-win64.zip"
else
	echo "Unsupported platform: ${PLATFORM}"
	exit 1
fi

TOOLS_DIR=$(dirname "$(readlink -f -- "$0")")
BUILD_DIR="${TOOLS_DIR}/build"
PROTOBUF_ZIP="/tmp/protobuf.zip"

curl -L "${PROTOBUF_URL}" -o "${PROTOBUF_ZIP}"

if ! unzip -o "${PROTOBUF_ZIP}" -d "${BUILD_DIR}"; then
	echo "failed to extract protobuf"
	exit 1
fi

if ! PATH="${BUILD_DIR}/bin" protoc --version; then
	echo "failed to verify protobuf installation"
	exit 1
fi

GOBIN="${BUILD_DIR}/bin" go install "google.golang.org/protobuf/cmd/protoc-gen-go@${PROTOC_GO_VERSION}"
GOBIN="${BUILD_DIR}/bin" go install "google.golang.org/grpc/cmd/protoc-gen-go-grpc@${PROTOC_GO_GRPC_VERSION}"
