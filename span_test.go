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
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.elastic.co/apm/v2"
	"go.elastic.co/apm/v2/apmtest"
	"go.elastic.co/apm/v2/model"
	"go.elastic.co/apm/v2/transport/transporttest"
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

func TestSpanEnsureType(t *testing.T) {
	tracer := apmtest.NewRecordingTracer()
	defer tracer.Close()

	tx := tracer.StartTransaction("name", "type")
	span := tx.StartSpan("name", "", nil)
	span.End()
	tx.End()
	tracer.Flush(nil)

	payloads := tracer.Payloads()
	require.Len(t, payloads.Spans, 1)

	assert.NotEmpty(t, payloads.Spans[0].Type)
}

func TestSpanLink(t *testing.T) {
	tracer := apmtest.NewRecordingTracer()
	defer tracer.Close()

	links := []apm.SpanLink{
		{Trace: apm.TraceID{1}, Span: apm.SpanID{1}},
		{Trace: apm.TraceID{2}, Span: apm.SpanID{2}},
	}

	tx := tracer.StartTransaction("name", "type")
	span := tx.StartSpanOptions("name", "type", apm.SpanOptions{
		Links: links,
	})

	span.End()
	tx.End()

	tracer.Flush(nil)

	payloads := tracer.Payloads()
	require.Len(t, payloads.Spans, 1)
	require.Len(t, payloads.Spans[0].Links, len(links))

	// Assert span links are identical.
	expectedLinks := []model.SpanLink{
		{TraceID: model.TraceID{1}, SpanID: model.SpanID{1}},
		{TraceID: model.TraceID{2}, SpanID: model.SpanID{2}},
	}
	assert.Equal(t, expectedLinks, payloads.Spans[0].Links)
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
		span.Duration = 2 * time.Millisecond
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

func TestFramesMinDurationSpecialCases(t *testing.T) {
	tracer := apmtest.NewRecordingTracer()

	// verify that no stacktraces are recorded
	tracer.SetSpanFramesMinDuration(0)
	tx := tracer.StartTransaction("name", "type")
	span := tx.StartSpan("span", "span", nil)
	span.End()
	tx.End()

	tracer.Flush(nil)
	tracer.Close()

	spans := tracer.Payloads().Spans
	require.Len(t, spans, 1)
	assert.Len(t, spans[0].Stacktrace, 0)

	// verify that stacktraces are always recorded
	tracer = apmtest.NewRecordingTracer()
	defer tracer.Close()
	tracer.SetSpanFramesMinDuration(-1)
	tx = tracer.StartTransaction("name", "type")
	span = tx.StartSpan("span2", "span2", nil)
	span.End()
	tx.End()

	tracer.Flush(nil)

	spans = tracer.Payloads().Spans
	require.Len(t, spans, 1)
	assert.NotEmpty(t, spans[0].Stacktrace)
}

func TestCompressSpanNonSiblings(t *testing.T) {
	// Asserts that non sibling spans are not compressed.
	tracer := apmtest.NewRecordingTracer()
	defer tracer.Close()

	tracer.SetSpanCompressionEnabled(true)
	// Avoid the spans from being dropped by fast exit spans.
	tracer.SetExitSpanMinDuration(time.Nanosecond)

	tx := tracer.StartTransaction("name", "type")
	parent := tx.StartSpan("parent", "parent", nil)

	createSpans := []struct {
		name, typ string
		parent    apm.TraceContext
	}{
		{name: "not compressed", typ: "internal", parent: parent.TraceContext()},
		{name: "not compressed", typ: "internal", parent: tx.TraceContext()},
		{name: "compressed", typ: "internal", parent: parent.TraceContext()},
		{name: "compressed", typ: "internal", parent: parent.TraceContext()},
		{name: "compressed", typ: "different", parent: tx.TraceContext()},
		{name: "compressed", typ: "different", parent: tx.TraceContext()},
	}
	for _, span := range createSpans {
		span := tx.StartSpanOptions(span.name, span.typ, apm.SpanOptions{
			ExitSpan: true, Parent: span.parent,
		})
		span.Duration = time.Millisecond
		span.End()
	}

	parent.End()
	tx.End()
	tracer.Flush(nil)

	spans := tracer.Payloads().Spans
	require.Len(t, spans, 5)

	// First two spans should not have been compressed together.
	require.Nil(t, spans[0].Composite)
	require.Nil(t, spans[1].Composite)

	assert.NotNil(t, spans[2].Composite)
	assert.Equal(t, 2, spans[2].Composite.Count)
	assert.Equal(t, float64(2), spans[2].Composite.Sum)
	assert.Equal(t, "exact_match", spans[2].Composite.CompressionStrategy)

	assert.NotNil(t, spans[3].Composite)
	assert.Equal(t, 2, spans[3].Composite.Count)
	assert.Equal(t, float64(2), spans[3].Composite.Sum)
	assert.Equal(t, "exact_match", spans[3].Composite.CompressionStrategy)
}

func TestCompressSpanExactMatch(t *testing.T) {
	// Aserts that that span compression works on compressable spans with
	// "exact_match" strategy.
	tests := []struct {
		setup              func(t *testing.T) func()
		assertFunc         func(t *testing.T, tx model.Transaction, spans []model.Span)
		name               string
		compressionEnabled bool
	}{
		// |______________transaction (095b51e1b6ca784c) - 2.0013ms_______________|
		// m
		// m
		// m
		// m
		// m
		// m
		// m
		// m
		// m
		// m
		// |___________________mysql SELECT * FROM users - 2ms____________________|
		//                                                                        r
		//                                                                        i
		//                                                                        r
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
		// |______________transaction (7d3254511f02b26b) - 2.0013ms_______________|
		// 10
		//  |___________________mysql SELECT * FROM users - 2ms___________________|
		//                                                                        r
		//                                                                        i
		//                                                                        r
		{
			name:               "CompressTrueSettingTweak",
			compressionEnabled: true,
			setup: func(*testing.T) func() {
				// This setting
				envVarName := "ELASTIC_APM_SPAN_COMPRESSION_EXACT_MATCH_MAX_DURATION"
				og := os.Getenv(envVarName)
				os.Setenv(envVarName, "1ms")
				return func() { os.Setenv(envVarName, og) }
			},
			assertFunc: func(t *testing.T, tx model.Transaction, spans []model.Span) {
				require.NotNil(t, tx)
				require.Equal(t, 5, len(spans))
				composite := spans[0]
				require.NotNil(t, composite.Composite)
				assert.Equal(t, "exact_match", composite.Composite.CompressionStrategy)
				assert.Equal(t, composite.Composite.Count, 10)
				assert.Equal(t, 0.001, composite.Composite.Sum)
				assert.Equal(t, 0.001, composite.Duration)

				for _, span := range spans[1:] {
					require.Nil(t, span.Composite)
				}
			},
		},
		// |______________transaction (5797fe58c6ccce29) - 2.0013ms_______________|
		// |_____________________11 Calls to mysql - 2.001ms______________________|
		//                                                                        r
		//                                                                        i
		//                                                                        r
		{
			name:               "CompressSpanCount4",
			compressionEnabled: true,
			assertFunc: func(t *testing.T, tx model.Transaction, spans []model.Span) {
				require.NotEmpty(t, tx)
				var composite = spans[0]
				assert.Equal(t, composite.Context.Destination.Service.Resource, "mysql")

				require.NotNil(t, composite.Composite)
				assert.Equal(t, composite.Composite.Count, 11)
				assert.Equal(t, "exact_match", composite.Composite.CompressionStrategy)
				// Sum should be at least the time that each span ran for. The
				// model time is in Milliseconds and the span duration should be
				// at least 2 Milliseconds
				assert.Equal(t, int(composite.Composite.Sum), 2)
				assert.Equal(t, int(composite.Duration), 2)

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
				defer test.setup(t)()
			}

			tracer := apmtest.NewRecordingTracer()
			tracer.SetExitSpanMinDuration(time.Nanosecond)
			defer tracer.Close()
			tracer.SetSpanCompressionEnabled(test.compressionEnabled)

			// When compression is enabled:
			// Compress 10 spans into 1 and add another span with a different type
			// [                    Transaction                    ]
			//  [ mysql (11) ] [ request ] [ internal ] [ request ]
			//
			txStart := time.Now()
			tx := tracer.StartTransactionOptions("name", "type",
				apm.TransactionOptions{Start: txStart},
			)
			currentTime := txStart
			for i := 0; i < 10; i++ {
				span := tx.StartSpanOptions("SELECT * FROM users", "mysql", apm.SpanOptions{
					ExitSpan: true, Start: currentTime,
				})
				span.Duration = 100 * time.Nanosecond
				currentTime = currentTime.Add(span.Duration)
				span.End()
			}
			// Compressed when the exact_match threshold is >= 2ms.
			{
				span := tx.StartSpanOptions("SELECT * FROM users", "mysql", apm.SpanOptions{
					ExitSpan: true, Start: currentTime,
				})
				span.Duration = 2 * time.Millisecond
				currentTime = currentTime.Add(span.Duration)
				span.End()
			}

			// None of these should be added to the composite.
			{
				span := tx.StartSpanOptions("GET /", "request", apm.SpanOptions{
					ExitSpan: true, Start: currentTime,
				})
				span.Duration = 100 * time.Nanosecond
				currentTime = currentTime.Add(span.Duration)
				span.End()
			}
			{
				// Not an exit span, should not be compressed
				span := tx.StartSpanOptions("calculate complex", "internal", apm.SpanOptions{
					Start: currentTime,
				})
				span.Duration = 100 * time.Nanosecond
				currentTime = currentTime.Add(span.Duration)
				span.End()
			}
			{
				// Exit span, this is a good candidate to be compressed, but
				// since it can't be compressed with the last request type ("internal")
				span := tx.StartSpanOptions("GET /", "request", apm.SpanOptions{
					ExitSpan: true, Start: currentTime,
				})
				span.Duration = 100 * time.Nanosecond
				currentTime = currentTime.Add(span.Duration)
				span.End()
			}
			tx.Duration = currentTime.Sub(txStart)
			tx.End()
			tracer.Flush(nil)

			transaction := tracer.Payloads().Transactions[0]
			spans := tracer.Payloads().Spans
			defer func() {
				if t.Failed() {
					apmtest.WriteTraceWaterfall(os.Stdout, transaction, spans)
					apmtest.WriteTraceTable(os.Stdout, transaction, spans)
				}
			}()

			if test.assertFunc != nil {
				test.assertFunc(t, transaction, spans)
			}
		})
	}
}

