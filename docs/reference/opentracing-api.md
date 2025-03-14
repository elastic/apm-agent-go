---
mapped_pages:
  - https://www.elastic.co/guide/en/apm/agent/go/current/opentracing.html
---

# OpenTracing API [opentracing]

The Elastic APM Go agent provides an implementation of the [OpenTracing API](https://opentracing.io), building on top of the core Elastic APM API.

Spans created through the OpenTracing API will be translated to Elastic APM transactions or spans. Root spans, and spans created with a remote span context, will be translated to Elastic APM transactions. All others will be created as Elastic APM spans.


## Initializing the tracer [opentracing-init]

The OpenTracing API implementation is implemented as a bridge on top of the core Elastic APM API. To initialize the OpenTracing tracer implementation, you must first import the `apmot` package:

```go
import (
	"go.elastic.co/apm/module/apmot/v2"
)
```

The apmot package exports a function, "New", which returns an implementation of the `opentracing.Tracer` interface. If you call `apmot.New()` without any arguments, the returned tracer will wrap `apm.DefaultTracer()`. If you wish to use a different `apm.Tracer`, then you can pass it with `apmot.New(apmot.WithTracer(t))`.

```go
otTracer := apmot.New()
```

Once you have obtained an `opentracing.Tracer`, use the standard OpenTracing API to report spans to Elastic APM. Please refer to [opentracing-go](https://github.com/opentracing/opentracing-go) for documentation on the OpenTracing Go API.

```go
import (
	"context"

	"go.elastic.co/apm/module/apmot/v2"

	"github.com/opentracing/opentracing-go"
)

func main() {
	opentracing.SetGlobalTracer(apmot.New())

	parent, ctx := opentracing.StartSpanFromContext(context.Background(), "parent")
	child, _ := opentracing.StartSpanFromContext(ctx, "child")
	child.Finish()
	parent.Finish()
}
```


## Mixing Native and OpenTracing APIs [opentracing-mixed]

When you import `apmot`, transactions and spans created with the [native API](/reference/api-documentation.md) will be made available as OpenTracing spans, enabling you to mix the use of the native and OpenTracing APIs. e.g.:

```go
// Transaction created through native API.
transaction := apm.DefaultTracer().StartTransaction("GET /", "request")
ctx := apm.ContextWithTransaction(context.Background(), transaction)

// Span created through OpenTracing API will be a child of the transaction.
otSpan, ctx := opentracing.StartSpanFromContext(ctx, "ot-span")

// Span created through the native API will be a child of the span created
// above via the OpenTracing API.
apmSpan, ctx := apm.StartSpan(ctx, "apm-span", "apm-span")
```

The `opentracing.SpanFromContext` function will return an `opentracing.Span` that wraps either an `apm.Span` or `apm.Transaction`. These span objects are intended only for passing in context when creating a new span through the OpenTracing API, and are not fully functional spans. In particular, the `Finish` and `Log*` methods are no-ops, and the `Tracer` method returns a no-op tracer.


## Elastic APM specific tags [opentracing-apm-tags]

Elastic APM defines some tags which are not included in the OpenTracing API, but are relevant in the context of Elastic APM. Some tags are relevant only to Elastic APM transactions.

* `type` - sets the type of the transaction or span, e.g. "request", or "ext.http". If `type` is not specified, then the type may be inferred from other tags. e.g. if "http.url" is specified, then the type will be "request" for transactions, and "ext.http" for spans. If no type can be inferred, it is set to "unknown".

The following tags are relevant only to root or service-entry spans, which are translated to Elastic APM transactions:

* `user.id` - sets the user ID, which appears in the "User" tab in the transaction details in the Elastic APM app
* `user.email` - sets the user email, which appears in the "User" tab in the transaction details in the Elastic APM app
* `user.username` - sets the user name, which appears in the "User" tab in the transaction details in the Elastic APM app
* `result` - sets the result of the transaction. If `result` is *not* specified, but `error` tag is set to `true`, then the transaction result will be set to "error"


## Span Logs [opentracing-logs]

The `Span.LogKV` and `Span.LogFields` methods will send error events to Elastic APM for logs with the "event" field set to "error".

The deprecated log methods `Span.Log`, `Span.LogEvent`, and `Span.LogEventWithPayload` are no-ops.


## Caveats [opentracing-caveats]


### Context Propagation [opentracing-caveats-propagation]

We support the `TextMap` and `HTTPHeaders` propagation formats; `Binary` is not currently supported.


### Span References [opentracing-caveats-spanrefs]

We support only `ChildOf` references. Other references, e.g. `FollowsFrom`, are not currently supported.


### Baggage [opentracing-caveats-baggage]

`Span.SetBaggageItem` is a no-op; baggage items are silently dropped.

