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

package apmotel_test

import (
	"context"
	"encoding/binary"
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"go.elastic.co/apm/module/apmotel/v2"
	"go.elastic.co/apm/v2"
	"go.elastic.co/apm/v2/model"
	"go.elastic.co/apm/v2/transport/transporttest"
)

func TestSpanStartAttributesNewRoot(t *testing.T) {
	tracer, apmtracer, recorder := newTestTracer()
	defer apmtracer.Close()

	tx := apmtracer.StartTransaction("root", "root")
	defer tx.End()
	ctx := context.Background()

	tcs := []struct {
		attrs    []attribute.KeyValue
		spanKind trace.SpanKind
		spanType string
	}{
		{
			spanType: "messaging",
			attrs: []attribute.KeyValue{
				attribute.String("messaging.system", "msgSystem"),
			},
			spanKind: trace.SpanKindConsumer,
		},
		{
			spanType: "unknown",
			attrs:    []attribute.KeyValue{},
			spanKind: trace.SpanKindConsumer,
		},
		{
			spanType: "unknown",
			attrs: []attribute.KeyValue{
				attribute.String("messaging.system", "msgSystem"),
			},
			spanKind: trace.SpanKindServer,
		},
		{
			spanType: "request",
			attrs: []attribute.KeyValue{
				attribute.String("rpc.system", "rpcSystem"),
			},
			spanKind: trace.SpanKindServer,
		},
		{
			spanType: "request",
			attrs: []attribute.KeyValue{
				attribute.String("http.url", "myURL"),
			},
			spanKind: trace.SpanKindServer,
		},
		{
			spanType: "unknown",
			attrs: []attribute.KeyValue{
				attribute.String("rpc.system", "rpcSystem"),
			},
			spanKind: trace.SpanKindConsumer,
		},
		{
			spanType: "unknown",
			attrs: []attribute.KeyValue{
				attribute.String("http.url", "myURL"),
			},
			spanKind: trace.SpanKindConsumer,
		},
	}

	for i, tc := range tcs {
		_, span := tracer.Start(ctx, fmt.Sprintf("tc%d", i), trace.WithAttributes(tc.attrs...), trace.WithSpanKind(tc.spanKind), trace.WithNewRoot())
		span.End()
	}

	apmtracer.Flush(nil)
	payloads := recorder.Payloads()
	txs := payloads.Transactions
	require.Len(t, txs, len(tcs))
	for i, tc := range tcs {
		assert.Equal(t, tc.spanType, txs[i].Type)
		assert.Equal(t, strings.ToUpper(tc.spanKind.String()), txs[i].OTel.SpanKind)
	}
}

func TestSpanStartAttributesNoTx(t *testing.T) {
	tracer, apmtracer, recorder := newTestTracer()
	defer apmtracer.Close()

	tx := apmtracer.StartTransaction("root", "root")
	defer tx.End()
	ctx := context.Background()

	tcs := []struct {
		attrs    []attribute.KeyValue
		spanKind trace.SpanKind
		spanType string
	}{
		{
			spanType: "messaging",
			attrs: []attribute.KeyValue{
				attribute.String("messaging.system", "msgSystem"),
			},
			spanKind: trace.SpanKindConsumer,
		},
		{
			spanType: "unknown",
			attrs:    []attribute.KeyValue{},
			spanKind: trace.SpanKindConsumer,
		},
		{
			spanType: "unknown",
			attrs: []attribute.KeyValue{
				attribute.String("messaging.system", "msgSystem"),
			},
			spanKind: trace.SpanKindServer,
		},
		{
			spanType: "request",
			attrs: []attribute.KeyValue{
				attribute.String("rpc.system", "rpcSystem"),
			},
			spanKind: trace.SpanKindServer,
		},
		{
			spanType: "request",
			attrs: []attribute.KeyValue{
				attribute.String("http.url", "myURL"),
			},
			spanKind: trace.SpanKindServer,
		},
		{
			spanType: "unknown",
			attrs: []attribute.KeyValue{
				attribute.String("rpc.system", "rpcSystem"),
			},
			spanKind: trace.SpanKindConsumer,
		},
		{
			spanType: "unknown",
			attrs: []attribute.KeyValue{
				attribute.String("http.url", "myURL"),
			},
			spanKind: trace.SpanKindConsumer,
		},
	}

	for i, tc := range tcs {
		_, span := tracer.Start(ctx, fmt.Sprintf("tc%d", i), trace.WithAttributes(tc.attrs...), trace.WithSpanKind(tc.spanKind))
		span.End()
	}

	apmtracer.Flush(nil)
	payloads := recorder.Payloads()
	txs := payloads.Transactions
	require.Len(t, txs, len(tcs))
	for i, tc := range tcs {
		assert.Equal(t, tc.spanType, txs[i].Type)
		assert.Equal(t, strings.ToUpper(tc.spanKind.String()), txs[i].OTel.SpanKind)
	}
}

