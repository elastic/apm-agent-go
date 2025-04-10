---
mapped_pages:
  - https://www.elastic.co/guide/en/apm/agent/go/current/introduction.html
  - https://www.elastic.co/guide/en/apm/agent/go/current/index.html
---

# APM Go agent [introduction]

The Elastic APM Go Agent enables you to trace the execution of operations in your [Go](https://golang.org/) applications, sending performance metrics and errors to the Elastic APM server. It has built-in support for popular frameworks and toolkits, like [Gorilla](http://www.gorillatoolkit.org/) and [Gin](https://gin-gonic.com/), as well as support for instrumenting Go’s built-in [net/http](https://golang.org/pkg/net/http/), [database/sql](https://golang.org/pkg/database/sql/) drivers. The Agent also offers an [*API Documentation*](/reference/api-documentation.md) for custom instrumentation.


## How does the Agent work? [how-it-works]

The Agent includes instrumentation modules for [*Supported Technologies*](/reference/supported-technologies.md), each providing middleware or wrappers for recording interesting events, such as incoming HTTP requests, outgoing HTTP requests, and database queries.

To collect data about incoming HTTP requests, install router middleware for one of the supported [Web Frameworks](/reference/supported-technologies.md#supported-tech-web-frameworks). Incoming requests will be recorded as transactions, along with any related panics or errors.

To collect data for outgoing HTTP requests, instrument an `http.Client` or `http.Transport` using [module/apmhttp](/reference/builtin-modules.md#builtin-modules-apmhttp). To collect data about database queries, use [module/apmsql](/reference/builtin-modules.md#builtin-modules-apmsql), which provides instrumentation for well known database drivers.

In order to connect transactions with related spans and errors, and propagate traces between services (distributed tracing), the agent relies on Go’s built-in [context](https://golang.org/pkg/context/) package: transactions and spans are stored in context objects. For example, for incoming HTTP requests, in-flight trace data will be recorded in the `context` object accessible through [net/http.Context](https://golang.org/pkg/net/http/#Request.Context). Read more about this in [Context propagation](/reference/custom-instrumentation-propagation.md).

In addition to capturing events like those mentioned above, the agent also collects system and application metrics at regular intervals. This collection happens in a background goroutine that is automatically started when the agent is initialized.


## Additional Components [additional-components]

APM Agents work in conjunction with the [APM Server](docs-content://solutions/observability/apm/index.md), [Elasticsearch](docs-content://get-started/index.md), and [Kibana](docs-content://get-started/the-stack.md). The [APM Guide](docs-content://solutions/observability/apm/index.md) provides details on how these components work together, and provides a matrix outlining [Agent and Server compatibility](docs-content://solutions/observability/apm/apm-agent-compatibility.md).

