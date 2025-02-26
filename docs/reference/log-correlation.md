---
mapped_pages:
  - https://www.elastic.co/guide/en/apm/agent/go/current/log-correlation-ids.html
---

# Log correlation [log-correlation-ids]

[Log correlation](docs-content://solutions/observability/apps/logs.md) allows you to navigate to all logs belonging to a particular trace and vice-versa: for a specific log, see in which context it has been logged and which parameters the user provided.

In order to correlate logs from your app with transactions captured by the Elastic APM Go Agent, your logs must contain one or more of the following identifiers:

* [`transaction.id`](ecs://docs/reference/ecs-tracing.md)
* [`trace.id`](ecs://docs/reference/ecs-tracing.md)
* [`span.id`](ecs://docs/reference/ecs-error.md)

In order to correlate the logs to the service and environment, the logs should also contain the following fields:

* [`service.name`](ecs://docs/reference/ecs-service.md)
* [`service.version`](ecs://docs/reference/ecs-service.md)
* [`service.environment`](ecs://docs/reference/ecs-service.md)


## Manual log correlation [log-correlation-manual]

If the agent-provided logging integrations are not suitable or not available for your application, then you can use the agent’s [API](/reference/api-documentation.md) to inject trace IDs manually. There are two main approaches you can take, depending on whether you are using structured or unstructured logging.


### Manual log correlation (structured) [log-correlation-manual-structured]

For correlating structured logs with traces and services, the fields defined above should be added to logs.

Given a transaction object, you can obtain its trace ID and transaction ID using the [apm.Transaction.TraceContext](/reference/api-documentation.md#transaction-tracecontext) method. Similarly, given a span object, you can obtain its span ID using [apm.Span.TraceContext](/reference/api-documentation.md#span-tracecontext).

If you use the context APIs to start transactions and spans, then you can obtain the context’s current transaction using [apm.TransactionFromContext](/reference/api-documentation.md#apm-transaction-from-context), and current span using [apm.SpanFromContext](/reference/api-documentation.md#apm-span-from-context). Note that if a transaction is not sampled, `apm.TransactionFromContext` will return `nil`. Similarly, spans may be dropped by the agent, so `apm.SpanFromContext` may also return `nil`.

```go
labels := make(map[string]string)
tx := apm.TransactionFromContext(ctx)
if tx != nil {
	traceContext := tx.TraceContext()
	labels["trace.id"] = traceContext.Trace.String()
	labels["transaction.id"] = traceContext.Span.String()
	if span := apm.SpanFromContext(ctx); span != nil {
		labels["span.id"] = span.TraceContext().Span
	}
}
```

Follow this article to ingest JSON-encoded logs with Filebeat: [How to instrument your Go app with the Elastic APM Go agent](https://www.elastic.co/blog/how-to-instrument-your-go-app-with-the-elastic-apm-go-agent#logs).


### Manual log correlation (unstructured) [log-correlation-manual-unstructured]

For correlating unstructured logs (e.g. basic printf-style logging, like the standard library’s `log` package), then you will need to need to include the trace IDs in your log message. Then, extract them using Filebeat.

If you already have a transaction or span object, use the [Transaction.TraceContext](/reference/api-documentation.md#transaction-tracecontext) or [Span.TraceContext](/reference/api-documentation.md#span-tracecontext) methods. The trace, transaction, and span ID types all provide `String` methods that yield their canonical hex-encoded string representation.

```go
traceContext := tx.TraceContext()
spanID := span.TraceContext().Span
log.Printf("ERROR [trace.id=%s transaction.id=%s span.id=%s] an error occurred", traceContext.Trace, traceContext.Span, spanID)
```

If instead you are dealing with context objects, you may prefer to use the [TraceFormatter](/reference/api-documentation.md#apm-traceformatter) function. For example, you could supply it as an argument to `log.Printf` as follows:

```go
log.Printf("ERROR [%+v] an error occurred", apm.TraceFormatter(ctx))
```

This would print a log message along the lines of:

```
2019/09/17 14:48:02 ERROR [trace.id=cd04f33b9c0c35ae8abe77e799f126b7 transaction.id=cd04f33b9c0c35ae span.id=960834f4538880a4] an error occurred
```
For log correlation to work, the trace IDs must be extracted from the log message and stored in separate fields in the Elasticsearch document. This can be achieved by [using an ingest pipeline to parse the data](beats://docs/reference/filebeat/configuring-ingest-node.md), in particular by using [the grok processor](elasticsearch://docs/reference/ingestion-tools/enrich-processor/grok-processor.md).

```json
{
  "description": "...",
  "processors": [
    {
      "grok": {
        "field": "message",
        "patterns": ["%{YEAR}/%{MONTHNUM}/%{MONTHDAY} %{TIME} %{LOGLEVEL:log.level} \\[trace.id=%{TRACE_ID:trace.id}(?: transaction.id=%{SPAN_ID:transaction.id})?(?: span.id=%{SPAN_ID:span.id})?\\] %{GREEDYDATA:message}"],
        "pattern_definitions": {
          "TRACE_ID": "[0-9A-Fa-f]{32}",
          "SPAN_ID": "[0-9A-Fa-f]{16}"
        }
      }
    }
  ]
}
```