func TestSpanAttributes(t *testing.T) {
	tracer, apmtracer, recorder := newTestTracer()
	defer apmtracer.Close()

	ctx := context.Background()

	attrs := []attribute.KeyValue{
		attribute.String("messaging.system", "messagingSystem"),
		attribute.String("messaging.destination", "destination"),
		attribute.String("net.peer.port", "1234"),
		attribute.String("net.peer.ip", "1.2.3.4"),
	}
	spanKind := trace.SpanKindServer

	_, otelTx := tracer.Start(ctx, "tc", trace.WithAttributes(attrs...), trace.WithSpanKind(spanKind))
	otelTx.End()

	tx := apmtracer.StartTransaction("root", "root")
	defer tx.End()
	ctx = apm.ContextWithTransaction(ctx, tx)
	_, span := tracer.Start(ctx, "tc", trace.WithAttributes(attrs...), trace.WithSpanKind(spanKind))
	span.End()

	apmtracer.Flush(nil)
	payloads := recorder.Payloads()
	spans := payloads.Spans
	require.Len(t, spans, 1)
	assert.Equal(t, "SERVER", spans[0].OTel.SpanKind)
	txs := payloads.Transactions
	require.Len(t, txs, 1)
	assert.Equal(t, "SERVER", txs[0].OTel.SpanKind)
	m := make(map[string]interface{}, len(attrs))
	for _, kv := range attrs {
		m[string(kv.Key)] = kv.Value.AsInterface()
	}
	assert.Equal(t, m, spans[0].OTel.Attributes)
	assert.Equal(t, m, txs[0].OTel.Attributes)
}

func TestSpanStartAttributesWithTx(t *testing.T) {
	tracer, apmtracer, recorder := newTestTracer()
	defer apmtracer.Close()

	tx := apmtracer.StartTransaction("root", "root")
	defer tx.End()
	ctx := context.Background()
	ctx = apm.ContextWithTransaction(ctx, tx)

	tcs := []struct {
		attrs                       []attribute.KeyValue
		spanKind                    trace.SpanKind
		spanType, subtype, resource string
	}{
		{
			spanType: "db",
			subtype:  "dbSystem",
			resource: "dbSystem/myDB",
			attrs: []attribute.KeyValue{
				attribute.String("db.name", "myDB"),
				attribute.String("db.system", "dbSystem"),
			},
			spanKind: trace.SpanKindServer,
		},
		{
			spanType: "db",
			subtype:  "dbSystem",
			resource: "peerName:1234/myDB",
			attrs: []attribute.KeyValue{
				attribute.String("db.name", "myDB"),
				attribute.String("db.system", "dbSystem"),
				attribute.String("net.peer.port", "1234"),
				attribute.String("net.peer.name", "peerName"),
			},
			spanKind: trace.SpanKindServer,
		},
		{
			spanType: "messaging",
			subtype:  "messagingSystem",
			resource: "1.2.3.4:1234/destination",
			attrs: []attribute.KeyValue{
				attribute.String("messaging.system", "messagingSystem"),
				attribute.String("messaging.destination", "destination"),
				attribute.String("net.peer.port", "1234"),
				attribute.String("net.peer.ip", "1.2.3.4"),
			},
			spanKind: trace.SpanKindServer,
		},
		{
			spanType: "messaging",
			subtype:  "messagingSystem",
			resource: "1.2.3.4",
			attrs: []attribute.KeyValue{
				attribute.String("messaging.system", "messagingSystem"),
				attribute.String("net.peer.ip", "1.2.3.4"),
			},
			spanKind: trace.SpanKindServer,
		},
		{
			spanType: "external",
			subtype:  "rpcSystem",
			resource: "rpcSystem/rpcService",
			attrs: []attribute.KeyValue{
				attribute.String("rpc.system", "rpcSystem"),
				attribute.String("rpc.service", "rpcService"),
			},
			spanKind: trace.SpanKindServer,
		},
		{
			spanType: "external",
			subtype:  "http",
			resource: "example.com:443",
			attrs: []attribute.KeyValue{
				attribute.String("http.host", "example.com"),
				attribute.String("http.scheme", "https"),
			},
			spanKind: trace.SpanKindServer,
		},
		{
			spanType: "external",
			subtype:  "http",
			resource: "example.com:80",
			attrs: []attribute.KeyValue{
				attribute.String("http.host", "example.com"),
				attribute.String("http.scheme", "http"),
			},
			spanKind: trace.SpanKindServer,
		},
		{
			spanType: "external",
			subtype:  "http",
			resource: "www.example.com:30443",
			attrs: []attribute.KeyValue{
				attribute.String("http.url", "https://www.example.com:30443"),
				attribute.String("http.scheme", "https"),
			},
			spanKind: trace.SpanKindServer,
		},
		{
			spanType: "app",
			subtype:  "internal",
			resource: "",
			attrs: []attribute.KeyValue{
				attribute.String("not.known", "unknown"),
			},
			spanKind: trace.SpanKindInternal,
		},
		{
			spanType: "unknown",
			subtype:  "",
			resource: "",
			attrs: []attribute.KeyValue{
				attribute.String("not.known", "unknown"),
			},
			spanKind: trace.SpanKindClient,
		},
	}

	for i, tc := range tcs {
		_, span := tracer.Start(ctx, fmt.Sprintf("tc%d", i), trace.WithAttributes(tc.attrs...), trace.WithSpanKind(tc.spanKind))
		span.End()
	}

	apmtracer.Flush(nil)
	payloads := recorder.Payloads()
	spans := payloads.Spans
	require.Len(t, spans, len(tcs))
	for i, tc := range tcs {
		assert.Equal(t, model.SpanID(tx.TraceContext().Span), spans[i].ParentID)
		assert.Equal(t, model.SpanID(tx.TraceContext().Span), spans[i].TransactionID)

		assert.Equal(t, tc.spanType, spans[i].Type)
		assert.Equal(t, tc.subtype, spans[i].Subtype)
		if tc.resource != "" {
			assert.Equal(t, tc.resource, spans[i].Context.Destination.Service.Resource)
		}
		assert.Equal(t, strings.ToUpper(tc.spanKind.String()), spans[i].OTel.SpanKind)
	}
}

