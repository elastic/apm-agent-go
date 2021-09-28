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
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.elastic.co/apm"
	"go.elastic.co/apm/apmtest"
	"go.elastic.co/apm/model"
	"go.elastic.co/apm/transport"
	"go.elastic.co/apm/transport/transporttest"
)

func TestStartTransactionTraceContextOptions(t *testing.T) {
	testStartTransactionTraceContextOptions(t, false)
	testStartTransactionTraceContextOptions(t, true)
}

func testStartTransactionTraceContextOptions(t *testing.T, recorded bool) {
	tracer, _ := transporttest.NewRecorderTracer()
	defer tracer.Close()
	tracer.SetSampler(samplerFunc(func(apm.TraceContext) bool {
		panic("nope")
	}))

	opts := apm.TransactionOptions{
		TraceContext: apm.TraceContext{
			Trace: apm.TraceID{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15},
			Span:  apm.SpanID{0, 1, 2, 3, 4, 5, 6, 7},
		},
	}
	opts.TraceContext.Options = opts.TraceContext.Options.WithRecorded(recorded)

	tx := tracer.StartTransactionOptions("name", "type", opts)
	result := tx.TraceContext()
	assert.Equal(t, recorded, result.Options.Recorded())
	tx.Discard()
}

func TestStartTransactionInvalidTraceContext(t *testing.T) {
	startTransactionInvalidTraceContext(t, apm.TraceContext{
		// Trace is all zeroes, which is invalid.
		Span: apm.SpanID{0, 1, 2, 3, 4, 5, 6, 7},
	})
}

func startTransactionInvalidTraceContext(t *testing.T, traceContext apm.TraceContext) {
	tracer, _ := transporttest.NewRecorderTracer()
	defer tracer.Close()

	var samplerCalled bool
	tracer.SetSampler(samplerFunc(func(apm.TraceContext) bool {
		samplerCalled = true
		return true
	}))

	opts := apm.TransactionOptions{TraceContext: traceContext}
	tx := tracer.StartTransactionOptions("name", "type", opts)
	assert.True(t, samplerCalled)
	tx.Discard()
}

func TestStartTransactionTraceParentSpanIDSpecified(t *testing.T) {
	startTransactionIDSpecified(t, apm.TraceContext{
		Trace: apm.TraceID{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15},
		Span:  apm.SpanID{0, 1, 2, 3, 4, 5, 6, 7},
	})
}

func TestStartTransactionTraceIDSpecified(t *testing.T) {
	startTransactionIDSpecified(t, apm.TraceContext{
		Trace: apm.TraceID{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15},
	})
}

func TestStartTransactionIDSpecified(t *testing.T) {
	startTransactionIDSpecified(t, apm.TraceContext{})
}

func startTransactionIDSpecified(t *testing.T, traceContext apm.TraceContext) {
	tracer, _ := transporttest.NewRecorderTracer()
	defer tracer.Close()

	opts := apm.TransactionOptions{
		TraceContext:  traceContext,
		TransactionID: apm.SpanID{0, 1, 2, 3, 4, 5, 6, 7},
	}
	tx := tracer.StartTransactionOptions("name", "type", opts)
	assert.Equal(t, opts.TransactionID, tx.TraceContext().Span)
	tx.Discard()
}

func TestTransactionEnsureParent(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	tx := tracer.StartTransaction("name", "type")
	traceContext := tx.TraceContext()

	parentSpan := tx.EnsureParent()
	assert.NotZero(t, parentSpan)
	assert.NotEqual(t, traceContext.Span, parentSpan)

	// EnsureParent is idempotent.
	parentSpan2 := tx.EnsureParent()
	assert.Equal(t, parentSpan, parentSpan2)

	tx.End()

	// For an ended transaction, EnsureParent will return a zero value
	// even if the transaction had a parent at the time it was ended.
	parentSpan3 := tx.EnsureParent()
	assert.Zero(t, parentSpan3)

	tracer.Flush(nil)
	payloads := transport.Payloads()
	require.Len(t, payloads.Transactions, 1)
	assert.Equal(t, model.SpanID(parentSpan), payloads.Transactions[0].ParentID)
}

