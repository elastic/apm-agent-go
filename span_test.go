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
	"os"
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

func TestSpanParentID(t *testing.T) {
	tracer := apmtest.NewRecordingTracer()
	defer tracer.Close()

	tx := tracer.StartTransaction("name", "type")
	span := tx.StartSpan("name", "type", nil)
	traceContext := tx.TraceContext()
	parentID := span.ParentID()

	span.End()
	tx.End()
	// Assert that the parentID is not empty when the span hasn't been ended.
	// And that the Span's parentID equals the traceContext Span.
	assert.NotEqual(t, parentID, apm.SpanID{})
	assert.Equal(t, traceContext.Span, parentID)

	// Assert that the parentID is not empty after the span has ended.
	assert.NotZero(t, span.ParentID())
	assert.Equal(t, traceContext.Span, span.ParentID())

	tracer.Flush(nil)
	payloads := tracer.Payloads()
	require.Len(t, payloads.Spans, 1)
	assert.Equal(t, model.SpanID(parentID), payloads.Spans[0].ParentID)
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

func TestStartExitSpan(t *testing.T) {
	_, spans, _ := apmtest.WithTransaction(func(ctx context.Context) {
		span, _ := apm.StartSpanOptions(ctx, "name", "type", apm.SpanOptions{ExitSpan: true})
		assert.True(t, span.IsExitSpan())
		span.End()
	})
	require.Len(t, spans, 1)
	// When the context's DestinationService is not explicitly set, ending
	// the exit span will assign the value.
	assert.Equal(t, spans[0].Context.Destination.Service.Resource, "type")

	tracer := apmtest.NewRecordingTracer()
	defer tracer.Close()

	tx := tracer.StartTransaction("name", "type")
	span := tx.StartSpanOptions("name", "type", apm.SpanOptions{ExitSpan: true})
	assert.True(t, span.IsExitSpan())
	// when the parent span is an exit span, any children should be noops.
	span2 := tx.StartSpan("name", "type", span)
	assert.True(t, span2.Dropped())
	span.End()
	span2.End()
	// Spans should still be marked as an exit span after they've been
	// ended.
	assert.True(t, span.IsExitSpan())
}

func TestCompressSpanExactMatch(t *testing.T) {
	tests := []struct {
		name               string
		compressionEnabled bool
		setup              func(t *testing.T) func()
		assertFunc         func(t *testing.T, tx model.Transaction, spans []model.Span)
	}{
		{
			name:               "CompressFalse",
			compressionEnabled: false,
			assertFunc: func(t *testing.T, tx model.Transaction, spans []model.Span) {
				require.NotEmpty(t, tx)
				require.Equal(t, 14, len(spans))
				for _, span := range spans {
					require.Nil(t, span.Composite)
				}
			},
		},
		{
			name:               "CompressTrueSettingTweak",
			compressionEnabled: true,
			setup: func(t *testing.T) func() {
				envVarName := "ELASTIC_APM_SPAN_COMPRESSION_EXACT_MATCH_MAX_DURATION"
				og := os.Getenv(envVarName)
				os.Setenv(envVarName, "1ms")
				return func() { os.Setenv(envVarName, og) }
			},
			assertFunc: func(t *testing.T, tx model.Transaction, spans []model.Span) {
				require.NotEmpty(t, tx)
				// This setting
				require.Equal(t, 5, len(spans))
				for _, span := range spans[1:] {
					require.Nil(t, span.Composite)
				}
			},
		},
		{
			name:               "CompressSpanCount4",
			compressionEnabled: true,
			assertFunc: func(t *testing.T, tx model.Transaction, spans []model.Span) {
				var composite = spans[0]
				assert.Equal(t, composite.Context.Destination.Service.Resource, "mysql")
				compositeSpanCount := 11
				assert.Equal(t, composite.Composite.Count, compositeSpanCount)
				assert.Equal(t, composite.Composite.CompressionStrategy, "exact_match")
				// Sum should be at least the time that each span ran for.
				assert.Greater(t, composite.Composite.Sum,
					float64(int64(compositeSpanCount)*100*time.Nanosecond.Milliseconds()),
				)

				for _, span := range spans {
					if span.Type == "mysql" {
						continue
					}
					assert.Nil(t, span.Composite)
				}
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.setup != nil {
				t.Cleanup(test.setup(t))
			}

			tracer := apmtest.NewRecordingTracer()
			tracer.SetSpanCompressionEnabled(test.compressionEnabled)

			// When compression is enabled:
			// Compress 10 spans into 1 and add another span with a different type
			// [                    Transaction                    ]
			//  [ mysql (11) ] [ request ] [ internal ] [ request ]
			//
			tx, spans, _ := tracer.WithTransaction(func(ctx context.Context) {
				exitSpanOpt := apm.SpanOptions{ExitSpan: true}
				for i := 0; i < 10; i++ {
					span, _ := apm.StartSpanOptions(ctx, "SELECT * FROM users", "mysql", exitSpanOpt)
					<-time.After(100 * time.Nanosecond)
					span.End()
				}
				{
					span, _ := apm.StartSpanOptions(ctx, "SELECT * FROM users", "mysql", exitSpanOpt)
					<-time.After(2 * time.Millisecond)
					span.End()
				}

				// None of these should be added to the composite.
				{
					span, _ := apm.StartSpanOptions(ctx, "GET /", "request", exitSpanOpt)
					<-time.After(100 * time.Nanosecond)
					span.End()
				}
				{
					// Not an exit span, should not be compressed
					span, _ := apm.StartSpan(ctx, "calculate complex", "internal")
					<-time.After(100 * time.Nanosecond)
					span.End()
				}
				{
					// Exit span, this is a good candidate to be compressed, but
					// since it can't be compressed with the last request type ("internal")
					span, _ := apm.StartSpanOptions(ctx, "GET /", "request", exitSpanOpt)
					<-time.After(100 * time.Nanosecond)
					span.End()
				}
			})
			defer func() {
				if t.Failed() {
					apmtest.WriteTraceWaterfall(os.Stdout, tx, spans)
				}
			}()

			if test.assertFunc != nil {
				test.assertFunc(t, tx, spans)
			}
		})
	}
}

func TestCompressSpanSameKind(t *testing.T) {
	tracer := apmtest.NewRecordingTracer()
	tracer.SetSpanCompressionEnabled(true)

	// Compress 5 spans into 1 and add another span with a different type
	// [       Transaction       ]
	//  [ mysql ] [ request (5) ]
	//
	tx, spans, _ := tracer.WithTransaction(func(ctx context.Context) {
		exitSpanOpt := apm.SpanOptions{ExitSpan: true}
		path := []string{"/a", "/b", "/c", "/d", "/e"}
		// Span is compressable, but cannot be compressed since the next span
		// is not the same kind. It gets published.
		{
			span, _ := apm.StartSpanOptions(ctx, "SELECT * FROM users", "mysql", exitSpanOpt)
			<-time.After(100 * time.Nanosecond)
			span.End()
		}
		// These spans should be compressed into 1.
		for i := 0; i < 5; i++ {
			uri := fmt.Sprint("GET ", path[i])
			span, _ := apm.StartSpanOptions(ctx, uri, "request", exitSpanOpt)
			<-time.After(100 * time.Nanosecond)
			span.End()
		}
	})

	defer func() {
		if t.Failed() {
			apmtest.WriteTraceWaterfall(os.Stdout, tx, spans)
		}
	}()

	assert.NotNil(t, tx)
	assert.Equal(t, 2, len(spans))

	mysqlSpan := spans[0]
	assert.Equal(t, mysqlSpan.Context.Destination.Service.Resource, "mysql")
	assert.Nil(t, mysqlSpan.Composite)

	requestSpan := spans[1]
	assert.Equal(t, requestSpan.Context.Destination.Service.Resource, "request")
	assert.NotNil(t, requestSpan.Composite)
	assert.Equal(t, requestSpan.Composite.Count, 5)
	assert.Equal(t, requestSpan.Name, "Calls to request")
	// Check that the aggregate sum is at least the duration of the time we
	// we waited for.
	assert.Greater(t, requestSpan.Composite.Sum, float64(5*100/time.Millisecond))

	// Check that the total composite span duration is at least 5 milliseconds.
	assert.Greater(t, requestSpan.Duration, float64(5*100/time.Millisecond))
}

func TestCompressSpanSameKindParentSpan(t *testing.T) {
	tracer := apmtest.NewRecordingTracer()
	tracer.SetSpanCompressionEnabled(true)

	// This test case covers spans that have other spans as parents.
	tx, spans, _ := tracer.WithTransaction(func(ctx context.Context) {
		{
			// Doesn't compress any spans since none meet the necessary conditions
			// the "request" type are both the same type but the parent
			// [       Transaction       ...snip
			//  [       internal op     ]
			//   [        request      ]
			//          [   request   ]
			//
			parent, ctx := apm.StartSpan(ctx, "internal op", "internal")
			// Have span propagate context downstream, this should not allow for
			// compression
			child, ctx := apm.StartSpan(ctx, "GET /resource", "request")
			grandChild, _ := apm.StartSpanOptions(ctx, "GET /different", "request", apm.SpanOptions{
				ExitSpan: true,
				Parent:   child.TraceContext(),
			})
			<-time.After(500 * time.Nanosecond)
			grandChild.End()
			child.End()
			parent.End()
		}
		{
			// Compresses the last two spans together since they are both exit
			// spans, same "request" type, don't propagate ctx and succeed.
			// ..continued  Transaction   ]
			//  [       internal op     ]
			//       [  request (2)   ]  ( [GET /res] [GET /diff] )
			//
			exitSpanOpts := apm.SpanOptions{ExitSpan: true}
			parent, ctx := apm.StartSpan(ctx, "another op", "internal")
			child, _ := apm.StartSpanOptions(ctx, "GET /res", "request", exitSpanOpts)
			<-time.After(300 * time.Nanosecond)

			otherChild, _ := apm.StartSpanOptions(ctx, "GET /diff", "request", exitSpanOpts)
			<-time.After(300 * time.Nanosecond)

			otherChild.End()
			child.End()

			parent.End()
		}
	})

	defer func() {
		if t.Failed() {
			apmtest.WriteTraceWaterfall(os.Stdout, tx, spans)
		}
	}()
	assert.NotNil(t, tx)
	assert.Equal(t, 5, len(spans))

	// Since the spans are started very close together, even a time.Sleep
	// doesn't return the spans in a deterministic order, which is OK.
	var compositeSpan model.Span
	var compositeParent model.Span
	for _, span := range spans {
		if span.Name == "another op" && span.Type == "internal" {
			compositeParent = span
		}
	}
	for _, span := range spans {
		if span.Composite != nil {
			compositeSpan = span
			assert.Equal(t, "Calls to request", span.Name)
			assert.Equal(t, "request", span.Type)
		}
	}

	assert.NotNil(t, compositeSpan)
	assert.NotNil(t, compositeSpan.Composite)
	assert.Equal(t, compositeSpan.Composite.Count, 2)
	assert.Equal(t, compositeSpan.ParentID, compositeParent.ID)
	assert.GreaterOrEqual(t, compositeParent.Duration, compositeSpan.Duration)
}

func TestExitSpanDoesNotOverwriteDestinationServiceResource(t *testing.T) {
	_, spans, _ := apmtest.WithTransaction(func(ctx context.Context) {
		span, _ := apm.StartSpanOptions(ctx, "name", "type", apm.SpanOptions{ExitSpan: true})
		assert.True(t, span.IsExitSpan())
		span.Context.SetDestinationService(apm.DestinationServiceSpanContext{
			Resource: "my-custom-resource",
		})
		span.End()
	})
	require.Len(t, spans, 1)
	assert.Equal(t, spans[0].Context.Destination.Service.Resource, "my-custom-resource")
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

func TestSpanSampleRate(t *testing.T) {
	tracer := apmtest.NewRecordingTracer()
	defer tracer.Close()
	tracer.SetSampler(apm.NewRatioSampler(0.55555))

	tx := tracer.StartTransactionOptions("name", "type", apm.TransactionOptions{
		// Use a known transaction ID for deterministic sampling.
		TransactionID: apm.SpanID{1, 2, 3, 4, 5, 6, 7, 8},
	})
	s1 := tx.StartSpan("name", "type", nil)
	s2 := tx.StartSpan("name", "type", s1)
	s2.End()
	s1.End()
	tx.End()
	tracer.Flush(nil)

	payloads := tracer.Payloads()
	assert.Equal(t, 0.5556, *payloads.Transactions[0].SampleRate)
	assert.Equal(t, 0.5556, *payloads.Spans[0].SampleRate)
	assert.Equal(t, 0.5556, *payloads.Spans[1].SampleRate)
}
