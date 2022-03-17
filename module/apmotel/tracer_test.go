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
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"go.elastic.co/apm/module/apmotel/v2"
	"go.elastic.co/apm/v2"
	"go.elastic.co/apm/v2/transport/transporttest"
)

func TestSpanStartAttributes(t *testing.T) {
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

	// TODO: Check when ctx contains span/tx
	for i, tc := range tcs {
		_, span := tracer.Start(ctx, fmt.Sprintf("tc%d", i), trace.WithAttributes(tc.attrs...), trace.WithSpanKind(tc.spanKind))
		span.End()
	}

	apmtracer.Flush(nil)
	payloads := recorder.Payloads()
	spans := payloads.Spans
	require.Len(t, spans, len(tcs))
	for i, tc := range tcs {
		assert.Equal(t, [8]byte(tx.TraceContext().Span), [8]byte(spans[i].ParentID))
		assert.Equal(t, [8]byte(tx.TraceContext().Span), [8]byte(spans[i].TransactionID))

		assert.Equal(t, tc.spanType, spans[i].Type)
		assert.Equal(t, tc.subtype, spans[i].Subtype)
		if tc.resource != "" {
			assert.Equal(t, tc.resource, spans[i].Context.Destination.Service.Resource)
		}
		assert.Equal(t, strings.ToUpper(tc.spanKind.String()), spans[i].OTel.SpanKind)
	}
}

func newTestTracer() (trace.Tracer, *apm.Tracer, *transporttest.RecorderTransport) {
	apmtracer, recorder := transporttest.NewRecorderTracer()
	tracer := apmotel.NewTracerProvider(apmtracer).Tracer("otel_tracer")
	return tracer, apmtracer, recorder
}