func TestSpanStartAttributesValidSpanCtx(t *testing.T) {
	tracer, apmtracer, recorder := newTestTracer()
	defer apmtracer.Close()

	ctx := context.Background()
	traceID := [16]byte{}
	binary.LittleEndian.PutUint64(traceID[:8], rand.Uint64())
	binary.LittleEndian.PutUint64(traceID[8:], rand.Uint64())

	spanID := [8]byte{}
	copy(spanID[:], traceID[:])

	traceFlags := trace.TraceFlags(1)

	s := "es=s:1;a:b,z=w,a=d"
	traceState, err := trace.ParseTraceState(s)
	require.NoError(t, err)

	cfg := trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: traceFlags,
		TraceState: traceState,
		Remote:     false,
	}
	spanCtx := trace.NewSpanContext(cfg)
	ctx = trace.ContextWithSpanContext(ctx, spanCtx)

	spanType := "unknown"
	attrs := []attribute.KeyValue{
		attribute.String("not.known", "unknown"),
	}
	spanKind := trace.SpanKindClient
	timestamp := time.Now().Add(time.Hour)
	_, span := tracer.Start(ctx, "tc", trace.WithAttributes(attrs...), trace.WithSpanKind(spanKind), trace.WithTimestamp(timestamp))
	span.End()

	apmtracer.Flush(nil)
	payloads := recorder.Payloads()
	txs := payloads.Transactions
	require.Len(t, txs, 1)
	assert.Equal(t, model.SpanID(cfg.SpanID), txs[0].ParentID)
	assert.Equal(t, model.TraceID(cfg.TraceID), txs[0].TraceID)
	assert.Equal(t, timestamp.Unix(), time.Time(txs[0].Timestamp).Unix())
	assert.Equal(t, spanType, txs[0].Type)
	// model.Transaction doesn't seem to have a tracecontext or
	// tracestate, maybe it's only in the headers? how to access them?
	// assert.Equal(t, model.TraceOptions(cfg.TraceFlags), txs[i].TraceContext().Options)
	assert.Equal(t, strings.ToUpper(spanKind.String()), txs[0].OTel.SpanKind)
}

func TestSetStatus(t *testing.T) {
	tracer, apmtracer, recorder := newTestTracer()
	defer apmtracer.Close()
	ctx := context.Background()

	tcs := []struct {
		code    codes.Code
		outcome string
	}{
		{codes.Unset, "unknown"},
		{codes.Error, "failure"},
		{codes.Ok, "success"},
	}

	for i, tc := range tcs {
		// No tx in ctx, root tx created
		_, span := tracer.Start(ctx, fmt.Sprintf("tc%d", i))
		span.SetStatus(tc.code, "")
		span.End()
	}

	tx := apmtracer.StartTransaction("root", "root")
	defer tx.End()
	ctx = apm.ContextWithTransaction(ctx, tx)

	for i, tc := range tcs {
		// tx in ctx, span created
		_, tx := tracer.Start(ctx, fmt.Sprintf("tc%d", i))
		tx.SetStatus(tc.code, "")
		tx.End()
	}

	apmtracer.Flush(nil)
	payloads := recorder.Payloads()
	spans := payloads.Spans
	require.Len(t, spans, len(tcs))
	txs := payloads.Transactions
	require.Len(t, txs, len(tcs))
	for i, tc := range tcs {
		assert.Equal(t, tc.outcome, spans[i].Outcome)
		assert.Equal(t, tc.outcome, txs[i].Outcome)
	}
}

func newTestTracer() (trace.Tracer, *apm.Tracer, *transporttest.RecorderTransport) {
	apmtracer, recorder := transporttest.NewRecorderTracer()
	tracer := apmotel.NewTracerProvider(apmtracer).Tracer("otel_tracer")
	return tracer, apmtracer, recorder
}
