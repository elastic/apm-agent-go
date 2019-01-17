#!/bin/sh
set -e

export GO111MODULE=on

prefix=go.elastic.co/apm
version=$(sed 's@^\s*AgentVersion = "\(.*\)"$@\1@;t;d' version.go)
modules=$(for dir in $(./scripts/moduledirs.sh); do (cd $dir && go list -m); done | grep $prefix/)

echo "# Create tags"
for m in "" $modules; do
	p=$(echo $m | sed "s@^$prefix/\(.\{0,\}\)@\1/@")
	echo git tag -s ${p}v$version -m v$version
done

echo
echo "# Push tags"
echo -n git push upstream
for m in "" $modules; do
	p=$(echo $m | sed "s@^$prefix/\(.\{0,\}\)@\1/@")
	echo -n " ${p}v$version"
done
echo