func TestTransactionParentID(t *testing.T) {
	tracer := apmtest.NewRecordingTracer()
	defer tracer.Close()

	tx := tracer.StartTransaction("name", "type")
	traceContext := tx.TraceContext()

	// root tx has no parent.
	parentSpan := tx.ParentID()
	assert.Zero(t, parentSpan)

	// Create a child transaction with the TraceContext from the parent.
	txChild := tracer.StartTransactionOptions("child", "type", apm.TransactionOptions{
		TraceContext: tx.TraceContext(),
	})

	// Assert that the Parent ID isn't zero and matches the parent ID.
	// Parent TX ID
	parentTxID := traceContext.Span
	childSpanParentID := txChild.ParentID()
	assert.NotZero(t, childSpanParentID)
	assert.Equal(t, parentTxID, childSpanParentID)

	txChild.End()
	tx.End()

	// Assert that we can obtain the parent ID even after the transaction
	// has ended.
	assert.NotZero(t, txChild.ParentID())

	tracer.Flush(nil)
	payloads := tracer.Payloads()
	require.Len(t, payloads.Transactions, 2)

	// First recorded transaction Parent ID matches the child's Parent ID
	assert.Equal(t, model.SpanID(childSpanParentID), payloads.Transactions[0].ParentID)
	// First recorded transaction Parent ID matches the parent transaction ID.
	assert.Equal(t, model.SpanID(parentTxID), payloads.Transactions[0].ParentID)

	// Parent transaction has a zero ParentID.
	assert.Zero(t, payloads.Transactions[1].ParentID)
}

func TestTransactionParentIDWithEnsureParent(t *testing.T) {
	tracer := apmtest.NewRecordingTracer()
	defer tracer.Close()

	tx := tracer.StartTransaction("name", "type")

	rootParentIDEmpty := tx.ParentID()
	assert.Zero(t, rootParentIDEmpty)

	ensureParentResult := tx.EnsureParent()
	assert.NotZero(t, ensureParentResult)

	rootParentIDNotEmpty := tx.ParentID()
	assert.Equal(t, ensureParentResult, rootParentIDNotEmpty)

	tx.End()

	tracer.Flush(nil)
	payloads := tracer.Payloads()
	require.Len(t, payloads.Transactions, 1)

	assert.Equal(t, model.SpanID(rootParentIDNotEmpty), payloads.Transactions[0].ParentID)
}

func TestTransactionContextNotSampled(t *testing.T) {
	tracer := apmtest.NewRecordingTracer()
	defer tracer.Close()
	tracer.SetSampler(samplerFunc(func(apm.TraceContext) bool { return false }))

	tx := tracer.StartTransaction("name", "type")
	tx.Context.SetLabel("foo", "bar")
	tx.End()
	tracer.Flush(nil)

	payloads := tracer.Payloads()
	require.Len(t, payloads.Transactions, 1)
	assert.Nil(t, payloads.Transactions[0].Context)
}

func TestTransactionNotRecording(t *testing.T) {
	tracer := apmtest.NewRecordingTracer()
	defer tracer.Close()
	tracer.SetRecording(false)
	tracer.SetSampler(samplerFunc(func(apm.TraceContext) bool {
		panic("should not be called")
	}))

	tx := tracer.StartTransaction("name", "type")
	require.NotNil(t, tx)
	require.NotNil(t, tx.TransactionData)
	tx.End()
	require.Nil(t, tx.TransactionData)
	tracer.Flush(nil)

	payloads := tracer.Payloads()
	require.Empty(t, payloads.Transactions)
}

func TestTransactionSampleRate(t *testing.T) {
	type test struct {
		actualSampleRate   float64
		recordedSampleRate float64
		expectedTraceState string
	}
	tests := []test{
		{0, 0, "es=s:0"},
		{1, 1, "es=s:1"},
		{0.00001, 0.0001, "es=s:0.0001"},
		{0.55554, 0.5555, "es=s:0.5555"},
		{0.55555, 0.5556, "es=s:0.5556"},
		{0.55556, 0.5556, "es=s:0.5556"},
	}
	for _, test := range tests {
		test := test // copy for closure
		t.Run(fmt.Sprintf("%v", test.actualSampleRate), func(t *testing.T) {
			tracer := apmtest.NewRecordingTracer()
			defer tracer.Close()

			tracer.SetSampler(apm.NewRatioSampler(test.actualSampleRate))
			tx := tracer.StartTransactionOptions("name", "type", apm.TransactionOptions{
				// Use a known transaction ID for deterministic sampling.
				TransactionID: apm.SpanID{0, 1, 2, 3, 4, 5, 6, 7},
			})
			tx.End()
			tracer.Flush(nil)

			payloads := tracer.Payloads()
			assert.Equal(t, test.recordedSampleRate, *payloads.Transactions[0].SampleRate)
			assert.Equal(t, test.expectedTraceState, tx.TraceContext().State.String())
		})
	}
}

func TestTransactionUnsampledSampleRate(t *testing.T) {
	tracer := apmtest.NewRecordingTracer()
	defer tracer.Close()
	tracer.SetSampler(apm.NewRatioSampler(0.5))

	// Create transactions until we get an unsampled one.
	//
	// Even though the configured sampling rate is 0.5,
	// we record sample_rate=0 to ensure the server does
	// not count the transaction toward metrics.
	var tx *apm.Transaction
	for {
		tx = tracer.StartTransactionOptions("name", "type", apm.TransactionOptions{})
		if !tx.Sampled() {
			tx.End()
			break
		}
		tx.Discard()
	}
	tracer.Flush(nil)

	payloads := tracer.Payloads()
	assert.Equal(t, float64(0), *payloads.Transactions[0].SampleRate)
	assert.Equal(t, "es=s:0", tx.TraceContext().State.String())
}

