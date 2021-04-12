#!/usr/bin/env bash
set -euxo pipefail

if (go run scripts/mingoversion.go 1.12 &>/dev/null); then
  go install -v golang.org/x/lint/golint
  go install -v golang.org/x/tools/cmd/goimports
  go install -v github.com/elastic/go-licenser
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
  (cd "$orgdir" && git clone "$url" "$projname" && cd $projname && git checkout $commit)
}

if (! go run scripts/mingoversion.go 1.11 &>/dev/null); then
  pin go.uber.org/multierr v1.6.0 https://github.com/uber-go/multierr
  pin github.com/astaxie/beego v1.11.1
  pin github.com/gin-gonic/gin v1.3.0
  pin github.com/stretchr/testify v1.4.0
  pin github.com/cucumber/godog v0.8.0
  pin github.com/elastic/go-sysinfo v1.3.0
  pin google.golang.org/grpc v1.30.0 https://github.com/grpc/grpc-go
  pin github.com/jinzhu/gorm v1.9.16
  pin github.com/ugorji/go v1.1.10
  pin github.com/go-chi/chi v1.5.1
  pin github.com/prometheus/client_golang v1.1.0
  pin github.com/emicklei/go-restful v2.9.6
fi

if (! go run scripts/mingoversion.go 1.10 &>/dev/null); then
  pin github.com/gocql/gocql 16cf9ea1b3e2
  pin github.com/go-sql-driver/mysql v1.4.1
  pin github.com/labstack/echo v4.1.9
  pin github.com/lib/pq v1.0.0
fi

if (! go run scripts/mingoversion.go 1.9 &>/dev/null); then
  pin github.com/golang/protobuf v1.3.5
  pin github.com/olivere/elastic release-branch.v6
  pin golang.org/x/sys fc99dfbffb4e https://go.googlesource.com/sys
fi