func TestCompressSpanSameKind(t *testing.T) {
	// Aserts that that span compression works on compressable spans with
	// "same_kind" strategy, and that different span types are not compressed.
	testCase := func(tracer *apmtest.RecordingTracer) (model.Transaction, []model.Span, func()) {
		txStart := time.Now()
		tx := tracer.StartTransactionOptions("name", "type",
			apm.TransactionOptions{Start: txStart},
		)
		currentTime := txStart

		// Span is compressable, but cannot be compressed since the next span
		// is not the same kind. It's published.
		{
			span := tx.StartSpanOptions("SELECT * FROM users", "mysql", apm.SpanOptions{
				ExitSpan: true, Start: currentTime,
			})
			span.Duration = 100 * time.Nanosecond
			currentTime = currentTime.Add(span.Duration)
			span.End()
		}
		// These should be compressed into 1 since they meet the compression
		// criteria.
		path := []string{"/a", "/b", "/c", "/d", "/e"}
		for i := 0; i < len(path); i++ {
			span := tx.StartSpanOptions(fmt.Sprint("GET ", path[i]), "request", apm.SpanOptions{
				ExitSpan: true, Start: currentTime,
			})
			span.Duration = 100 * time.Nanosecond
			currentTime = currentTime.Add(span.Duration)
			span.End()
		}
		// This span exceeds the default threshold (5ms) and won't be compressed.
		{
			span := tx.StartSpanOptions("GET /f", "request", apm.SpanOptions{
				ExitSpan: true, Start: currentTime,
			})
			span.Duration = 6 * time.Millisecond
			currentTime = currentTime.Add(span.Duration)
			span.End()
		}
		tx.Duration = currentTime.Sub(txStart)
		tx.End()
		tracer.Flush(nil)

		transaction := tracer.Payloads().Transactions[0]
		spans := tracer.Payloads().Spans
		debugFunc := func() {
			if t.Failed() {
				apmtest.WriteTraceWaterfall(os.Stdout, transaction, spans)
				apmtest.WriteTraceTable(os.Stdout, transaction, spans)
			}
		}
		return transaction, spans, debugFunc
	}

	t.Run("DefaultDisabled", func(t *testing.T) {
		// By default same kind compression is disabled thus count will be 7.
		tracer := apmtest.NewRecordingTracer()
		defer tracer.Close()
		tracer.SetSpanCompressionEnabled(true)
		// Don't drop fast exit spans.
		tracer.SetExitSpanMinDuration(0)

		_, spans, debugFunc := testCase(tracer)
		defer debugFunc()

		require.Equal(t, 7, len(spans))
		mysqlSpan := spans[0]
		assert.Equal(t, "mysql", mysqlSpan.Context.Destination.Service.Resource)
		assert.Nil(t, mysqlSpan.Composite)

		requestSpan := spans[1]
		assert.Equal(t, "request", requestSpan.Context.Destination.Service.Resource)
		require.Nil(t, requestSpan.Composite)
	})

	t.Run("10msThreshold", func(t *testing.T) {
		// With this threshold the composite count will be 6.
		os.Setenv("ELASTIC_APM_SPAN_COMPRESSION_SAME_KIND_MAX_DURATION", "10ms")
		defer os.Unsetenv("ELASTIC_APM_SPAN_COMPRESSION_SAME_KIND_MAX_DURATION")

		tracer := apmtest.NewRecordingTracer()
		defer tracer.Close()
		tracer.SetSpanCompressionEnabled(true)
		// Don't drop fast exit spans.
		tracer.SetExitSpanMinDuration(0)

		_, spans, debugFunc := testCase(tracer)
		defer debugFunc()

		mysqlSpan := spans[0]
		assert.Equal(t, mysqlSpan.Context.Destination.Service.Resource, "mysql")
		assert.Nil(t, mysqlSpan.Composite)

		requestSpan := spans[1]
		assert.Equal(t, requestSpan.Context.Destination.Service.Resource, "request")
		assert.NotNil(t, requestSpan.Composite)
		assert.Equal(t, 6, requestSpan.Composite.Count)
		assert.Equal(t, "Calls to request", requestSpan.Name)
		assert.Equal(t, "same_kind", requestSpan.Composite.CompressionStrategy)
		// Check that the aggregate sum is at least the duration of the time we
		// we waited for.
		assert.Greater(t, requestSpan.Composite.Sum, float64(5*100/time.Millisecond))

		// Check that the total composite span duration is at least 5 milliseconds.
		assert.Greater(t, requestSpan.Duration, float64(5*100/time.Millisecond))
	})
	t.Run("DefaultThresholdDropFastExitSpan", func(t *testing.T) {
		tracer := apmtest.NewRecordingTracer()
		defer tracer.Close()
		tracer.SetSpanCompressionEnabled(true)

		tx, spans, debugFunc := testCase(tracer)
		defer debugFunc()

		// drops all spans except the last request span.
		require.Equal(t, 1, len(spans))
		// Collects statistics about the dropped spans (request and mysql).
		require.Equal(t, 2, len(tx.DroppedSpansStats))
	})
}

