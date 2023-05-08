#!/usr/bin/env bash

# Bash strict mode
set -eo pipefail
trap 's=$?; echo >&2 "$0: Error on line "$LINENO": $BASH_COMMAND"; exit $s' ERR

# Found current script directory
readonly RELATIVE_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

# Found project directory
readonly BASE_PROJECT="$(dirname $(dirname "${RELATIVE_DIR}"))"

# Extract application version
APP_VERSION="$(sed -n 's/AgentVersion = \"\(.*\)\"/\1/p' version.go | sed -e 's/^[[:space:]]*//')"

# Create a dist folder
rm -rf "${BASE_PROJECT}/dist"
mkdir -p "${BASE_PROJECT}/dist"

# Create tarball
cd "${BASE_PROJECT}"
tar -czf "./dist/workspace-${APP_VERSION}.tar.gz" --exclude="workspace-${APP_VERSION}.tar.gz" -C ./ .
