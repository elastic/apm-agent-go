#!/usr/bin/env bash

set -ex

# TODO(axw) use master branch when v2 is merged
BRANCH=v2

FILES=( \
    "errors/common_error.json" \
    "errors/v2_error.json" \
    "sourcemaps/payload.json" \
    "spans/common_span.json" \
    "spans/v2_span.json" \
    "transactions/mark.json" \
    "transactions/common_transaction.json" \
    "transactions/v2_transaction.json" \
    "metrics/metricset.json" \
    "metrics/sample.json" \
    "context.json" \
    "metadata.json" \
    "process.json" \
    "request.json" \
    "service.json" \
    "stacktrace_frame.json" \
    "system.json" \
    "tags.json" \
    "user.json" \
)

mkdir -p jsonschema/errors jsonschema/transactions jsonschema/sourcemaps jsonschema/spans jsonschema/metrics

for i in "${FILES[@]}"; do
  o=jsonschema/$i
  curl -sf https://raw.githubusercontent.com/elastic/apm-server/${BRANCH}/docs/spec/${i} --compressed -o $o
done