func TestTransactionSampleRatePropagation(t *testing.T) {
	tracer := apmtest.NewRecordingTracer()
	defer tracer.Close()

	for _, tracestate := range []apm.TraceState{
		apm.NewTraceState(apm.TraceStateEntry{Key: "es", Value: "s:0.5"}),
		apm.NewTraceState(apm.TraceStateEntry{Key: "es", Value: "x:y;s:0.5;zz:y"}),
		apm.NewTraceState(
			apm.TraceStateEntry{Key: "other", Value: "s:1.0"},
			apm.TraceStateEntry{Key: "es", Value: "s:0.5"},
		),
	} {
		tx := tracer.StartTransactionOptions("name", "type", apm.TransactionOptions{
			TraceContext: apm.TraceContext{
				Trace: apm.TraceID{1},
				Span:  apm.SpanID{1},
				State: tracestate,
			},
		})
		tx.End()
	}
	tracer.Flush(nil)

	payloads := tracer.Payloads()
	assert.Len(t, payloads.Transactions, 3)
	for _, tx := range payloads.Transactions {
		assert.Equal(t, 0.5, *tx.SampleRate)
	}
}

func TestTransactionSampleRateOmission(t *testing.T) {
	tracer := apmtest.NewRecordingTracer()
	defer tracer.Close()

	// For downstream transactions, sample_rate should be
	// omitted if a valid value is not found in tracestate.
	for _, tracestate := range []apm.TraceState{
		apm.TraceState{}, // empty
		apm.NewTraceState(apm.TraceStateEntry{Key: "other", Value: "s:1.0"}), // not "es", ignored
		apm.NewTraceState(apm.TraceStateEntry{Key: "es", Value: "s:123.0"}),  // out of range
		apm.NewTraceState(apm.TraceStateEntry{Key: "es", Value: ""}),         // 's' missing
		apm.NewTraceState(apm.TraceStateEntry{Key: "es", Value: "wat"}),      // malformed
	} {
		for _, sampled := range []bool{false, true} {
			tx := tracer.StartTransactionOptions("name", "type", apm.TransactionOptions{
				TraceContext: apm.TraceContext{
					Trace:   apm.TraceID{1},
					Span:    apm.SpanID{1},
					Options: apm.TraceOptions(0).WithRecorded(sampled),
					State:   tracestate,
				},
			})
			tx.End()
		}
	}
	tracer.Flush(nil)

	payloads := tracer.Payloads()
	assert.Len(t, payloads.Transactions, 10)
	for _, tx := range payloads.Transactions {
		assert.Nil(t, tx.SampleRate)
	}
}

func TestTransactionDiscard(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	tx := tracer.StartTransaction("name", "type")
	tx.Discard()
	assert.Nil(t, tx.TransactionData)
	tx.End() // ending after discarding should be a no-op

	tracer.Flush(nil)
	payloads := transport.Payloads()
	require.Empty(t, payloads)
}

func TestTransactionDroppedSpansStats(t *testing.T) {
	// Creates a huge transaction which drops all spans over 500 spans
	// and asserts a few things, after 500 spans have been added to the tx:
	// * Any spans that go over the 500 max span limit are dropped.
	// * Of these dropped spans, collect timing statistics for them.
	// * The limit for distinct span type/subtype/outcome/destination is 128.
	// * When the list of items has reached 128, any spans that match entries
	//   of the existing spans are still aggregated.
	tx, spans, _ := apmtest.WithTransaction(func(ctx context.Context) {
		for i := 0; i < 1000; i++ {
			span, _ := apm.StartSpan(ctx,
				fmt.Sprintf("GET %d", i),
				fmt.Sprintf("request_%d", i),
			)
			span.Context.SetDestinationService(apm.DestinationServiceSpanContext{
				Resource: "host:443",
			})
			<-time.After(10 * time.Microsecond)
			span.End()
		}

		for i := 0; i < 100; i++ {
			span, _ := apm.StartSpan(ctx, "GET 501", "request_501")
			span.Context.SetDestinationService(apm.DestinationServiceSpanContext{
				Resource: "host:443",
			})
			<-time.After(10 * time.Microsecond)
			span.End()
		}

		for i := 0; i < 50; i++ {
			span, _ := apm.StartSpan(ctx, "GET 600", "request_600")
			span.Context.SetDestinationService(apm.DestinationServiceSpanContext{
				Resource: "host:443",
			})
			<-time.After(10 * time.Microsecond)
			span.End()
		}
	})

	// We drop any spans when we reach 500 reported spans.
	require.Len(t, spans, 500)
	// We've dropped 600 spans.
	require.Equal(t, 650, tx.SpanCount.Dropped)
	// We've dropped 650 spans, we only report statistics for up to 128 distinct
	// groups.
	require.Len(t, tx.DroppedSpansStats, 128)

	// Ensure that the extra spans we generated are aggregated
	for _, span := range tx.DroppedSpansStats {
		if span.Type == "request_501" {
			assert.Equal(t, 101, span.Duration.Count)
			// Assert that the sum of the duration is at least
			// 1.1 ms.
			assert.Greater(t, span.Duration.Sum.Us, int64(1100))
		} else if span.Type == "request_600" {
			assert.Equal(t, 51, span.Duration.Count)
			// Assert that the sum of the duration is at least
			// 0.5 ms.
			assert.Greater(t, span.Duration.Sum.Us, int64(510))
		} else {
			assert.Equal(t, 1, span.Duration.Count)
			assert.GreaterOrEqual(t, span.Duration.Sum.Us, int64(10))
		}
	}
}

