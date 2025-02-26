---
mapped_pages:
  - https://www.elastic.co/guide/en/apm/agent/go/current/custom-instrumentation.html
---

# Custom instrumentation [custom-instrumentation]

To report on the performance of transactions served by your application, use the Go agent’s [API](/reference/api-documentation.md). Instrumentation refers to modifying your application code to report a:

* [Transaction](#custom-instrumentation-transactions) - A top-level operation in your application, such as an HTTP or RPC request.
* [Span within a transaction](#custom-instrumentation-spans) - An operation within a transaction, such as a database query, or a request to another service.
* [Error](#custom-instrumentation-errors) - May refer to Go errors or panics.

To report these, use a [apm.Tracer](/reference/api-documentation.md#tracer-api) — typically `apm.DefaultTracer()`, which is configured via environment variables. In the code examples below, we will refer to `apm.DefaultTracer()`. Please refer to the [API documentation](/reference/api-documentation.md) for a more thorough description of the types and methods.

## Transactions [custom-instrumentation-transactions]

To report a transaction, call [apm.DefaultTracer().StartTransaction](/reference/api-documentation.md#tracer-api-start-transaction) with the transaction name and type. This returns a `Transaction` object; the transaction can be customized with additional context before you call its `End` method to indicate that the transaction has completed. Once the transaction’s `End` method is called, it will be enqueued for sending to the Elastic APM server, and made available to the APM app.

```go
tx := apm.DefaultTracer().StartTransaction("GET /api/v1", "request")
defer tx.End()
...
tx.Result = "HTTP 2xx"
tx.Context.SetLabel("region", "us-east-1")
```

The agent supports sampling transactions: non-sampled transactions will be still be reported, but with limited context and without any spans. To determine whether a transaction is sampled, use the `Transaction.Sampled` method; if it returns false, you should avoid unnecessary storage or processing required for setting transaction context.

Once you have started a transaction, you can include it in a `context` object for propagating throughout the application. See [context propagation](/reference/custom-instrumentation-propagation.md) for more details.

```go
ctx = apm.ContextWithTransaction(ctx, tx)
```


## Spans [custom-instrumentation-spans]

To report an operation within a transaction, use [Transaction.StartSpan](/reference/api-documentation.md#transaction-start-span) or [apm.StartSpan](/reference/api-documentation.md#apm-start-span) to start a span given a transaction or a `context` containing a transaction, respectively. Like a transaction, a span has a name and a type. A span can have a parent span within the same transaction. If the context provided to `apm.StartSpan` contains a span, then that will be considered the parent. See [context propagation](/reference/custom-instrumentation-propagation.md) for more details.

```go
span, ctx := apm.StartSpan(ctx, "SELECT FROM foo", "db.mysql.query")
defer span.End()
```

`Transaction.StartSpan` and `apm.StartSpan` will always return a non-nil `Span`, even if the transaction is nil. It is always safe to defer a call to the span’s End method. If setting the span’s context would incur significant overhead, you may want to check if the span is dropped first, by calling the `Span.Dropped` method.


## Panic recovery and errors [custom-instrumentation-errors]

To recover panics and report them along with your transaction, use the [Tracer.Recovered](/reference/api-documentation.md#tracer-recovered) method in a recovery function. There are also methods for reporting non-panic errors: [Tracer.NewError](/reference/api-documentation.md#tracer-new-error), [Tracer.NewErrorLog](/reference/api-documentation.md#tracer-new-error-log), and [apm.CaptureError](/reference/api-documentation.md#apm-captureerror).

```go
defer func() {
	if v := recover(); v != nil {
		e := apm.DefaultTracer().Recovered()
		e.SetTransaction(tx) // or e.SetSpan(span)
		e.Send()
	}
}()
```

See the [Error API](/reference/api-documentation.md#error-api) for details and examples of the other methods.


