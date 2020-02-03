#!/usr/bin/env bash
set -euxo pipefail

if (go run scripts/mingoversion.go 1.10 &>/dev/null); then
  go get -v golang.org/x/lint/golint;
  go get -v golang.org/x/tools/cmd/goimports;
  go get -v github.com/elastic/go-licenser;
fi

# Pin various dependencies for old Go versions.

function pin() {
  repo=$1
  commit=$2
  orgdir=$(dirname "${GOPATH}/src/$repo")
  projname=$(basename "$repo")
  if [ $# -eq 3 ]; then
    url=$3
  else
    url="https://$repo"
  fi
  mkdir -p "$orgdir"
  (cd "$orgdir" && git clone "$url" && cd $projname && git checkout $commit)
}

if (! go run scripts/mingoversion.go 1.11 &>/dev/null); then
  pin github.com/gin-gonic/gin v1.3.0
  pin github.com/stretchr/testify v1.4.0
fi

if (! go run scripts/mingoversion.go 1.10 &>/dev/null); then
  pin github.com/gocql/gocql 16cf9ea1b3e2
  pin github.com/go-sql-driver/mysql v1.4.1
  pin github.com/labstack/echo v4.1.9
fi

if (! go run scripts/mingoversion.go 1.9 &>/dev/null); then
  pin github.com/olivere/elastic release-branch.v6
  pin golang.org/x/sys fc99dfbffb4e https://go.googlesource.com/sys
  pin github.com/prometheus/client_golang v1.1.0
fi
