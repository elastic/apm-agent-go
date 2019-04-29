#!/usr/bin/env bash

# Print the module directories. If modules are disabled or unsupported
# by the installed Go toolchain, then just print the current working
# directory (assumed to be the repo top level.)
if test -z "$(go env GOMOD)"; then
    pwd
else
    find . -type f -name go.mod \! -path './vendor/*' -exec dirname '{}' \;
fi
