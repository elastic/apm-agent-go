---
navigation_title: "Elastic APM Go Agent"
mapped_pages:
  - https://www.elastic.co/guide/en/apm/agent/go/current/release-notes-2.x.html
  - https://www.elastic.co/guide/en/apm/agent/go/current/release-notes.html
---

# Elastic APM Go Agent release notes [elastic-apm-go-agent-release-notes]

Review the changes, fixes, and more in each version of Elastic APM Go Agent.

To check for security updates, go to [Security announcements for the Elastic stack](https://discuss.elastic.co/c/announcements/security-announcements/31).

% Release notes includes only features, enhancements, and fixes. Add breaking changes, deprecations, and known issues to the applicable release notes sections.

% version.next [elastic-apm-go-agent-versionext-release-notes]
% **Release date:** Month day, year

% ### Features and enhancements [elastic-apm-go-agent-versionext-features-enhancements]

% ### Fixes [elastic-apm-go-agent-versionext-fixes]

## 2.7.0
**Release date:** March 13, 2025

_No new features, enhancements, and fixes_

## 2.6.3 [elastic-apm-go-agent-2-6-3-release-notes]
**Release date:** January 13, 2025

### Fixes [elastic-apm-go-agent-2-6-3-fixes]
* Avoid panic when unwrapping errors

## 2.6.2 [elastic-apm-go-agent-2-6-2-release-notes]
**Release date:** August 29, 2025

### Features and enhancements [elastic-apm-go-agent-2-6-2-features-enhancements]
* Update version string

## 2.6.1 [elastic-apm-go-agent-2-6-1-release-notes]
**Release date:** August 29, 2024

### Features and enhancements [elastic-apm-go-agent-2-6-1-features-enhancements]
* Support all upstream GOOS [#1646](https://github.com/elastic/apm-agent-go/pull/1646)

### Fixes [elastic-apm-go-agent-2-6-1-fixes]
* apm.DefaultTracer misbehaves when transport configuration is invalid by [#1618](https://github.com/elastic/apm-agent-go/pull/1618)
* gin web framework does not properly sanitize filename parameter of Context.FileAttachment function [#1620](https://github.com/elastic/apm-agent-go/pull/1620)

## 2.6.0 [elastic-apm-go-agent-2-6-0-release-notes]
**Release date:** April 11, 2024

### Features and enhancements [elastic-apm-go-agent-2-6-0-features-enhancements]
* Bump minimum Go version to 1.21 [#1602](https://github.com/elastic/apm-agent-go/pull/1602)

### Fixes [elastic-apm-go-agent-2-6-0-fixes]
* module/apmotel: fix compatibility issue with newer version of otel libs. [#1605](https://github.com/elastic/apm-agent-go/pull/1605)

## 2.5.0 [elastic-apm-go-agent-2-5-0-release-notes]
**Release date:** March 12, 2024

### Features and enhancements [elastic-apm-go-agent-2-5-0-features-enhancements]
* module/apmgorm: Switch from `github.com/denisenkom/go-mssqldb` package to `github.com/microsoft/go-mssqldb`. [#1569](https://github.com/elastic/apm-agent-go/pull/1569)
* module/apmrestful: Upgrade `github.com/emicklei/go-restful` package to `gituhub.co/emicklei/go-restful/v3`. [#1580](https://github.com/elastic/apm-agent-go/pull/1580)

## 2.4.8 [elastic-apm-go-agent-2-4-8-release-notes]
**Release date:** March 12, 2024

### Features and enhancements [elastic-apm-go-agent-2-4-8-features-enhancements]
* module/apmotel: Add nil and recording check to span.RecordError [#1566](https://github.com/elastic/apm-agent-go/pull/1566)

## 2.4.7 [elastic-apm-go-agent-2-4-7-release-notes]
**Release date:** November 23, 2024

### Features and enhancements [elastic-apm-go-agent-2-4-7-features-enhancements]
* Bump submodule dependency version [#1546](https://github.com/elastic/apm-agent-go/pull/1546)

## 2.4.6 [elastic-apm-go-agent-2-4-6-release-notes]
**Release date:** November 22, 2023

### Fixes [elastic-apm-go-agent-2-4-6-fixes]
* module/apmotel: Fix compatibility issue with newer version of otel [#1544](https://github.com/elastic/apm-agent-go/pull/1544)

## 2.4.5 [elastic-apm-go-agent-2-4-5-release-notes]
**Release date:** October 11, 2023

### Fixes [elastic-apm-go-agent-2-4-5-fixes]
* module/apmotel: Fix panic on multiple span close calls [#1512](https://github.com/elastic/apm-agent-go/pull/1512)

## 2.4.4 [elastic-apm-go-agent-2-4-4-release-notes]
**Release date:** August 29, 2023

### Features and enhancements [elastic-apm-go-agent-2-4-4-features-enhancements]
* module/apmotel: Bumped minimum OpenTelemetry version [#1501](https://github.com/elastic/apm-agent-go/pull/1501)
* module/apmotel: Return usable spans when retrieving them from otel.SpanFromContext [#1478](https://github.com/elastic/apm-agent-go/pull/1478)

### Fixes [elastic-apm-go-agent-2-4-4-fixes]
* Fixed concurrent map write condition where some child spans couldn’t acquire the transaction lock [#1487](https://github.com/elastic/apm-agent-go/pull/1487)

## 2.4.3 [elastic-apm-go-agent-2-4-3-release-notes]
**Release date:** June 22, 2023

### Features and enhancements [elastic-apm-go-agent-2-4-3-features-enhancements]
* Bumped minimum Go version to 1.19 [#1453](https://github.com/elastic/apm-agent-go/pull/1453)
* Updated to stable OTel metrics API [#1448](https://github.com/elastic/apm-agent-go/pull/1448)

### Fixes [elastic-apm-go-agent-2-4-3-fixes]
* Fixed a data race in HTTP client instrumentation [#1472](https://github.com/elastic/apm-agent-go/pull/1472)
* Fixed mixing of OTel and Elastic APM instrumentation [#1450](https://github.com/elastic/apm-agent-go/pull/1450)

## 2.4.2 [elastic-apm-go-agent-2-4-2-release-notes]
**Release date:** May 22, 2023

### Features and enhancements [elastic-apm-go-agent-2-4-2-features-enhancements]
* module/apmotel: handle resources [#1424](https://github.com/elastic/apm-agent-go/pull/1424)
* Drop x/net dependency [#1434](https://github.com/elastic/apm-agent-go/pull/1434)
* module/apmotel: bump go.opentelemetry.io/otel/metric [#1435](https://github.com/elastic/apm-agent-go/pull/1435)
* module/apmotel: follow APM OTel spec and prefer delta temporality [#1437](https://github.com/elastic/apm-agent-go/pull/1437)
* module/apmotel: set the proper trace ID and span ID in trace context [#1438](https://github.com/elastic/apm-agent-go/pull/1438)
* module/apmotel: handle context flags when creating remote transactions and spans [#1441](https://github.com/elastic/apm-agent-go/pull/1441)

## 2.4.1 [elastic-apm-go-agent-2-4-1-release-notes]
**Release date:** April 27, 2023

### Features and enhancements [elastic-apm-go-agent-2-4-1-features-enhancements]
* Downgrade OpenTelemetry metrics from v1.15.0-rc.2 to 0.37.0 [#1420](https://github.com/elastic/apm-agent-go/pull/1420)
* Mark OpenTelemetry metrics as technical preview [#1419](https://github.com/elastic/apm-agent-go/pull/1419)

## 2.4.0 [elastic-apm-go-agent-2-4-0-release-notes]
**Release date:** April 26, 2023

### Features and enhancements [elastic-apm-go-agent-2-4-0-features-enhancements]
* Add bridge to support OpenTelemetry metrics [#1407](https://github.com/elastic/apm-agent-go/pull/1407)
* Add custom SDK support OpenTelemetry traces [#1410](https://github.com/elastic/apm-agent-go/pull/1410)

## 2.3.0 [elastic-apm-go-agent-2-3-0-release-notes]
**Release date:** March 30, 2023

### Features and enhancements [elastic-apm-go-agent-2-3-0-features-enhancements]
* Ensure minimum retry interval of 5 seconds for fetching central configuration [#1337](https://github.com/elastic/apm-agent-go/pull/1337)
* Update span compression logic to handle `service.target.*` fields [#1339](https://github.com/elastic/apm-agent-go/pull/1339)
* module/apmchiv5: Add panic propogation option [#1359](https://github.com/elastic/apm-agent-go/pull/1359)
* module/apmgormv2: Add sqlserver support [#1356](https://github.com/elastic/apm-agent-go/pull/1356)
* module/apmsql: Add sqlserver support [#1356](https://github.com/elastic/apm-agent-go/pull/1356)
* Update compressed spans to use `service.target.*` fields to derive its name [#1336](https://github.com/elastic/apm-agent-go/pull/1336)
* module/apmpgxv5: new instrumentation module for jackc/pgx v5 with enhanced support e.g. detailed `BATCH` and `CONNECT` traces [#1364](https://github.com/elastic/apm-agent-go/pull/1364)
* Add support for `Unwrap []error` [#1400](https://github.com/elastic/apm-agent-go/pull/1400)

## 2.2.0 [elastic-apm-go-agent-2-2-0-release-notes]
**Release date:** October 31, 2022

### Features and enhancements [elastic-apm-go-agent-2-2-0-features-enhancements]
* Global labels are now parsed when the tracer is constructed, instead of parsing only once on package initialization [#1290](https://github.com/elastic/apm-agent-go/pull/1290)
* Rename span_frames_min_duration to span_stack_trace_min_duration [#1285](https://github.com/elastic/apm-agent-go/pull/1285)
* Ignore `\*principal\*` headers by default [#1332](https://github.com/elastic/apm-agent-go/pull/1332)
* Add `apmpgx` module for postgres tracing with jackc/pgx driver enhanced support e.g. Copy and Batch statements [#1301](https://github.com/elastic/apm-agent-go/pull/1301)
* Disable same-kind and enable exact-match compression by default [#1256](https://github.com/elastic/apm-agent-go/pull/1256)
* module/apmechov4: add `WithRequestName` option [#1268](https://github.com/elastic/apm-agent-go/pull/1268)
* Added support for adding span links when starting transactions and spans [#1269](https://github.com/elastic/apm-agent-go/pull/1269)
* Added support for the `trace_continuation_strategy` [#1270](https://github.com/elastic/apm-agent-go/pull/1270)
* `transaction.type` and `span.type` are now set to "custom" if an empty string is specified [#1272](https://github.com/elastic/apm-agent-go/pull/1272)
* We now capture the database instance name in `service.target.*`, for improved backend granularity [#1279](https://github.com/elastic/apm-agent-go/pull/1279)
* Improved Kubernetes pod UID and container ID discovery coverage [#1288](https://github.com/elastic/apm-agent-go/pull/1288)
* module/apmgin: add `WithPanicPropagation` option [#1314](https://github.com/elastic/apm-agent-go/pull/1314)
* Exit spans may now have non-exit child spans if they have the same type and subtype [#1320](https://github.com/elastic/apm-agent-go/pull/1320)
* Updated instrumentation modules to mark spans as exit spans where possible [#1317](https://github.com/elastic/apm-agent-go/pull/1317)

### Fixes [elastic-apm-go-agent-2-2-0-fixes]
* module/apmawssdkgo: fixed a panic related to drop spans [#1273](https://github.com/elastic/apm-agent-go/pull/1273)
* Fixed `span.name` for AWS SNS spans to match the spec [#1286](https://github.com/elastic/apm-agent-go/pull/1286)

## 2.1.0 [elastic-apm-go-agent-2-1-0-release-notes]
**Release date:** May 20, 2022

### Features and enhancements [elastic-apm-go-agent-2-1-0-features-enhancements]
* Replace `authorization` with `*auth*` pattern for sanitizing field names [#1230](https://github.com/elastic/apm-agent-go/pull/1230)
* Fetch initial server version async to prevent blocking NewTracer for 10 seconds [#1239](https://github.com/elastic/apm-agent-go/pull/1239)

### Fixes [elastic-apm-go-agent-2-1-0-fixes]
* Fix race in `apm.DefaultTracer` which could lead to multiple tracers being created [#1248](https://github.com/elastic/apm-agent-go/pull/1248)

## 2.0.0 [elastic-apm-go-agent-2-0-0-release-notes]
**Release date:** March 17, 2022

### Features and enhancements [elastic-apm-go-agent-2-0-0-features-enhancements]
* Record `transaction.name` on errors [#1177](https://github.com/elastic/apm-agent-go/pull/1177)
* Stop recording unused `transaction.duration.*` and `transaction.breakdown.count` metrics [#1167](https://github.com/elastic/apm-agent-go/pull/1167)
* Make tracestate parsing more lenient, according to W3c spec, allowing duplicate vendor keys [#1183](https://github.com/elastic/apm-agent-go/pull/1183)
* Introduced `transport.NewHTTPTransportOptions` [#1168](https://github.com/elastic/apm-agent-go/pull/1168)
* Change `ELASTIC_APM_SPAN_FRAMES_MIN_DURATION` special cases to match agent spec [#1188](https://github.com/elastic/apm-agent-go/pull/1188)
* Remove stacktrace.ContextSetter [#1187](https://github.com/elastic/apm-agent-go/pull/1187)
* Drop support for versions of Go prior to 1.15.0 [#1190](https://github.com/elastic/apm-agent-go/pull/1190)
* Replace apm.DefaultTracer with an initialization function [#1189](https://github.com/elastic/apm-agent-go/pull/1189)
* Remove transport.Default, construct a new Transport in each new tracer [#1195](https://github.com/elastic/apm-agent-go/pull/1195)
* Add service name and version to User-Agent header [#1196](https://github.com/elastic/apm-agent-go/pull/1196)
* Remove WarningLogger, add Warningf methe to Logger [#1205](https://github.com/elastic/apm-agent-go/pull/1205)
* Replace Sampler with ExtendedSampler [#1206](https://github.com/elastic/apm-agent-go/pull/1206)
* Drop unsampled txs when connected to an APM Server >= 8.0 [#1208](https://github.com/elastic/apm-agent-go/pull/1208)
* Removed SetTag [#1218](https://github.com/elastic/apm-agent-go/pull/1218)
* Unexport Tracer’s fields — TracerOptions must be used instead [#1219](https://github.com/elastic/apm-agent-go/pull/1219)

### Fixes [elastic-apm-go-agent-2-0-0-fixes]
* Fix panic in apmgocql [#1180](https://github.com/elastic/apm-agent-go/pull/1180)
