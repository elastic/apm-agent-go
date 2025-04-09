---
mapped_pages:
  - https://www.elastic.co/guide/en/apm/agent/go/current/api.html
---

# API documentation [api]

This section describes the most commonly used parts of the API.

The Go agent is documented using standard godoc. For complete documentation, refer to the documentation at [pkg.go.dev/go.elastic.co/apm/v2](https://pkg.go.dev/go.elastic.co/apm/v2), or by using the "godoc" tool.


## Tracer API [tracer-api]

The initial point of contact your application will have with the Go agent is the `apm.Tracer` type, which provides methods for reporting transactions and errors.

To make instrumentation simpler, the Go agent provides an initialization function, `apm.DefaultTracer()`. This tracer is initialized the first time `apm.DefaultTracer()` is called, and returned on subsequent calls. The tracer returned by this function can be modified using `apm.SetDefaultTracer(tracer)`. Calling this will close the previous default tracer, if any exists.  This tracer is configured with environment variables; see [*Configuration*](/reference/configuration.md) for details.

```go
import (
	"go.elastic.co/apm/v2"
)

func main() {
	tracer := apm.DefaultTracer()
	...
}
```


## Transactions [transaction-api]


### `func (*Tracer) StartTransaction(name, type string) *Transaction` [tracer-api-start-transaction]

StartTransaction returns a new Transaction with the specified name and type, and with the start time set to the current time. If you need to set the timestamp or the parent [trace context](#trace-context), use [Tracer.StartTransactionOptions](#tracer-api-start-transaction-options).

This method should be called at the beginning of a transaction such as a web or RPC request. e.g.:

```go
transaction := apm.DefaultTracer().StartTransaction("GET /", "request")
```

Transactions will be grouped by type and name in the Elastic APM app.

After starting a transaction, you can record a result and add context to further describe the transaction.

```go
transaction.Result = "Success"
transaction.Context.SetLabel("region", "us-east-1")
```

See [Context](#context-api) for more details on setting transaction context.


### `func (*Tracer) StartTransactionOptions(name, type string, opts TransactionOptions) *Transaction` [tracer-api-start-transaction-options]

StartTransactionOptions is essentially the same as StartTransaction, but also accepts an options struct. This struct allows you to specify the parent [trace context](#trace-context) and/or the transaction’s start time.

```go
opts := apm.TransactionOptions{
	Start: time.Now(),
	TraceContext: parentTraceContext,
}
transaction := apm.DefaultTracer().StartTransactionOptions("GET /", "request", opts)
```


### `func (*Transaction) End()` [transaction-end]

End enqueues the transaction for sending to the Elastic APM server. The transaction must not be modified after this, but it may still be used for starting spans.

The transaction’s duration is calculated as the amount of time elapsed between the start of the transaction and this call. To override this behavior, the transaction’s `Duration` field may be set before calling End.

```go
transaction.End()
```


### `func (*Transaction) TraceContext() TraceContext` [transaction-tracecontext]

TraceContext returns the transaction’s [trace context](#trace-context).


### `func (*Transaction) EnsureParent() SpanID` [transaction-ensureparent]

EnsureParent returns the transaction’s parent span ID—​generating and recording one if it did not previously have one.

EnsureParent enables correlation with spans created by the JavaScript Real User Monitoring (RUM) agent for the initial page load. If your backend service generates the HTML page dynamically, you can inject the trace and parent span ID into the page in order to initialize the JavaScript RUM agent, such that the web browser’s page load appears as the root of the trace.

```go
var initialPageTemplate = template.Must(template.New("").Parse(`
<html>
<head>
<script src="elastic-apm-js-base/dist/bundles/elastic-apm-js-base.umd.min.js"></script>
<script>
  elasticApm.init({
    serviceName: '',
    serverUrl: 'http://localhost:8200',
    pageLoadTraceId: {{.TraceContext.Trace}},
    pageLoadSpanId: {{.EnsureParent}},
    pageLoadSampled: {{.Sampled}},
  })
</script>
</head>
<body>...</body>
</html>
`))

func initialPageHandler(w http.ResponseWriter, req *http.Request) {
	err := initialPageTemplate.Execute(w, apm.TransactionFromContext(req.Context()))
	if err != nil {
		...
	}
}
```

See the [JavaScript RUM agent documentation](apm-agent-rum-js://reference/index.md) for more information.


### `func (*Transaction) ParentID() SpanID` [transaction-parentid]

ParentID returns the ID of the transaction’s parent, or a zero (invalid) SpanID if the transaction has no parent.


### `func ContextWithTransaction(context.Context, *Transaction) context.Context` [apm-context-with-transaction]

ContextWithTransaction adds the transaction to the context, and returns the resulting context.

The transaction can be retrieved using [apm.TransactionFromContext](#apm-transaction-from-context). The context may also be passed into [apm.StartSpan](#apm-start-span), which uses TransactionFromContext under the covers to create a span as a child of the transaction.


### `func TransactionFromContext(context.Context) *Transaction` [apm-transaction-from-context]

TransactionFromContext returns a transaction previously stored in the context using [apm.ContextWithTransaction](#apm-context-with-transaction), or nil if the context does not contain a transaction.


### `func DetachedContext(context.Context) context.Context` [apm-detached-context]

DetachedContext returns a new context detached from the lifetime of the input, but which still returns the same values as the input.

DetachedContext can be used to maintain trace context required to correlate events, but where the operation is "fire-and-forget" and should not be affected by the deadline or cancellation of the surrounding context.


### `func TraceFormatter(context.Context) fmt.Formatter` [apm-traceformatter]

TraceFormatter returns an implementation of [fmt.Formatter](https://golang.org/pkg/fmt/#Formatter) that can be used to format the IDs of the transaction and span stored in the provided context.

The formatter understands the following formats:

* %v: trace ID, transaction ID, and (if in the context of a span) span ID, space separated
* %t: trace ID only
* %x: transaction ID only
* %s: span ID only

The "+" option can be used to format the values in "key=value" style, with the field names `trace.id`, `transaction.id`, and `span.id`. For example, using "%+v" as the format would yield "trace.id=…​ transaction.id=…​ span.id=…​".

For a more in-depth example, see [Manual log correlation (unstructured)](/reference/log-correlation.md#log-correlation-manual-unstructured).


## Spans [span-api]

To describe an activity within a transaction, we create spans. The Go agent has built-in support for generating spans for some activities, such as database queries. You can use the API to report spans specific to your application.


### `func (*Transaction) StartSpan(name, spanType string, parent *Span) *Span` [transaction-start-span]

StartSpan starts and returns a new Span within the transaction, with the specified name, type, and optional parent span, and with the start time set to the current time. If you need to set the timestamp or parent [trace context](#trace-context), use [Transaction.StartSpanOptions](#transaction-start-span-options).

If the span type contains two dots, they are assumed to separate the span type, subtype, and action; a single dot separates span type and subtype, and the action will not be set.

If the transaction is sampled, then the span’s ID will be set, and its stacktrace will be set if the tracer is configured accordingly. If the transaction is not sampled, then the returned span will be silently discarded when its End method is called. To avoid any unnecessary computation for these dropped spans, call the [Dropped](#span-dropped) method.

As a convenience, it is valid to create a span on a nil Transaction; the resulting span will be non-nil and safe for use, but will not be reported to the APM server.

```go
span := tx.StartSpan("SELECT FROM foo", "db.mysql.query", nil)
```


### `func (*Transaction) StartSpanOptions(name, spanType string, opts SpanOptions) *Span` [transaction-start-span-options]

StartSpanOptions is essentially the same as StartSpan, but also accepts an options struct. This struct allows you to specify the parent [trace context](#trace-context) and/or the spans’s start time. If the parent trace context is not specified in the options, then the span will be a direct child of the transaction. Otherwise, the parent trace context should belong to some span descended from the transaction.

```go
opts := apm.SpanOptions{
	Start: time.Now(),
	Parent: parentSpan.TraceContext(),
}
span := tx.StartSpanOptions("SELECT FROM foo", "db.mysql.query", opts)
```


### `func StartSpan(ctx context.Context, name, spanType string) (*Span, context.Context)` [apm-start-span]

StartSpan starts and returns a new Span within the sampled transaction and parent span in the context, if any. If the span isn’t dropped, it will be included in the resulting context.

```go
span, ctx := apm.StartSpan(ctx, "SELECT FROM foo", "db.mysql.query")
```


### `func (*Span) End()` [span-end]

End marks the span as complete. The Span must not be modified after this, but may still be used as the parent of a span.

The span’s duration will be calculated as the amount of time elapsed since the span was started until this call. To override this behaviour, the span’s Duration field may be set before calling End.


### `func (*Span) Dropped() bool` [span-dropped]

Dropped indicates whether or not the span is dropped, meaning it will not be reported to the APM server. Spans are dropped when the created with a nil, or non-sampled transaction, or one whose max spans limit has been reached.


### `func (*Span) TraceContext() TraceContext` [span-tracecontext]

TraceContext returns the span’s [trace context](#trace-context).


### `func ContextWithSpan(context.Context, *Span) context.Context` [apm-context-with-span]

ContextWithSpan adds the span to the context and returns the resulting context.

The span can be retrieved using [apm.SpanFromContext](#apm-span-from-context). The context may also be passed into [apm.StartSpan](#apm-start-span), which uses SpanFromContext under the covers to create another span as a child of the span.


### `func SpanFromContext(context.Context) *Span` [apm-span-from-context]

SpanFromContext returns a span previously stored in the context using [apm.ContextWithSpan](#apm-context-with-span), or nil if the context does not contain a span.


### `func (*Span) ParentID() SpanID` [span-parentid]

ParentID returns the ID of the span’s parent.


## Context [context-api]

When reporting transactions and errors you can provide context to describe those events. Built-in instrumentation will typically provide some context, e.g. the URL and remote address for an HTTP request. You can also provide custom context and tags.


### `func (*Context) SetLabel(key string, value interface{})` [context-set-label]

SetLabel labels the transaction or error with the given key and value. If the key contains any special characters (`.`, `*`, `"`), they will be replaced with underscores.

If the value is numerical or boolean, then it will be sent to the server as a JSON number or boolean; otherwise it will converted to a string, using `fmt.Sprint` if necessary. Numerical and boolean values are supported by the server from version 6.7 onwards.

String values longer than 1024 characters will be truncated. Labels are indexed in Elasticsearch as keyword fields.

::::{tip}
Before using labels, ensure you understand the different types of [metadata](docs-content://solutions/observability/apm/metadata.md) that are available.
::::


::::{warning}
Avoid defining too many user-specified labels. Defining too many unique fields in an index is a condition that can lead to a [mapping explosion](docs-content://manage-data/data-store/mapping.md#mapping-limit-settings).
::::



### `func (*Context) SetCustom(key string, value interface{})` [context-set-custom]

SetCustom is used to add custom, non-indexed, contextual information to transactions or errors. If the key contains any special characters (`.`, `*`, `"`), they will be replaced with underscores.

Non-indexed means the data is not searchable or aggregatable in Elasticsearch, and you cannot build dashboards on top of the data. However, non-indexed information is useful for other reasons, like providing contextual information to help you quickly debug performance issues or errors.

The value can be of any type that can be encoded using `encoding/json`.

::::{tip}
Before using custom context, ensure you understand the different types of [metadata](docs-content://solutions/observability/apm/metadata.md) that are available.
::::



### `func (*Context) SetUsername(username string)` [context-set-username]

SetUsername records the username of the user associated with the transaction.


### `func (*Context) SetUserID(id string)` [context-set-user-id]

SetUserID records the ID of the user associated with the transaction.


### `func (*Context) SetUserEmail(email string)` [context-set-user-email]

SetUserEmail records the email address of the user associated with the transaction.


## Errors [error-api]

Elastic APM provides two methods of capturing an error event: reporting an error log record, and reporting an "exception" (either a panic or an error in Go parlance).


### `func (*Tracer) NewError(error) *Error` [tracer-new-error]

NewError returns a new Error with details taken from err.

The exception message will be set to `err.Error()`. The exception module and type will be set to the package and type name of the cause of the error, respectively, where the cause has the same definition as given by [github.com/pkg/errors](https://github.com/pkg/errors).

```go
e := apm.DefaultTracer().NewError(err)
...
e.Send()
```

The provided error can implement any of several interfaces to provide additional information:

```go
// Errors implementing ErrorsStacktracer will have their stacktrace
// set based on the result of the StackTrace method.
type ErrorsStacktracer interface {
    StackTrace() github.com/pkg/errors.StackTrace
}

// Errors implementing Stacktracer will have their stacktrace
// set based on the result of the StackTrace method.
type Stacktracer interface {
    StackTrace() []go.elastic.co/apm/v2/stacktrace.Frame
}

// Errors implementing Typer will have a "type" field set to the
// result of the Type method.
type Typer interface {
	Type() string
}

// Errors implementing StringCoder will have a "code" field set to the
// result of the Code method.
type StringCoder interface {
	Code() string
}

// Errors implementing NumberCoder will have a "code" field set to the
// result of the Code method.
type NumberCoder interface {
	Code() float64
}
```

Errors created by with NewError will have their ID field populated with a unique ID. This can be used in your application for correlation.


### `func (*Tracer) NewErrorLog(ErrorLogRecord) *Error` [tracer-new-error-log]

NewErrorLog returns a new Error for the given ErrorLogRecord:

```go
type ErrorLogRecord struct {
	// Message holds the message for the log record,
	// e.g. "failed to connect to %s".
	//
	// If this is empty, "[EMPTY]" will be used.
	Message string

	// MessageFormat holds the non-interpolated format
	// of the log record, e.g. "failed to connect to %s".
	//
	// This is optional.
	MessageFormat string

	// Level holds the severity level of the log record.
	//
	// This is optional.
	Level string

	// LoggerName holds the name of the logger used.
	//
	// This is optional.
	LoggerName string

	// Error is an error associated with the log record.
	//
	// This is optional.
	Error error
}
```

The resulting Error’s log stacktrace will not be set. Call the SetStacktrace method to set it.

```go
e := apm.DefaultTracer().NewErrorLog(apm.ErrorLogRecord{
	Message: "Somebody set up us the bomb.",
})
...
e.Send()
```


### `func (*Error) SetTransaction(*Transaction)` [error-set-transaction]

SetTransaction associates the error with the given transaction.


### `func (*Error) SetSpan(*Span)` [error-set-span]

SetSpan associates the error with the given span and the span’s transaction. When calling SetSpan, it is not necessary to also call SetTransaction.


### `func (*Error) Send()` [error-send]

Send enqueues the error for sending to the Elastic APM server.


### `func (*Tracer) Recovered(interface{}) *Error` [tracer-recovered]

Recovered returns an Error from the recovered value, optionally associating it with a transaction. The error is not sent; it is the caller’s responsibility to set the error’s context, and then call its `Send` method.

```go
tx := apm.DefaultTracer().StartTransaction(...)
defer tx.End()
defer func() {
	if v := recover(); v != nil {
		e := apm.DefaultTracer().Recovered(v)
		e.SetTransaction(tx)
		e.Send()
	}
}()
```


### `func CaptureError(context.Context, error) *Error` [apm-captureerror]

CaptureError returns a new error related to the sampled transaction and span present in the context, if any, and sets its exception details using the given error. The Error.Handled field will be set to true, and a stacktrace set.

If there is no transaction in the context, or it is not being sampled, CaptureError returns nil. As a convenience, if the provided error is nil, then CaptureError will also return nil.

```go
if err != nil {
        e := apm.CaptureError(ctx, err)
        e.Send()
}
```


### Trace Context [trace-context]

Trace context contains the ID for a transaction or span, the ID of the end-to-end trace to which the transaction or span belongs, and trace options such as flags relating to sampling. Trace context is propagated between processes, e.g. in HTTP headers, in order to correlate events originating from related services.

Elastic APM’s trace context is based on the [W3C Trace Context](https://w3c.github.io/trace-context/) draft.


### Error Context [error-context]

Errors can be associated with context just like transactions. See [Context](#context-api) for details. In addition, errors can be associated with an active transaction or span using [SetTransaction](#error-set-transaction) or [SetSpan](#error-set-span), respectively.

```go
tx := apm.DefaultTracer().StartTransaction("GET /foo", "request")
defer tx.End()
e := apm.DefaultTracer().NewError(err)
e.SetTransaction(tx)
e.Send()
```


### Tracer Config [tracer-config-api]

Many configuration attributes can be be updated dynamically via `apm.Tracer` method calls. Please refer to the documentation at [pkg.go.dev/go.elastic.co/apm/v2#Tracer](https://pkg.go.dev/go.elastic.co/apm/v2#Tracer) for details. The configuration methods are primarily prefixed with `Set`, such as [apm#Tracer.SetLogger](https://pkg.go.dev/go.elastic.co/apm/v2#Tracer.SetLogger).

