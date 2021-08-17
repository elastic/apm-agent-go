#!/usr/bin/env bash
set -e

for dir in $(scripts/moduledirs.sh); do
    (
      cd $dir
      # Before go1.13, there was a bug that prevented the toolchain from
      # finding a module when no buildable files were present.
      # In go1.14 this is detected and the message states that build
      # constraints exclude all Go files.
      go list -json 2>&1 | grep -E 'build constraints exclude|cannot find module' > /dev/null
      [ "$?" = "0" ] && echo "skipping $dir: no buildable Go files" && exit 0
      go vet ./...
    ) || exit $?
done
