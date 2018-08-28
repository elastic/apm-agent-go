#!/usr/bin/env bash

set -ex

# TODO(axw) use master branch when v2 is merged
BRANCH=v2

FILES=( \
    "errors/error.json" \
    "errors/payload.json" \
    "sourcemaps/payload.json" \
    "spans/span.json" \
    "transactions/mark.json" \
    "transactions/payload.json" \
    "transactions/transaction.json" \
    "metrics/payload.json" \
    "metrics/metric.json" \
    "metrics/sample.json" \
    "context.json" \
    "metadata.json" \
    "process.json" \
    "request.json" \
    "service.json" \
    "stacktrace_frame.json" \
    "system.json" \
    "user.json" \
)

mkdir -p jsonschema/errors jsonschema/transactions jsonschema/sourcemaps jsonschema/spans jsonschema/metrics

for i in "${FILES[@]}"; do
  o=jsonschema/$i
  curl -sf https://raw.githubusercontent.com/elastic/apm-server/${BRANCH}/docs/spec/${i} --compressed -o $o
done
