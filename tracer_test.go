package elasticapm_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/apm-agent-go"
	"github.com/elastic/apm-agent-go/transport/transporttest"
)

func TestTracerStats(t *testing.T) {
	tracer, err := elasticapm.NewTracer("tracer_testing", "")
	assert.NoError(t, err)
	defer tracer.Close()
	tracer.Transport = transporttest.Discard

	for i := 0; i < 500; i++ {
		tracer.StartTransaction("name", "type").End()
	}
	tracer.Flush(nil)
	assert.Equal(t, elasticapm.TracerStats{
		TransactionsSent: 500,
	}, tracer.Stats())
}

func TestTracerClosedSendNonblocking(t *testing.T) {
	tracer, err := elasticapm.NewTracer("tracer_testing", "")
	assert.NoError(t, err)
	tracer.Close()

	for i := 0; i < 1001; i++ {
		tracer.StartTransaction("name", "type").End()
	}
	assert.Equal(t, uint64(1), tracer.Stats().TransactionsDropped)
}

func TestTracerFlushEmpty(t *testing.T) {
	tracer, err := elasticapm.NewTracer("tracer_testing", "")
	assert.NoError(t, err)
	defer tracer.Close()
	tracer.Flush(nil)
}

// TODO(axw) test request size, buffer size

func TestTracerMaxSpans(t *testing.T) {
	tracer, r := transporttest.NewRecorderTracer()
	defer tracer.Close()

	tracer.SetMaxSpans(2)
	tx := tracer.StartTransaction("name", "type")
	// SetMaxSpans only affects transactions started
	// after the call.
	tracer.SetMaxSpans(99)

	s0 := tx.StartSpan("name", "type", nil)
	s1 := tx.StartSpan("name", "type", nil)
	s2 := tx.StartSpan("name", "type", nil)
	tx.End()

	assert.False(t, s0.Dropped())
	assert.False(t, s1.Dropped())
	assert.True(t, s2.Dropped())

	tracer.Flush(nil)
	payloads := r.Payloads()
	transaction := payloads.Transactions[0]
	assert.Len(t, transaction.Spans, 2)
}

func TestTracerErrors(t *testing.T) {
	tracer, r := transporttest.NewRecorderTracer()
	defer tracer.Close()

	error_ := tracer.NewError(errors.New("zing"))
	error_.Send()
	tracer.Flush(nil)

	payloads := r.Payloads()
	exception := payloads.Errors[0].Exception
	stacktrace := exception.Stacktrace
	assert.Equal(t, "zing", exception.Message)
	assert.Equal(t, "errors", exception.Module)
	assert.Equal(t, "errorString", exception.Type)
	assert.NotEmpty(t, stacktrace)
	assert.Equal(t, "TestTracerErrors", stacktrace[0].Function)
}

func TestTracerRecover(t *testing.T) {
	tracer, r := transporttest.NewRecorderTracer()
	defer tracer.Close()

	capturePanic(tracer, "blam")
	tracer.Flush(nil)

	payloads := r.Payloads()
	error0 := payloads.Errors[0]
	transaction := payloads.Transactions[0]
	assert.Equal(t, "blam", error0.Exception.Message)
	assert.Equal(t, transaction.ID.UUID, error0.Transaction.ID)
}

func capturePanic(tracer *elasticapm.Tracer, v interface{}) {
	tx := tracer.StartTransaction("name", "type")
	defer tx.End()
	defer tracer.Recover(tx)
	panic(v)
}

func TestTracerServiceNameValidation(t *testing.T) {
	_, err := elasticapm.NewTracer("wot!", "")
	assert.EqualError(t, err, `invalid service name "wot!": character '!' is not in the allowed set (a-zA-Z0-9 _-)`)
}

func TestSpanStackTrace(t *testing.T) {
	tracer, r := transporttest.NewRecorderTracer()
	defer tracer.Close()
	tracer.SetSpanFramesMinDuration(10 * time.Millisecond)

	tx := tracer.StartTransaction("name", "type")
	s := tx.StartSpan("name", "type", nil)
	s.Duration = 9 * time.Millisecond
	s.End()
	s = tx.StartSpan("name", "type", nil)
	s.Duration = 10 * time.Millisecond
	s.End()
	s = tx.StartSpan("name", "type", nil)
	s.SetStacktrace(1)
	s.Duration = 11 * time.Millisecond
	s.End()
	tx.End()
	tracer.Flush(nil)

	transaction := r.Payloads().Transactions[0]
	assert.Len(t, transaction.Spans, 3)

	// Span 0 took only 9ms, so we don't set its stacktrace.
	span0 := transaction.Spans[0]
	assert.Nil(t, span0.Stacktrace)

	// Span 1 took the required 10ms, so we set its stacktrace.
	span1 := transaction.Spans[1]
	assert.NotNil(t, span1.Stacktrace)
	assert.NotEqual(t, span1.Stacktrace[0].Function, "TestSpanStackTrace")

	// Span 2 took more than the required 10ms, but its stacktrace
	// was already set; we don't replace it.
	span2 := transaction.Spans[2]
	assert.NotNil(t, span2.Stacktrace)
	assert.Equal(t, span2.Stacktrace[0].Function, "TestSpanStackTrace")
}
