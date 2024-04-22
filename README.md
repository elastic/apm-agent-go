[![Build Status](https://github.com/elastic/apm-agent-go/actions/workflows/ci.yml/badge.svg)](https://github.com/elastic/apm-agent-go/actions/workflows/ci.yml)
[![PkgGoDev](https://pkg.go.dev/badge/go.elastic.co/apm/v2)](https://pkg.go.dev/go.elastic.co/apm/v2)
[![Go Report Card](https://goreportcard.com/badge/go.elastic.co/apm/v2)](https://goreportcard.com/report/go.elastic.co/apm/v2)
[![codecov.io](https://codecov.io/github/elastic/apm-agent-go/coverage.svg?branch=main)](https://codecov.io/github/elastic/apm-agent-go?branch=main)

**NOTE**: This repository is in maintenance mode. Bug fixes will continue to be
applied, but no further development will take place. To replace this agent, we
recommend you to [migrate to the OpenTelemetry Go API and
SDK](https://www.elastic.co/blog/elastic-go-apm-agent-to-opentelemetry-go-sdk),
which provides similar features. In order to help you do a seamless migration,
we recommend using our [OpenTelemetry
Bridge](https://www.elastic.co/guide/en/apm/agent/go/current/opentelemetry.html).
Please refer to the blog post above for further details.

# apm-agent-go: APM Agent for Go

This is the official Go package for [Elastic APM](https://www.elastic.co/solutions/apm).

The Go agent enables you to trace the execution of operations in your application,
sending performance metrics and errors to the Elastic APM server.

## Installation

Within a Go module:

```bash
go get go.elastic.co/apm/v2
```

## Requirements

Requires [APM Server](https://github.com/elastic/apm-server) v6.5 or newer.

You can find a list of the supported frameworks and other technologies in the
[documentation](https://www.elastic.co/guide/en/apm/agent/go/current/supported-tech.html).

## License

Apache 2.0.

## Documentation

[Elastic APM Go documentation](https://www.elastic.co/guide/en/apm/agent/go/current/index.html).

## Getting Help

If you find a bug, please [report an issue](https://github.com/elastic/apm-agent-go/issues).
For any other assistance, please open or add to a topic on the [APM discuss forum](https://discuss.elastic.co/c/apm).
