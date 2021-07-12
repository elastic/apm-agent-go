#!/usr/bin/env bash
set -euxo pipefail

if (go run scripts/mingoversion.go 1.11 &>/dev/null); then
  exit
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

# Go 1.8-1.10
pin go.uber.org/multierr v1.6.0 https://github.com/uber-go/multierr
pin github.com/astaxie/beego v1.11.1
pin github.com/stretchr/testify v1.4.0
pin github.com/cucumber/godog v0.8.0
pin github.com/elastic/go-sysinfo v1.3.0
pin google.golang.org/grpc v1.30.0 https://github.com/grpc/grpc-go
pin github.com/jinzhu/gorm v1.9.16
pin github.com/ugorji/go v1.1.10
pin github.com/go-chi/chi v1.5.1
pin github.com/prometheus/client_golang v1.1.0
pin github.com/emicklei/go-restful v2.9.6
pin github.com/go-sql-driver/mysql v1.4.1
pin golang.org/x/net 5f58ad60dda6 https://github.com/golang/net

# Go 1.8-1.9
if (! go run scripts/mingoversion.go 1.10 &>/dev/null); then
  pin github.com/gocql/gocql 16cf9ea1b3e2
  pin github.com/labstack/echo v4.1.9
  pin github.com/lib/pq v1.0.0
  pin github.com/gin-gonic/gin v1.3.0
else
  pin github.com/gin-gonic/gin v1.5.0 # Use gin v1.5.0 for 1.10 only
fi

# Go 1.8 only.
if (! go run scripts/mingoversion.go 1.9 &>/dev/null); then
  pin github.com/golang/protobuf v1.3.5
  pin github.com/olivere/elastic release-branch.v6
  pin golang.org/x/sys fc99dfbffb4e https://go.googlesource.com/sys
fi