func TestCompressSpanSameKindParentSpan(t *testing.T) {
	// This test asserts the span compression works when the spans are children
	// of another span.
	tracer := apmtest.NewRecordingTracer()
	tracer.SetSpanCompressionEnabled(true)
	tracer.SetExitSpanMinDuration(0)
	tracer.SetSpanCompressionSameKindMaxDuration(5 * time.Millisecond)

	// This test case covers spans that have other spans as parents.
	// |_______________transaction (6b1e4866252dea6f) - 1.45ms________________|
	// |__internal internal op - 700µs___|
	//      |request GET /r|
	//       |request G|
	//                                    |___internal another op - 750µs____|
	//                                         |2 Calls to re|
	txStart := time.Now()
	tx := tracer.StartTransactionOptions("name", "type",
		apm.TransactionOptions{Start: txStart},
	)

	ctx := apm.ContextWithTransaction(context.Background(), tx)
	currentTime := txStart
	{
		// Doesn't compress any spans since none meet the necessary conditions
		// the "request" type are both the same type but the parent
		parent, ctx := apm.StartSpanOptions(ctx, "internal op", "internal", apm.SpanOptions{
			Start: currentTime,
		})
		// Have span propagate context downstream, this should not allow for
		// compression
		child, ctx := apm.StartSpanOptions(ctx, "GET /resource", "request", apm.SpanOptions{
			Start: currentTime.Add(100 * time.Microsecond),
		})

		grandChild, _ := apm.StartSpanOptions(ctx, "GET /different", "request", apm.SpanOptions{
			ExitSpan: true,
			Start:    currentTime.Add(120 * time.Microsecond),
		})

		grandChild.Duration = 200 * time.Microsecond
		grandChild.End()

		child.Duration = 300 * time.Microsecond
		child.End()

		parent.Duration = 700 * time.Microsecond
		currentTime = currentTime.Add(parent.Duration)
		parent.End()
	}
	{
		// Compresses the last two spans together since they are  both exit
		// spans, same "request" type, don't propagate ctx and succeed.
		parent, ctx := apm.StartSpanOptions(ctx, "another op", "internal", apm.SpanOptions{
			Start: currentTime.Add(50 * time.Microsecond),
		})
		child, _ := apm.StartSpanOptions(ctx, "GET /res", "request", apm.SpanOptions{
			ExitSpan: true,
			Start:    currentTime.Add(120 * time.Microsecond),
		})

		otherChild, _ := apm.StartSpanOptions(ctx, "GET /diff", "request", apm.SpanOptions{
			ExitSpan: true,
			Start:    currentTime.Add(150 * time.Microsecond),
		})

		otherChild.Duration = 250 * time.Microsecond
		otherChild.End()

		child.Duration = 300 * time.Microsecond
		child.End()

		parent.Duration = 750 * time.Microsecond
		currentTime = currentTime.Add(parent.Duration)
		parent.End()
	}

	tx.Duration = currentTime.Sub(txStart)
	tx.End()
	tracer.Flush(nil)

	transaction := tracer.Payloads().Transactions[0]
	spans := tracer.Payloads().Spans

	defer func() {
		if t.Failed() {
			apmtest.WriteTraceTable(os.Stdout, transaction, spans)
			apmtest.WriteTraceWaterfall(os.Stdout, transaction, spans)
		}
	}()
	require.NotNil(t, transaction)
	assert.Equal(t, 5, len(spans))

	compositeSpan := spans[3]
	compositeParent := spans[4]
	require.NotNil(t, compositeSpan)
	require.NotNil(t, compositeSpan.Composite)
	assert.Equal(t, "Calls to request", compositeSpan.Name)
	assert.Equal(t, "request", compositeSpan.Type)
	assert.Equal(t, "internal", compositeParent.Type)
	assert.Equal(t, compositeSpan.Composite.Count, 2)
	assert.Equal(t, compositeSpan.ParentID, compositeParent.ID)
	assert.GreaterOrEqual(t, compositeParent.Duration, compositeSpan.Duration)
}

