#!/usr/bin/env bash
set -euxo pipefail

if (go run scripts/mingoversion.go 1.10 &>/dev/null); then
  go get -v golang.org/x/lint/golint;
  go get -v golang.org/x/tools/cmd/goimports;
  go get -v github.com/elastic/go-licenser;
fi

if (! go run scripts/mingoversion.go 1.10 &>/dev/null); then
  # Before Go 1.10.0, pin gin-gonic to v1.3.0.
  mkdir -p "${GOPATH}/src/github.com/gin-gonic";
  (
    cd "${GOPATH}/src/github.com/gin-gonic" &&
    git clone https://github.com/gin-gonic/gin &&
    cd gin && git checkout v1.3.0
  );
  # Ping gocql to the latest commit that supports Go 1.9.
  mkdir -p "${GOPATH}/src/github.com/gocql";
  (
    cd "${GOPATH}/src/github.com/gocql" &&
    git clone https://github.com/gocql/gocql &&
    cd gocql && git checkout 16cf9ea1b3e2
  );
  # Before Go 1.10.0, pin go-sql-driver to v1.4.1.
  mkdir -p "${GOPATH}/src/github.com/go-sql-driver";
  (
    cd "${GOPATH}/src/github.com/go-sql-driver" &&
    git clone https://github.com/go-sql-driver/mysql &&
    cd mysql && git checkout v1.4.1
  );
fi

if (! go run scripts/mingoversion.go 1.9 &>/dev/null); then
  # Pin olivere/elastic to release-branch.v6 for Go 1.8.
  mkdir -p "${GOPATH}/src/github.com/olivere/elastic";
  (
    cd "${GOPATH}/src/github.com/olivere" &&
    git clone https://github.com/olivere/elastic &&
    cd elastic && git checkout release-branch.v6
  );
  # Pin golang.org/x/sys to the last commit that supports Go 1.8.
  mkdir -p "${GOPATH}/src/golang.org/x/sys";
  (
    cd "${GOPATH}/src/golang.org/x" &&
    git clone https://go.googlesource.com/sys &&
    cd sys && git checkout fc99dfbffb4e
  );
  # Pin github.com/prometheus/client_golang to v1.1.0,
  # the last release that supports Go 1.8.
  mkdir -p "${GOPATH}/src/github.com/prometheus";
  (
    cd "${GOPATH}/src/github.com/prometheus" &&
    git clone https://github.com/prometheus/client_golang &&
    cd client_golang && git checkout v1.1.0
  );
fi
