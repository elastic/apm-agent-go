package elasticapm_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/apm-agent-go"
	"github.com/elastic/apm-agent-go/transport/transporttest"
)

func TestStartTransactionTraceContextOptions(t *testing.T) {
	traceContext := startTransactionTraceContextOptions(t, false, false)
	assert.False(t, traceContext.Options.Requested())
	assert.False(t, traceContext.Options.MaybeRecorded())

	traceContext = startTransactionTraceContextOptions(t, false, true)
	assert.False(t, traceContext.Options.Requested())
	assert.False(t, traceContext.Options.MaybeRecorded())

	traceContext = startTransactionTraceContextOptions(t, true, false)
	assert.True(t, traceContext.Options.Requested())
	assert.True(t, traceContext.Options.MaybeRecorded())

	traceContext = startTransactionTraceContextOptions(t, true, true)
	assert.True(t, traceContext.Options.Requested())
	assert.True(t, traceContext.Options.MaybeRecorded())
}

func startTransactionTraceContextOptions(t *testing.T, requested, maybeRecorded bool) elasticapm.TraceContext {
	tracer, _ := transporttest.NewRecorderTracer()
	defer tracer.Close()
	tracer.SetSampler(samplerFunc(func(elasticapm.TraceContext) bool {
		panic("nope")
	}))

	opts := elasticapm.TransactionOptions{
		TraceContext: elasticapm.TraceContext{
			Trace: elasticapm.TraceID{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15},
			Span:  elasticapm.SpanID{0, 1, 2, 3, 4, 5, 6, 7},
		},
	}
	opts.TraceContext.Options = opts.TraceContext.Options.WithRequested(requested)
	opts.TraceContext.Options = opts.TraceContext.Options.WithMaybeRecorded(maybeRecorded)

	tx := tracer.StartTransactionOptions("name", "type", opts)
	result := tx.TraceContext()
	tx.Discard()
	return result
}

func TestStartTransactionInvalidTraceContext(t *testing.T) {
	startTransactionInvalidTraceContext(t, elasticapm.TraceContext{
		// Trace is all zeroes, which is invalid.
		Span: elasticapm.SpanID{0, 1, 2, 3, 4, 5, 6, 7},
	})
	startTransactionInvalidTraceContext(t, elasticapm.TraceContext{
		Trace: elasticapm.TraceID{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15},
		// Span is all zeroes, which is invalid.
	})
}

func startTransactionInvalidTraceContext(t *testing.T, traceContext elasticapm.TraceContext) {
	tracer, _ := transporttest.NewRecorderTracer()
	defer tracer.Close()

	var samplerCalled bool
	tracer.SetSampler(samplerFunc(func(elasticapm.TraceContext) bool {
		samplerCalled = true
		return true
	}))

	opts := elasticapm.TransactionOptions{TraceContext: traceContext}
	tx := tracer.StartTransactionOptions("name", "type", opts)
	tx.Discard()
	assert.True(t, samplerCalled)
}

type samplerFunc func(elasticapm.TraceContext) bool

func (f samplerFunc) Sample(t elasticapm.TraceContext) bool {
	return f(t)
}
