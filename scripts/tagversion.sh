#!/bin/bash
set -e

SED=sed
if [[ $(uname -s) == "Darwin" ]]; then SED=gsed; fi

prefix=go.elastic.co/apm
version=$(${SED} 's@^\s*AgentVersion = "\(.*\)"$@\1@;t;d' version.go)
major_version=$(echo $version | cut -d. -f1)
modules=$(for dir in $(./scripts/moduledirs.sh); do (cd $dir && go list -m); done | grep ${prefix})

echo "# Create tags"
for m in $modules; do
	unversioned=$(echo $m | ${SED} "s@^${prefix}/\(.*\)@\1@" | $SED "s@/\?v${major_version}\$@@")
	tag=$(echo "${unversioned}/v${version}" | $SED "s@^/@@")
	echo git tag -s ${tag} -m v${version}
done

echo
echo "# Push tags"
echo -n git push upstream
for m in $modules; do
	unversioned=$(echo $m | ${SED} "s@^${prefix}/\(.*\)@\1@" | $SED "s@/\?v${major_version}\$@@")
	tag=$(echo "${unversioned}/v${version}" | $SED "s@^/@@")
	echo -n " ${tag}"
done
echo
