#!/usr/bin/env bash

set -ex

BRANCH=master

FILES=( \
    "errors/error.json" \
    "sourcemaps/payload.json" \
    "spans/span.json" \
    "transactions/mark.json" \
    "transactions/transaction.json" \
    "metricsets/metricset.json" \
    "metricsets/sample.json" \
    "context.json" \
    "cloud.json" \
    "http_response.json" \
    "message.json" \
    "metadata.json" \
    "outcome.json" \
    "process.json" \
    "request.json" \
    "rum_experience.json" \
    "service.json" \
    "span_subtype.json" \
    "span_type.json" \
    "stacktrace_frame.json" \
    "system.json" \
    "tags.json" \
    "timestamp_epoch.json" \
    "transaction_name.json" \
    "transaction_type.json" \
    "user.json" \
)

mkdir -p jsonschema/errors jsonschema/transactions jsonschema/sourcemaps jsonschema/spans jsonschema/metricsets

for i in "${FILES[@]}"; do
  o=jsonschema/$i
  curl -sf https://raw.githubusercontent.com/elastic/apm-server/${BRANCH}/docs/spec/${i} --compressed -o $o
done
