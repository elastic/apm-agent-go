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
fi

if (! go run scripts/mingoversion.go 1.9 &>/dev/null); then
  # For Go 1.8.x, pin go-sql-driver to v1.4.1,
  # the last release that supports Go 1.8.
  mkdir -p "${GOPATH}/src/github.com/go-sql-driver";
  (
    cd "${GOPATH}/src/github.com/go-sql-driver" &&
    git clone https://github.com/go-sql-driver/mysql &&
    cd mysql && git checkout v1.4.1
  );
  # Pin olivere/elastic to release-branch.v6 for Go 1.8.
  mkdir -p "${GOPATH}/src/github.com/olivere/elastic";
  (
    cd "${GOPATH}/src/github.com/olivere" &&
    git clone https://github.com/olivere/elastic &&
    cd elastic && git checkout release-branch.v6
  );
fi
