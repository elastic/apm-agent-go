package elasticapm_test

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/apm-agent-go"
	"github.com/elastic/apm-agent-go/transport/transporttest"
)

func TestStartSpanTransactionNotSampled(t *testing.T) {
	tracer, _ := elasticapm.NewTracer("tracer_testing", "")
	defer tracer.Close()
	// sample nothing
	tracer.SetSampler(elasticapm.NewRatioSampler(0, rand.New(rand.NewSource(0))))

	tx := tracer.StartTransaction("name", "type")
	assert.False(t, tx.Sampled())
	span := tx.StartSpan("name", "type", nil)
	assert.True(t, span.Dropped())
}

func TestTracerStartSpan(t *testing.T) {
	tracer, r := transporttest.NewRecorderTracer()
	defer tracer.Close()

	tx := tracer.StartTransaction("name", "type")
	txTraceContext := tx.TraceContext()
	span0 := tx.StartSpan("name", "type", nil)
	span0TraceContext := span0.TraceContext()
	span0.End()
	tx.End()

	// Even if the transaction and parent span have been ended,
	// it is possible to report a span with their IDs.
	tracer.StartSpan("name", "type", txTraceContext.Span, elasticapm.SpanOptions{
		Parent: span0TraceContext,
	}).End()

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

	// TODO(axw) check total span count for tx is just 1
}
