[![GoDoc](https://godoc.org/github.com/elastic/apm-agent-go?status.svg)](http://godoc.org/github.com/elastic/apm-agent-go)
[![Travis-CI](https://travis-ci.org/elastic/apm-agent-go.svg)](https://travis-ci.org/elastic/apm-agent-go)
[![AppVeyor](https://ci.appveyor.com/api/projects/status/28fhswvqqc7p90f7?svg=true)](https://ci.appveyor.com/project/AndrewWilkins/apm-agent-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/elastic/apm-agent-go)](https://goreportcard.com/report/github.com/elastic/apm-agent-go)
[![codecov.io](https://codecov.io/github/elastic/apm-agent-go/coverage.svg?branch=master)](https://codecov.io/github/elastic/apm-agent-go?branch=master)

# apm-agent-go: APM Agent for Go (pre-alpha)

This is the official Go package for [Elastic APM](https://www.elastic.co/solutions/apm).

The Go agent enables you to trace the execution of operations in your application,
sending performance metrics and errors to the Elastic APM server.

This repository is in a pre-alpha state and under heavy development.
Do not deploy into production!

## Installation

```bash
go get -u github.com/elastic/apm-agent-go
```

## Requirements

Tested with Go 1.8+ on Linux, Windows and MacOS.

## License

Apache 2.0.

## Documentation

[Elastic APM Go documentation](./docs/index.asciidoc).

## Getting Help

If you find a bug, please [report an issue](https://github.com/elastic/apm-agent-go/issues).
For any other assistance, please open or add to a topic on the [APM discuss forum](https://discuss.elastic.co/c/apm).
