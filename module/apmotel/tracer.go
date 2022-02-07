// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package apmotel // import "go.elastic.co/apm/module/apmotel/v2"

import (
	"context"
	"net/url"
	"strings"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"go.elastic.co/apm/v2"
)

// Tracer creates a named tracer that implements otel.Tracer.
// If the name is an empty string then provider uses default name.
func Tracer(_ string, opts ...trace.TracerOption) trace.Tracer {
	apmOpts := apm.TracerOptions{}
	tracerCfg := trace.NewTracerConfig(opts...)
	if version := (&tracerCfg).InstrumentationVersion(); version != "" {
		apmOpts.ServiceVersion = version
	}

	t, err := apm.NewTracerOptions(apmOpts)
	if err != nil {
		panic(err)
	}

	return &tracer{inner: t}
}

// NewTracerProvider creates a new trace.Tracer with the provided apm.Tracer.
func NewTracerProvider(t *apm.Tracer) trace.TracerProvider {
	return tracerFunc(func(_ string, _ ...trace.TracerOption) trace.Tracer {
		return &tracer{inner: t}
	})
}

// SetTracerProvider is a noop function to match opentelemetry's trace module.
func SetTracerProvider(_ trace.TracerProvider) {}

// GetTracerProvider returns a trace.TraceProvider that executes this package's
// Trace() function.
func GetTracerProvider() trace.TracerProvider {
	return tracerFunc(Tracer)
}

type tracerFunc func(string, ...trace.TracerOption) trace.Tracer

func (fn tracerFunc) Tracer(name string, opts ...trace.TracerOption) trace.Tracer {
	return fn(name, opts...)
}

type tracer struct {
	inner *apm.Tracer
}

// Start starts a new trace.Span. The span is stored in the returned context.
func (t *tracer) Start(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return newSpan(ctx, t, spanName, opts...)
}

