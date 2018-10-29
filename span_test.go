package apm_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.elastic.co/apm"
	"go.elastic.co/apm/apmtest"
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
	tx, spans, _ := apmtest.WithTransaction(func(ctx context.Context) {
		time.Sleep(200 * time.Millisecond)
		span, _ := apm.StartSpan(ctx, "name", "type")
		time.Sleep(200 * time.Millisecond)
		span.End()
	})
	require.Len(t, spans, 1)
	span := spans[0]

	assert.InDelta(t,
		time.Time(span.Timestamp).Sub(time.Time(tx.Timestamp)),
		float64(200*time.Millisecond),
		float64(100*time.Millisecond),
	)
	assert.InDelta(t, span.Duration, 200, 100)
}
