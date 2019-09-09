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

package apm_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.elastic.co/apm"
	"go.elastic.co/apm/apmtest"
	"go.elastic.co/apm/model"
	"go.elastic.co/apm/transport/transporttest"
)

func TestStartSpanTransactionNotSampled(t *testing.T) {
	tracer, _ := apm.NewTracer("tracer_testing", "")
	defer tracer.Close()
	// sample nothing
	tracer.SetSampler(apm.NewRatioSampler(0))

	tx := tracer.StartTransaction("name", "type")
	assert.False(t, tx.Sampled())
	span := tx.StartSpan("name", "type", nil)
	assert.True(t, span.Dropped())
}

func TestTracerStartSpan(t *testing.T) {
	tracer, r := transporttest.NewRecorderTracer()
	defer tracer.Close()

	txTimestamp := time.Now()
	tx := tracer.StartTransactionOptions("name", "type", apm.TransactionOptions{
		Start: txTimestamp,
	})
	txTraceContext := tx.TraceContext()
	span0 := tx.StartSpan("name", "type", nil)
	span0TraceContext := span0.TraceContext()
	span0.End()
	tx.End()

	// Even if the transaction and parent span have been ended,
	// it is possible to report a span with their IDs.
	tracer.StartSpan("name", "type",
		txTraceContext.Span,
		apm.SpanOptions{
			Parent: span0TraceContext,
			Start:  txTimestamp.Add(time.Second),
		},
	).End()

	tracer.Flush(nil)
	payloads := r.Payloads()
	assert.Len(t, payloads.Transactions, 1)
	assert.Len(t, payloads.Spans, 2)

	assert.Equal(t, payloads.Transactions[0].ID, payloads.Spans[0].ParentID)
	assert.Equal(t, payloads.Spans[0].ID, payloads.Spans[1].ParentID)
	for _, span := range payloads.Spans {
		assert.Equal(t, payloads.Transactions[0].TraceID, span.TraceID)
		assert.Equal(t, payloads.Transactions[0].ID, span.TransactionID)
	}
	assert.NotZero(t, payloads.Spans[1].ID)

	assert.Equal(t, time.Time(payloads.Transactions[0].Timestamp).Add(time.Second), time.Time(payloads.Spans[1].Timestamp))

	// The span created after the transaction (obviously?)
	// doesn't get included in the transaction's span count.
	assert.Equal(t, 1, payloads.Transactions[0].SpanCount.Started)
}

func TestSpanTiming(t *testing.T) {
	var spanStart, spanEnd time.Time
	txStart := time.Now()
	tx, spans, _ := apmtest.WithTransactionOptions(
		apm.TransactionOptions{Start: txStart},
		func(ctx context.Context) {
			time.Sleep(500 * time.Millisecond)
			span, _ := apm.StartSpan(ctx, "name", "type")
			spanStart = time.Now()
			time.Sleep(500 * time.Millisecond)
			spanEnd = time.Now()
			span.End()
		},
	)
	require.Len(t, spans, 1)
	span := spans[0]

	assert.InEpsilon(t,
		spanStart.Sub(txStart),
		time.Time(span.Timestamp).Sub(time.Time(tx.Timestamp)),
		0.1, // 10% error
	)
	assert.InEpsilon(t,
		spanEnd.Sub(spanStart)/time.Millisecond,
		span.Duration,
		0.1, // 10% error
	)
}

func TestSpanType(t *testing.T) {
	spanTypes := []string{"type", "type.subtype", "type.subtype.action", "type.subtype.action.figure"}
	_, spans, _ := apmtest.WithTransaction(func(ctx context.Context) {
		for _, spanType := range spanTypes {
			span, _ := apm.StartSpan(ctx, "name", spanType)
			span.End()
		}
	})
	require.Len(t, spans, 4)

	check := func(s model.Span, spanType, spanSubtype, spanAction string) {
		assert.Equal(t, spanType, s.Type)
		assert.Equal(t, spanSubtype, s.Subtype)
		assert.Equal(t, spanAction, s.Action)
	}
	check(spans[0], "type", "", "")
	check(spans[1], "type", "subtype", "")
	check(spans[2], "type", "subtype", "action")
	check(spans[3], "type", "subtype", "action.figure")
}

func TestTracerStartSpanIDSpecified(t *testing.T) {
	spanID := apm.SpanID{0, 1, 2, 3, 4, 5, 6, 7}
	_, spans, _ := apmtest.WithTransaction(func(ctx context.Context) {
		span, _ := apm.StartSpanOptions(ctx, "name", "type", apm.SpanOptions{SpanID: spanID})
		span.End()
	})
	require.Len(t, spans, 1)
	assert.Equal(t, model.SpanID(spanID), spans[0].ID)
}
