#!/usr/bin/env bash
set -euxo pipefail

# Install Golang using gvm
echo "Installing ${GO_VERSION} with gvm."
MSG="environment variable missing"
GO_VERSION=${GO_VERSION:?$MSG}
HOME=${HOME:?$MSG}
OS=$(uname -s| tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m| tr '[:upper:]' '[:lower:]')
GVM_CMD="${HOME}/bin/gvm"

if command -v go ; then
    set +e
    echo "Found Go. Checking version.."
    FOUND_GO_VERSION=$(go version|awk '{print $3}'|sed s/go//)
    if [ "$FOUND_GO_VERSION" == "$GO_VERSION" ]
    then
        echo "Versions match. No need to install Go. Exiting."
        exit 0
    fi
    set -e
fi

if [ "${ARCH}" == "aarch64" ] ; then
    GVM_ARCH_SUFFIX=arm64
elif [ "${ARCH}" == "x86_64" ] ; then
    GVM_ARCH_SUFFIX=amd64
elif [ "${ARCH}" == "i686" ] ; then
    GVM_ARCH_SUFFIX=386
elif [ "${ARCH}" == "arm64" ] ; then
    GVM_ARCH_SUFFIX=arm64
else
    GVM_ARCH_SUFFIX=arm
fi

echo "UNMET DEP: Installing Go"
mkdir -p "${HOME}/bin"

curl -sSLo "${GVM_CMD}" "https://github.com/andrewkroh/gvm/releases/download/v0.5.2/gvm-${OS}-${GVM_ARCH_SUFFIX}"
chmod +x "${GVM_CMD}"

eval "$("${GVM_CMD}" "${GO_VERSION}")"

# Install tools used only in CI using a local go.mod file.
GO_INSTALL_FLAGS="-modfile=$PWD/scripts/ci/ci.go.mod"

go install $GO_INSTALL_FLAGS -v github.com/jstemmer/go-junit-report
go install $GO_INSTALL_FLAGS -v github.com/t-yuki/gocover-cobertura
go install $GO_INSTALL_FLAGS -v github.com/elastic/gobench