func TestCompressSpanSameKindParentSpanContext(t *testing.T) {
	// This test ensures that the compression also works when the s.Parent is
	// set (via the context.Context).
	// |________________transaction (6df3948c6eff7b57) - 15ms_________________|
	// |_____________________internal parent - 14ms______________________|
	// 	   |_3 db - 3ms__|
	// 							|_internal algorithm - 6ms__|
	// 								|2 Calls to client |
	// 											   |inte|
	tracer := apmtest.NewRecordingTracer()
	tracer.SetSpanCompressionEnabled(true)
	tracer.SetExitSpanMinDuration(0)
	tracer.SetSpanCompressionSameKindMaxDuration(5 * time.Millisecond)

	txStart := time.Now()
	tx := tracer.StartTransactionOptions("name", "type",
		apm.TransactionOptions{Start: txStart},
	)

	ctx := apm.ContextWithTransaction(context.Background(), tx)
	parentStart := txStart.Add(time.Millisecond)
	parent, ctx := apm.StartSpanOptions(ctx, "parent", "internal", apm.SpanOptions{
		Start: parentStart,
	})

	// These spans are all compressed into a composite.
	childrenStart := parentStart.Add(2 * time.Millisecond)
	for i := 0; i < 3; i++ {
		span, _ := apm.StartSpanOptions(ctx, "db", "redis", apm.SpanOptions{
			ExitSpan: true,
			Start:    childrenStart,
		})
		childrenStart = childrenStart.Add(time.Millisecond)
		span.Duration = time.Millisecond
		span.End()
	}

	// We create a nother "internal" type span from which 3 children (below)
	// are created. one of them
	testSpans := []struct {
		name     string
		typ      string
		duration time.Duration
	}{
		{name: "GET /some", typ: "client", duration: time.Millisecond},
		{name: "GET /resource", typ: "client", duration: 2 * time.Millisecond},
		{name: "compute something", typ: "internal", duration: time.Millisecond},
	}

	subParent, ctx := apm.StartSpanOptions(ctx, "algorithm", "internal", apm.SpanOptions{
		Start: childrenStart.Add(time.Millisecond),
	})
	childrenStart = childrenStart.Add(time.Millisecond)
	for _, childCfg := range testSpans {
		child, _ := apm.StartSpanOptions(ctx, childCfg.name, childCfg.typ, apm.SpanOptions{
			ExitSpan: true,
			Start:    childrenStart.Add(childCfg.duration),
		})
		childrenStart = childrenStart.Add(childCfg.duration)
		child.Duration = childCfg.duration
		child.End()
	}
	childrenStart = childrenStart.Add(time.Millisecond)
	subParent.Duration = 6 * time.Millisecond
	subParent.End()

	parent.Duration = childrenStart.Add(2 * time.Millisecond).Sub(txStart)
	parent.End()
	tx.Duration = 15 * time.Millisecond
	tx.End()

	tracer.Flush(nil)

	transaction := tracer.Payloads().Transactions[0]
	spans := tracer.Payloads().Spans

	defer func() {
		if t.Failed() {
			apmtest.WriteTraceTable(os.Stdout, transaction, spans)
			apmtest.WriteTraceWaterfall(os.Stdout, transaction, spans)
		}
	}()
	require.NotNil(t, transaction)
	assert.Equal(t, 5, len(spans))

	sort.SliceStable(spans, func(i, j int) bool {
		return time.Time(spans[i].Timestamp).Before(time.Time(spans[j].Timestamp))
	})

	redisSpan := spans[1]
	require.NotNil(t, redisSpan.Composite)
	assert.Equal(t, 3, redisSpan.Composite.Count)
	assert.Equal(t, float64(3), redisSpan.Composite.Sum)
	assert.Equal(t, "exact_match", redisSpan.Composite.CompressionStrategy)

	clientSpan := spans[3]
	require.NotNil(t, clientSpan.Composite)
	assert.Equal(t, clientSpan.ParentID, spans[2].ID)
	assert.Equal(t, 2, clientSpan.Composite.Count)
	assert.Equal(t, float64(3), clientSpan.Composite.Sum)
	assert.Equal(t, "same_kind", clientSpan.Composite.CompressionStrategy)
}