func newSpan(ctx context.Context, t *tracer, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	cfg := trace.NewSpanStartConfig(opts...)

	var (
		rpc, http, messaging bool

		spanType, subtype, resource string

		dbName, peerPort, netName, messagingDest, rpcService, httpHost, httpScheme, httpURL string
	)
	for _, attr := range cfg.Attributes() {
		if attr.Value.Type() == attribute.INVALID {
			continue
		}
		val := attr.Value.Emit()
		switch attr.Key {
		case "db.name":
			dbName = val
		case "db.system":
			spanType = "db"
			subtype = val
		case "http.host":
			httpHost = val
		case "http.scheme":
			http = true
			spanType = "external"
			subtype = "http"
			httpScheme = val
		case "http.url":
			http = true
			spanType = "external"
			subtype = "http"
			httpURL = val
		// TODO: Verify we want to set messaging for all of these, and
		// not just `messaging.system`.
		case "messaging.system", "messaging.operation", "message_bus.destination":
			spanType = "messaging"
			subtype = val
			messaging = true
		case "messaging.url":
			// "net.peer.name" takes precedence
			if netName == "" {
				netName = parseNetName(val)
			}
		case "messaging.destination":
			messagingDest = val
		case "net.peer.port":
			// Question: Are these string values? Can we just use the
			// string value?
			peerPort = val
		case "net.peer.name":
			netName = val
		case "net.peer.ip":
			// "net.peer.name" takes precedence
			if netName == "" {
				netName = val
			}
		case "rpc.system":
			rpc = true
			spanType = "external"
			subtype = val
		case "rpc.service":
			rpcService = val
		}
	}

	if netName != "" && len(peerPort) > 0 {
		netName += ":" + peerPort
	}

	if netName != "" {
		resource = netName
	} else {
		resource = subtype
	}

	// Question: This code is assuming that if a group is set, eg. db.* or
	// rpc.*, then other groups won't be set. This means we don't need to
	// coordinate updating the value of `resource`.
	if dbName != "" {
		resource += "/" + dbName
	} else if messagingDest != "" {
		resource += "/" + messagingDest
	} else if rpcService != "" {
		resource += "/" + rpcService
	} else if httpHost != "" && httpScheme != "" {
		resource = httpHost + ":" + httpPortFromScheme(httpScheme)
	} else if httpURL != "" {
		resource = parseNetName(httpURL)
	}

	txType := "unknown"
	switch cfg.SpanKind() {
	case trace.SpanKindServer:
		if rpc || http {
			txType = "request"
		}
	case trace.SpanKindConsumer:
		if messaging {
			txType = "messaging"
		}
	}

	if spanType == "" {
		if cfg.SpanKind() == trace.SpanKindInternal {
			spanType = "app"
			subtype = "internal"
		} else {
			spanType = "unknown"
		}
	}

	// Question: This is lowercase, but the example in our docs is showing them as being uppercase
	// Using uppercase for now.
	// https://github.com/open-telemetry/opentelemetry-go/blob/trace/v1.3.0/trace/trace.go#L469-L484
	// https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/trace/api.md#spankind
	spanKind := strings.ToUpper(cfg.SpanKind().String())
	spanCtx := trace.SpanContextFromContext(ctx)
	tx := apm.TransactionFromContext(ctx)

	if cfg.NewRoot() {
		return newRootTransaction(ctx, t.inner, cfg.Attributes(), spanKind, name, txType)
	} else if spanCtx.IsValid() {
		txCtx := apm.TraceContext{
			Options: apm.TraceOptions(spanCtx.TraceFlags()),
		}
		if spanCtx.HasTraceID() {
			txCtx.Trace = apm.TraceID(spanCtx.TraceID())
		}
		if spanCtx.HasSpanID() {
			txCtx.Span = apm.SpanID(spanCtx.SpanID())
		}
		if ts, err := apm.ParseTraceState(spanCtx.TraceState().String()); err == nil {
			txCtx.State = ts
		}
		txOpts := apm.TransactionOptions{TraceContext: txCtx}

		// If timestamp has been set, ie. it's not the default value,
		// set it on txOpts.
		if start := cfg.Timestamp(); start.After(time.Unix(0, 0)) {
			txOpts.Start = start
		}

		tx := t.inner.StartTransactionOptions(name, txType, txOpts)
		tx.Context.SetSpanKind(spanKind)
		tx.Context.SetOtelAttributes(cfg.Attributes()...)
		ctx := apm.ContextWithTransaction(ctx, tx)
		return ctx, &transaction{inner: tx, tracer: t.inner}
	} else if tx == nil {
		return newRootTransaction(ctx, t.inner, cfg.Attributes(), spanKind, name, txType)
	} else {
		// TODO: Populate data in SpanOptions
		spanOpts := apm.SpanOptions{}
		// If timestamp has been set, ie. it's not the default value,
		// set it on txOpts.
		if start := cfg.Timestamp(); start.After(time.Unix(0, 0)) {
			spanOpts.Start = start
		}

		txID := tx.TraceContext().Span
		s := t.inner.StartSpan(name, spanType, txID, spanOpts)

		s.Subtype = subtype
		s.Context.SetDestinationService(apm.DestinationServiceSpanContext{Resource: resource})
		s.Context.SetSpanKind(spanKind)
		tx.Context.SetOtelAttributes(cfg.Attributes()...)
		ctx := apm.ContextWithSpan(ctx, s)
		return ctx, &span{inner: s, tracer: t.inner}
	}
}

func parseNetName(u string) string {
	parsed, err := url.Parse(u)
	if err != nil {
		// Question: What do we do in this situation?
		return ""
	}
	if parsed.Port() != "" {
		return parsed.Host
	} else if port := httpPortFromScheme(parsed.Scheme); port != "" {
		return parsed.Host + ":" + port
	}

	return parsed.Host
}

func httpPortFromScheme(scheme string) string {
	switch scheme {
	case "https":
		return "443"
	case "http":
		return "80"
	default:
		return ""
	}
}
