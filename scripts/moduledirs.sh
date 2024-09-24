#!/usr/bin/env bash

# Print the module directories. If modules are disabled or unsupported
# by the installed Go toolchain, then just print the current working
# directory (assumed to be the repo top level.)
if test -z "$(go env GOMOD)"; then
    pwd
else
    # Remove the folder internal/scripts ignore once
    # https://github.com/golang/go/issues/65653 has been fixed
    find . -type f -not -path '*/tools/*' \
      -not -path '*/internal/*' -not -path '*/scripts/*' \
      -name go.mod -exec dirname '{}' \;
fi