func TestCompressSpanSameKindConcurrent(t *testing.T) {
	// This test verifies there aren't any deadlocks on calling
	// span.End(), Parent.End() and tx.End().
	// Additionally, ensures that we're not leaking or losing any
	// spans on parents and transaction being ended early.
	tracer := apmtest.NewRecordingTracer()
	tracer.SetSpanCompressionEnabled(true)
	tracer.SetExitSpanMinDuration(0)

	tx := tracer.StartTransaction("name", "type")
	ctx := apm.ContextWithTransaction(context.Background(), tx)
	parent, ctx := apm.StartSpan(ctx, "parent", "internal")

	var wg sync.WaitGroup
	count := 100
	wg.Add(count)
	spanStarted := make(chan struct{})
	for i := 0; i < count; i++ {
		go func(i int) {
			child, _ := apm.StartSpanOptions(ctx, fmt.Sprint(i), "request", apm.SpanOptions{
				ExitSpan: true,
			})
			spanStarted <- struct{}{}
			child.End()
			wg.Done()
		}(i)
	}

	var received int
	for range spanStarted {
		received++
		if received >= 30 {
			tx.End()
		}
		if received >= 50 {
			parent.End()
		}
		if received == count {
			close(spanStarted)
		}
	}
	// Wait until all the spans have ended.
	wg.Wait()

	tracer.Flush(nil)
	payloads := tracer.Payloads()
	require.Len(t, payloads.Transactions, 1)
	defer func() {
		if t.Failed() {
			apmtest.WriteTraceTable(os.Stdout, payloads.Transactions[0], payloads.Spans)
			apmtest.WriteTraceWaterfall(os.Stdout, payloads.Transactions[0], payloads.Spans)
		}
	}()

	var spanCount int
	for _, span := range payloads.Spans {
		if span.Composite != nil {
			// The real span count is the composite count.
			spanCount += span.Composite.Count
			continue
		}
		// If it's a normal span, then increment by 1.
		spanCount++
	}

	// Asserts that the total spancount is 101, (100 generated spans + parent).
	assert.Equal(t, 101, spanCount)
}