func TestTransactionDroppedSpansStatsChangedLimit(t *testing.T) {
	// Creates a huge transaction which drops around anything over 100 spans
	// and asserts a few things, after 100 spans have been added to the tx:
	// * Any spans that go over the 100 max span limit are dropped.
	// * Of these dropped spans, collect timing statistics for them.
	// * The limit for distinct span type/subtype/outcome/destination is 128.
	// * When the list of items has reached 128, any spans that match entries
	//   of the existing spans are still aggregated.
	const maxSpanLimit = 100
	tracer := apmtest.NewRecordingTracer()
	// Limit max spans to maxSpanLimit.
	tracer.SetMaxSpans(maxSpanLimit)

	tx, spans, _ := tracer.WithTransaction(func(ctx context.Context) {
		for i := 0; i < 300; i++ {
			span, _ := apm.StartSpan(ctx,
				fmt.Sprintf("GET %d", i),
				fmt.Sprintf("request_%d", i),
			)
			span.Context.SetDestinationService(apm.DestinationServiceSpanContext{
				Resource: "host:443",
			})
			<-time.After(10 * time.Microsecond)
			span.End()
		}

		for i := 0; i < 50; i++ {
			span, _ := apm.StartSpan(ctx, "GET 51", "request_51")
			span.Context.SetDestinationService(apm.DestinationServiceSpanContext{
				Resource: "host:443",
			})
			<-time.After(10 * time.Microsecond)
			span.End()
		}

		for i := 0; i < 20; i++ {
			span, _ := apm.StartSpan(ctx, "GET 60", "request_60")
			span.Context.SetDestinationService(apm.DestinationServiceSpanContext{
				Resource: "host:443",
			})
			<-time.After(10 * time.Microsecond)
			span.End()
		}
	})

	// We drop any spans when we reach 100 reported spans.
	require.Len(t, spans, 100)
	// We've dropped 170 spans.
	require.Equal(t, 270, tx.SpanCount.Dropped)
	// We've dropped 270 spans, we only report statistics for up to 128 distinct
	// groups.
	require.Len(t, tx.DroppedSpansStats, 128)

	// Ensure that the extra spans we generated are aggregated
	for _, span := range tx.DroppedSpansStats {
		if span.Type == "request_51" {
			assert.Equal(t, 50, span.Duration.Count)
			// Assert that the sum of the duration is at least
			// 0.5 ms.
			assert.Greater(t, span.Duration.Sum.Us, int64(500))
		} else if span.Type == "request_60" {
			assert.Equal(t, 20, span.Duration.Count)
			// Assert that the sum of the duration is at least
			// 0.2 ms.
			assert.Greater(t, span.Duration.Sum.Us, int64(20))
		} else {
			assert.Equal(t, 1, span.Duration.Count)
			assert.GreaterOrEqual(t, span.Duration.Sum.Us, int64(10))
		}
	}
}

func BenchmarkTransaction(b *testing.B) {
	tracer, err := apm.NewTracer("service", "")
	require.NoError(b, err)

	tracer.Transport = transport.Discard
	defer tracer.Close()

	names := []string{}
	for i := 0; i < 1000; i++ {
		names = append(names, fmt.Sprintf("/some/route/%d", i))
	}

	var mu sync.Mutex
	globalRand := rand.New(rand.NewSource(time.Now().UnixNano()))
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		mu.Lock()
		rand := rand.New(rand.NewSource(globalRand.Int63()))
		mu.Unlock()
		for pb.Next() {
			tx := tracer.StartTransaction(names[rand.Intn(len(names))], "type")
			tx.End()
		}
	})
}

type samplerFunc func(apm.TraceContext) bool

func (f samplerFunc) Sample(t apm.TraceContext) bool {
	return f(t)
}
