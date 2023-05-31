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

//go:build go1.18
// +build go1.18

package apmotel

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/otel/trace"

	"go.elastic.co/apm/v2"
	"go.elastic.co/apm/v2/transport/transporttest"
)

func TestTracerStartStoresSpanInContext(t *testing.T) {
	tp, err := NewTracerProvider()
	assert.NoError(t, err)
	tracer := newTracer(tp.(*tracerProvider))

	ctx := context.Background()
	ctx, s := tracer.Start(ctx, "name")

	assert.Equal(t, s, trace.SpanFromContext(ctx))
}

func TestTracerStartTransaction(t *testing.T) {
	tp, err := NewTracerProvider()
	assert.NoError(t, err)
	tracer := newTracer(tp.(*tracerProvider))

	ctx := context.Background()
	ctx, s := tracer.Start(ctx, "name")

	assert.NotNil(t, s.(*span).tx)
	assert.Nil(t, s.(*span).span)

	assert.True(t, trace.SpanContextFromContext(ctx).IsValid())
	assert.True(t, trace.SpanContextFromContext(ctx).IsSampled())

	assert.True(t, apm.TransactionFromContext(ctx).Sampled())
	assert.Nil(t, apm.SpanFromContext(ctx))
}

func TestTracerStartTransactionWithParentContext(t *testing.T) {
	for _, tt := range []struct {
		name string

		spanContext     trace.SpanContext
		expectedSampled bool
	}{
		{
			name: "with an empty span context",

			expectedSampled: true,
		},
		{
			name: "with a sampled span context",
			spanContext: trace.NewSpanContext(trace.SpanContextConfig{
				TraceID:    [16]byte{1},
				SpanID:     [8]byte{42},
				TraceFlags: trace.TraceFlags(0).WithSampled(true),
			}),

			expectedSampled: true,
		},
		{
			name: "with a non-sampled span context",
			spanContext: trace.NewSpanContext(trace.SpanContextConfig{
				TraceID:    [16]byte{1},
				SpanID:     [8]byte{42},
				TraceFlags: trace.TraceFlags(0),
			}),

			expectedSampled: false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {

			tp, err := NewTracerProvider()
			assert.NoError(t, err)
			tracer := newTracer(tp.(*tracerProvider))

			ctx := context.Background()
			ctx = trace.ContextWithSpanContext(context.Background(), tt.spanContext)

			ctx, s := tracer.Start(ctx, "name")

			assert.NotNil(t, s.(*span).tx)
			assert.Nil(t, s.(*span).span)

			assert.True(t, trace.SpanContextFromContext(ctx).IsValid())
			assert.Equal(t, trace.SpanContextFromContext(ctx).IsSampled(), tt.expectedSampled)
		})
	}
}

func TestTracerStartChildSpan(t *testing.T) {
	tp, err := NewTracerProvider()
	assert.NoError(t, err)
	tracer := newTracer(tp.(*tracerProvider))

	ctx := context.Background()
	ctx, ps := tracer.Start(ctx, "parentSpan")
	ctx, cs := tracer.Start(ctx, "childSpan")

	assert.Equal(t, ps.(*span).tx, cs.(*span).tx)
	assert.NotNil(t, cs.(*span).span)

	assert.True(t, trace.SpanContextFromContext(ctx).IsValid())

	assert.True(t, apm.TransactionFromContext(ctx).Sampled())
	assert.False(t, apm.SpanFromContext(ctx).Dropped())
}

func TestTracerStartChildSpanFromTransactionInContext(t *testing.T) {
	for _, tt := range []struct {
		name  string
		txOpt apm.TransactionOptions

		expectedSampled bool
	}{
		{
			name: "with an empty trace context",

			expectedSampled: true,
		},
		{
			name: "with a sampled transaction",
			txOpt: apm.TransactionOptions{
				TraceContext: apm.TraceContext{
					Trace:   apm.TraceID{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15},
					Options: apm.TraceOptions(0).WithRecorded(true),
				},
			},

			expectedSampled: true,
		},
		{
			name: "with a non-sampled transaction",
			txOpt: apm.TransactionOptions{
				TraceContext: apm.TraceContext{
					Trace:   apm.TraceID{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15},
					Options: apm.TraceOptions(0),
				},
			},

			expectedSampled: false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {

			apmTracer, _ := transporttest.NewRecorderTracer()
			tp, err := NewTracerProvider(WithAPMTracer(apmTracer))
			assert.NoError(t, err)
			tracer := newTracer(tp.(*tracerProvider))

			ctx := context.Background()
			tx := apmTracer.StartTransactionOptions("parent", "", tt.txOpt)
			ctx = apm.ContextWithTransaction(context.Background(), tx)
			ctx, cs := tracer.Start(ctx, "childSpan")

			assert.Equal(t, tx, cs.(*span).tx)
			assert.NotNil(t, cs.(*span).span)

			assert.True(t, trace.SpanContextFromContext(ctx).IsValid())
			assert.Equal(t, tt.expectedSampled, trace.SpanContextFromContext(ctx).IsSampled())
		})
	}
}

func TestTracerStartChildSpanWithNewRoot(t *testing.T) {
	tp, err := NewTracerProvider()
	assert.NoError(t, err)
	tracer := newTracer(tp.(*tracerProvider))

	ctx := context.Background()
	ctx, ps := tracer.Start(ctx, "parentSpan")
	ctx, cs := tracer.Start(ctx, "childSpan", trace.WithNewRoot())

	assert.Nil(t, cs.(*span).span)
	assert.NotNil(t, cs.(*span).tx)
	assert.NotEqual(t, ps.(*span).tx, cs.(*span).tx)
}