func TestCompressSpanPrematureEnd(t *testing.T) {
	// This test cases assert that the cached spans are sent when the span or
	// tx that holds their cache is ended and the cache isn't lost.
	type expect struct {
		compressionStrategy string
		compositeSum        float64
		spanCount           int
		compositeCount      int
	}
	assertResult := func(t *testing.T, tx model.Transaction, spans []model.Span, expect expect) {
		defer func() {
			if t.Failed() {
				apmtest.WriteTraceTable(os.Stdout, tx, spans)
				apmtest.WriteTraceWaterfall(os.Stdout, tx, spans)
			}
		}()
		assert.Equal(t, expect.spanCount, len(spans))
		var composite *model.CompositeSpan
		for _, span := range spans {
			if span.Composite != nil {
				assert.Equal(t, expect.compositeCount, span.Composite.Count)
				assert.Equal(t, expect.compressionStrategy, span.Composite.CompressionStrategy)
				assert.Equal(t, expect.compositeSum, span.Composite.Sum)
				composite = span.Composite
			}
		}
		if expect.compositeCount > 0 {
			require.NotNil(t, composite)
		}
	}

	testCases := []struct {
		name                string
		exitSpanMinDuration time.Duration
		expect              expect
		droppedSpansStats   int
	}{
		{
			name: "NoDropExitSpans",
			expect: expect{
				spanCount:           3,
				compositeCount:      3,
				compressionStrategy: "same_kind",
				compositeSum:        1.5,
			},
		},
		{
			name:                "DropExitSpans",
			exitSpanMinDuration: time.Millisecond,
			droppedSpansStats:   1,
			expect: expect{
				spanCount:           2,
				compositeCount:      3,
				compressionStrategy: "same_kind",
				compositeSum:        1.5,
			},
		},
	}

	// 1. The parent ends before they do.
	// The parent holds the compression cache in this test case.
	// |      tx      |
	// |  parent |        <--- The parent ends before the children ends.
	// | child |          <--- compressed
	// |  child |         <--- compressed
	// |  child |         <--- compressed
	// |    child  |      <--- NOT compressed
	// The expected result are 3 spans, the cache is invalidated and the span
	// ended after the parent ends.
	//
	// When drop fast exit spans is enabled, with 1ms min duration, the expected
	// span count is 2 (parent and the a composite which duration exceeds 1ms).
	// |      tx      |
	// |  parent |        <--- The parent ends before the children ends.
	// | child |          <--- compressed
	// |  child |         <--- compressed
	// |  child |         <--- compressed
	// |    child  |      <--- discarded since its duration is < than min exit.
	for _, test := range testCases {
		t.Run("ParentContext/"+test.name, func(t *testing.T) {
			tracer := apmtest.NewRecordingTracer()
			defer tracer.Close()
			tracer.SetSpanCompressionEnabled(true)
			tracer.SetExitSpanMinDuration(test.exitSpanMinDuration)
			tracer.SetSpanCompressionSameKindMaxDuration(5 * time.Millisecond)

			txStart := time.Now()
			tx := tracer.StartTransaction("name", "type")
			ctx := apm.ContextWithTransaction(context.Background(), tx)
			currentTime := time.Now()
			parent, ctx := apm.StartSpanOptions(ctx, "parent", "internal", apm.SpanOptions{
				Start: currentTime,
			})
			for i := 0; i < 4; i++ {
				child, _ := apm.StartSpanOptions(ctx, fmt.Sprint(i), "type", apm.SpanOptions{
					Parent:   parent.TraceContext(),
					ExitSpan: true,
					Start:    currentTime,
				})
				child.Duration = 500 * time.Microsecond
				currentTime = currentTime.Add(time.Millisecond)
				child.End()
				if i == 2 {
					parent.Duration = 2 * time.Millisecond
					parent.End()
				}
			}
			tx.Duration = currentTime.Sub(txStart)
			tx.End()
			tracer.Flush(nil)

			assertResult(t,
				tracer.Payloads().Transactions[0], tracer.Payloads().Spans, test.expect,
			)

			assert.Len(t,
				tracer.Payloads().Transactions[0].DroppedSpansStats,
				test.droppedSpansStats,
			)
		})
	}

	// 2. The tx ends before the parent ends.
	// The tx holds the compression cache in this test case.
	// |    tx   |          <--- The TX ends before parent.
	// |   parent  |
	// | child |            <--- compressed
	// |  child |           <--- compressed
	// The expected result are 3 spans, the cache is invalidated and the span
	// ended after the parent ends.
	t.Run("TxEndBefore", func(t *testing.T) {
		tracer := apmtest.NewRecordingTracer()
		defer tracer.Close()
		tracer.SetSpanCompressionEnabled(true)
		tracer.SetExitSpanMinDuration(time.Nanosecond)
		tracer.SetSpanCompressionSameKindMaxDuration(5 * time.Millisecond)

		tx := tracer.StartTransaction("name", "type")
		ctx := apm.ContextWithTransaction(context.Background(), tx)

		parent, ctx := apm.StartSpan(ctx, "parent", "internal")
		for i := 0; i < 2; i++ {
			child, _ := apm.StartSpanOptions(ctx, fmt.Sprint(i), "type", apm.SpanOptions{
				ExitSpan: true,
			})
			child.Duration = time.Microsecond
			child.End()
		}
		tx.End()
		parent.End()
		tracer.Flush(nil)
		assertResult(t, tracer.Payloads().Transactions[0], tracer.Payloads().Spans, expect{
			spanCount:           2,
			compositeCount:      2,
			compressionStrategy: "same_kind",
			compositeSum:        0.002,
		})
	})

	// 2. The parent ends before the last of the children span are finished.
	// The tx holds the compression cache in this test case.
	// |      tx      |
	// |  parent  |         <--- The parent ends before the last child ends.
	// | child |           <--- compressed
	// |  child |          <--- compressed
	// |    child  |       <--- NOT compressed
	t.Run("ParentFromTx", func(t *testing.T) {
		tracer := apmtest.NewRecordingTracer()
		defer tracer.Close()
		tracer.SetSpanCompressionEnabled(true)
		tracer.SetExitSpanMinDuration(time.Nanosecond)
		tracer.SetSpanCompressionSameKindMaxDuration(5 * time.Millisecond)

		tx := tracer.StartTransaction("name", "type")
		parent := tx.StartSpan("parent", "internal", nil)
		for i := 0; i < 3; i++ {
			child := tx.StartSpanOptions(fmt.Sprint(i), "type", apm.SpanOptions{
				Parent:   parent.TraceContext(),
				ExitSpan: true,
			})
			child.Duration = time.Microsecond
			child.End()
			if i == 1 {
				parent.End()
			}
		}
		tx.End()
		tracer.Flush(nil)
		assertResult(t, tracer.Payloads().Transactions[0], tracer.Payloads().Spans, expect{
			spanCount:           3,
			compositeCount:      2,
			compressionStrategy: "same_kind",
			compositeSum:        0.002,
		})
	})
}

