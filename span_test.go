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
	tests := []struct {
		setup              func(t *testing.T) func()
		assertFunc         func(t *testing.T, spans []model.Span)
		name               string
		compressionEnabled bool
	}{
		{
			name: "DefaultThreshold",
			assertFunc: func(t *testing.T, spans []model.Span) {
				require.Equal(t, 3, len(spans))
				mysqlSpan := spans[0]
				assert.Equal(t, "mysql", mysqlSpan.Context.Destination.Service.Resource)
				assert.Nil(t, mysqlSpan.Composite)

				requestSpan := spans[1]
				assert.Equal(t, "request", requestSpan.Context.Destination.Service.Resource)
				require.NotNil(t, requestSpan.Composite)
				assert.Equal(t, 5, requestSpan.Composite.Count)
				assert.Equal(t, "same_kind", requestSpan.Composite.CompressionStrategy)
				assert.Equal(t, "Calls to request", requestSpan.Name)
				// Check that the sum and span duration is at least the duration of the time set.
				assert.Equal(t, 0.0005, requestSpan.Composite.Sum, requestSpan.Composite.Sum)
				assert.Equal(t, 0.0005, requestSpan.Duration, requestSpan.Duration)
			},
		},
		{
			name: "10msThreshold",
			setup: func(*testing.T) func() {
				os.Setenv("ELASTIC_APM_SPAN_COMPRESSION_SAME_KIND_MAX_DURATION", "10ms")
				return func() { os.Unsetenv("ELASTIC_APM_SPAN_COMPRESSION_SAME_KIND_MAX_DURATION") }
			},
			assertFunc: func(t *testing.T, spans []model.Span) {
				require.Equal(t, 2, len(spans))

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
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.setup != nil {
				defer test.setup(t)()
			}

			tracer := apmtest.NewRecordingTracer()
			defer tracer.Close()
			tracer.SetSpanCompressionEnabled(true)

			// Compress 5 spans into 1 and add another span with a different type
			// |______________transaction (572da67c206e9996) - 6.0006ms_______________|
			// m
			// 5
			// |________________________request GET /f - 6ms_________________________|
			//
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

			// These spans should be compressed into 1.
			path := []string{"/a", "/b", "/c", "/d", "/e"}
			for i := 0; i < 5; i++ {
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
			defer func() {
				if t.Failed() {
					apmtest.WriteTraceWaterfall(os.Stdout, transaction, spans)
					apmtest.WriteTraceTable(os.Stdout, transaction, spans)
				}
			}()

			require.NotNil(t, transaction)
			if test.assertFunc != nil {
				test.assertFunc(t, spans)
			}
		})
	}
}

func TestCompressSpanSameKindParentSpan(t *testing.T) {
	// This test asserts the span compression works when the spans are children
	// of another span.
	tracer := apmtest.NewRecordingTracer()
	tracer.SetSpanCompressionEnabled(true)

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

	currentTime := txStart
	{
		// Doesn't compress any spans since none meet the necessary conditions
		// the "request" type are both the same type but the parent
		parent := tx.StartSpanOptions("internal op", "internal", apm.SpanOptions{
			Start: currentTime,
		})
		// Have span propagate context downstream, this should not allow for
		// compression
		child := tx.StartSpanOptions("GET /resource", "request", apm.SpanOptions{
			Parent: parent.TraceContext(),
			Start:  currentTime.Add(100 * time.Microsecond),
		})

		grandChild := tx.StartSpanOptions("GET /different", "request", apm.SpanOptions{
			ExitSpan: true,
			Parent:   parent.TraceContext(),
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
		parent := tx.StartSpanOptions("another op", "internal", apm.SpanOptions{
			Start: currentTime.Add(50 * time.Microsecond),
		})
		child := tx.StartSpanOptions("GET /res", "request", apm.SpanOptions{
			ExitSpan: true,
			Parent:   parent.TraceContext(),
			Start:    currentTime.Add(120 * time.Microsecond),
		})

		otherChild := tx.StartSpanOptions("GET /diff", "request", apm.SpanOptions{
			ExitSpan: true,
			Parent:   parent.TraceContext(),
			Start:    currentTime.Add(150 * time.Microsecond),
		})

		child.Duration = 300 * time.Microsecond
		child.End()

		otherChild.Duration = 250 * time.Microsecond
		otherChild.End()

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
	// |________________transaction (ab51fc698fef307a) - 15ms_________________|
	//      |___________________internal parent - 13ms____________________|
	//             |3 Calls to re|
	//                                  |internal algorithm - 5m|
	//                                      |2 Calls t|
	//                                                |inte|
	tracer := apmtest.NewRecordingTracer()
	tracer.SetSpanCompressionEnabled(true)

	txStart := time.Now()
	tx := tracer.StartTransactionOptions("name", "type",
		apm.TransactionOptions{Start: txStart},
	)

	ctx := apm.ContextWithTransaction(context.Background(), tx)
	parentStart := txStart.Add(time.Millisecond)
	parent, ctx := apm.StartSpanOptions(ctx, "parent", "internal", apm.SpanOptions{
		Start: parentStart,
	})

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

	{
		span, _ := apm.StartSpanOptions(ctx, "algorithm", "internal", apm.SpanOptions{
			ExitSpan: true, Start: childrenStart.Add(time.Millisecond),
		})
		childrenStart = childrenStart.Add(time.Millisecond)
		{
			child, _ := apm.StartSpanOptions(ctx, "GET /some", "client", apm.SpanOptions{
				ExitSpan: true, Start: childrenStart.Add(time.Millisecond),
			})
			childrenStart = childrenStart.Add(time.Millisecond)
			child.Duration = time.Millisecond
			child.End()
		}
		{
			child, _ := apm.StartSpanOptions(ctx, "GET /resource", "client", apm.SpanOptions{
				ExitSpan: true, Start: childrenStart.Add(time.Millisecond),
			})
			childrenStart = childrenStart.Add(2 * time.Millisecond)
			child.Duration = 2 * time.Millisecond
			child.End()
		}
		{
			child, _ := apm.StartSpanOptions(ctx, "compute something", "internal", apm.SpanOptions{
				ExitSpan: false,
				Start:    childrenStart.Add(time.Millisecond),
			})
			childrenStart = childrenStart.Add(time.Millisecond)
			child.Duration = time.Millisecond
			child.End()
		}
		childrenStart = childrenStart.Add(time.Millisecond)
		span.Duration = 6 * time.Millisecond
		span.End()
	}

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
	assert.Equal(t, 2, clientSpan.Composite.Count)
	assert.Equal(t, float64(3), clientSpan.Composite.Sum)
	assert.Equal(t, "same_kind", clientSpan.Composite.CompressionStrategy)
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