func TestExitSpanDoesNotOverwriteDestinationServiceResource(t *testing.T) {
	_, spans, _ := apmtest.WithTransaction(func(ctx context.Context) {
		span, _ := apm.StartSpanOptions(ctx, "name", "type", apm.SpanOptions{ExitSpan: true})
		assert.True(t, span.IsExitSpan())
		span.Context.SetDestinationService(apm.DestinationServiceSpanContext{
			Resource: "my-custom-resource",
		})
		span.Duration = 2 * time.Millisecond
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

func TestSpanFastExit(t *testing.T) {
	type expect struct {
		spans                  int
		droppedSpansStatsCount int
	}
	tests := []struct {
		expect expect
		setup  func() func()
		name   string
	}{
		{
			name: "DefaultSetting/KeepSpan",
			expect: expect{
				spans:                  1,
				droppedSpansStatsCount: 0,
			},
		},
		{
			name: "2msSetting/KeepSpan",
			setup: func() func() {
				os.Setenv("ELASTIC_APM_EXIT_SPAN_MIN_DURATION", "2ms")
				return func() { os.Unsetenv("ELASTIC_APM_EXIT_SPAN_MIN_DURATION") }
			},
			expect: expect{
				spans:                  1,
				droppedSpansStatsCount: 0,
			},
		},
		{
			name: "3msSetting/DropSpan",
			setup: func() func() {
				os.Setenv("ELASTIC_APM_EXIT_SPAN_MIN_DURATION", "3ms")
				return func() { os.Unsetenv("ELASTIC_APM_EXIT_SPAN_MIN_DURATION") }
			},
			expect: expect{
				spans:                  0,
				droppedSpansStatsCount: 1,
			},
		},
		{
			name: "100usSetting/DropSpan",
			setup: func() func() {
				os.Setenv("ELASTIC_APM_EXIT_SPAN_MIN_DURATION", "100us")
				return func() { os.Unsetenv("ELASTIC_APM_EXIT_SPAN_MIN_DURATION") }
			},
			expect: expect{
				spans:                  1,
				droppedSpansStatsCount: 0,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.setup != nil {
				defer test.setup()()
			}

			tracer := apmtest.NewRecordingTracer()
			defer tracer.Close()

			tx := tracer.StartTransaction("name", "type")
			span := tx.StartSpanOptions("name", "type", apm.SpanOptions{ExitSpan: true})
			span.Duration = 2 * time.Millisecond

			span.End()
			tx.End()
			tracer.Flush(nil)
			payloads := tracer.Payloads()
			require.Len(t, payloads.Transactions, 1)
			assert.Len(t, payloads.Spans, test.expect.spans)
			assert.Len(t,
				payloads.Transactions[0].DroppedSpansStats,
				test.expect.droppedSpansStatsCount,
			)
		})
	}
}

func TestSpanFastExitWithCompress(t *testing.T) {
	// This test case asserts compressing spans into a composite:
	//  * Takes precedence over dropping the spans
	//  * When spans cannot be compressed but are discardable, they are.
	//  * The compressed and dropped spans are not counted in tx.started.
	//  * Dropped spans increment the dropped count.
	// Since compressed spans rely on the first compressed child's timestamp
	// to calculate the span duration, we're using a running timestsamp for
	// the spans.

	tracer := apmtest.NewRecordingTracer()
	defer tracer.Close()
	tracer.SetSpanCompressionEnabled(true)

	txts := time.Now()
	tx := tracer.StartTransactionOptions("name", "type", apm.TransactionOptions{
		Start: txts,
	})
	ctx := apm.ContextWithTransaction(context.Background(), tx)
	ts := time.Now()

	// Compress 499 spans which are compressable and can be dropped, they will
	// be compressed since that takes precedence.
	for i := 0; i < 499; i++ {
		span, _ := apm.StartSpanOptions(ctx, "compressed", "type", apm.SpanOptions{
			ExitSpan: true, Start: ts,
		})
		span.Duration = time.Millisecond
		ts = ts.Add(span.Duration)
		span.End()
	}

	// This span is compressable and can be dropped too but won't be since its
	// outcome is "failure".
	errorSpan, _ := apm.StartSpanOptions(ctx, "compressed", "type", apm.SpanOptions{
		ExitSpan: true, Start: ts,
	})
	errorSpan.Duration = time.Millisecond
	ts = ts.Add(errorSpan.Duration)
	errorSpan.Outcome = "failure"
	errorSpan.End()

	// These spans will be compressed into a composite.
	for i := 0; i < 100; i++ {
		span, _ := apm.StartSpanOptions(ctx, "compressed", "anothertype", apm.SpanOptions{
			ExitSpan: true, Start: ts,
		})
		span.Duration = time.Millisecond
		ts = ts.Add(span.Duration)
		span.End()
	}

	// Uncompressable spans are dropped when they are considered fast exit spans
	// <= 1ms by default. They should not be accounted in the "Started" spans.
	for i := 0; i < 100; i++ {
		span, _ := apm.StartSpanOptions(ctx, fmt.Sprint(i), fmt.Sprint(i), apm.SpanOptions{
			ExitSpan: true, Start: ts,
		})
		span.Duration = 500 * time.Microsecond
		ts = ts.Add(span.Duration)
		span.End()
	}

	tx.Duration = ts.Sub(txts)
	tx.End()
	tracer.Flush(nil)
	payloads := tracer.Payloads()

	require.Len(t, payloads.Transactions, 1)
	defer func() {
		if t.Failed() {
			apmtest.WriteTraceTable(os.Stdout, payloads.Transactions[0], payloads.Spans)
			apmtest.WriteTraceWaterfall(os.Stdout, payloads.Transactions[0], payloads.Spans)
		}
	}()

	assert.Len(t, payloads.Spans, 3)
	transaction := payloads.Transactions[0]
	assert.Len(t, transaction.DroppedSpansStats, 100)
	assert.Equal(t, model.SpanCount{
		Dropped: 100,
		Started: 3,
	}, transaction.SpanCount)
}

func TestSpanFastExitNoTransaction(t *testing.T) {
	// This test case asserts that a discardable span is not discarded when the
	// transaction ends before the span, since the stats wouldn't be recorded.
	tracer := apmtest.NewRecordingTracer()
	defer tracer.Close()

	tx := tracer.StartTransaction("name", "type")
	ctx := apm.ContextWithTransaction(context.Background(), tx)
	span, _ := apm.StartSpanOptions(ctx, "compressed", "type", apm.SpanOptions{ExitSpan: true})

	tx.End()
	span.Duration = time.Millisecond
	span.End()

	tracer.Flush(nil)
	payloads := tracer.Payloads()

	require.Len(t, payloads.Transactions, 1)
	require.Len(t, payloads.Spans, 1)
	transaction := payloads.Transactions[0]

	assert.Len(t, transaction.DroppedSpansStats, 0)
	assert.Equal(t, model.SpanCount{
		Started: 1,
		Dropped: 0,
	}, transaction.SpanCount)
}

func TestSpanOutcome(t *testing.T) {
	_, spans, _ := apmtest.WithTransaction(func(ctx context.Context) {
		span1, _ := apm.StartSpan(ctx, "name", "type")
		span1.End()

		span2, _ := apm.StartSpan(ctx, "name", "type")
		span2.Outcome = "unknown"
		span2.End()

		span3, _ := apm.StartSpan(ctx, "name", "type")
		span3.Context.SetHTTPStatusCode(200)
		span3.End()

		span4, _ := apm.StartSpan(ctx, "name", "type")
		span4.Context.SetHTTPStatusCode(400)
		span4.End()

		span5, ctx := apm.StartSpan(ctx, "name", "type")
		apm.CaptureError(ctx, errors.New("an error")).Send()
		span5.End()
	})

	require.Len(t, spans, 5)
	assert.Equal(t, "success", spans[0].Outcome) // default
	assert.Equal(t, "unknown", spans[1].Outcome) // specified
	assert.Equal(t, "success", spans[2].Outcome) // HTTP status < 400
	assert.Equal(t, "failure", spans[3].Outcome) // HTTP status >= 400
	assert.Equal(t, "failure", spans[4].Outcome)
}
