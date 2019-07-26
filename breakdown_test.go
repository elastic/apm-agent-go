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

func TestBreakdownMetrics_NonSampled(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	// Non-sampled transactions and their dropped spans still attribute to breakdown metrics.
	tracer.SetSampler(apm.NewRatioSampler(0))

	t0 := time.Now()
	tx := tracer.StartTransactionOptions("test", "request", apm.TransactionOptions{Start: t0})
	span := tx.StartSpanOptions("whatever", "db.mysql", apm.SpanOptions{Start: t0.Add(10 * time.Millisecond)})
	span.Duration = 10 * time.Millisecond // t0 + 20ms
	span.End()
	tx.Duration = 30 * time.Millisecond
	tx.End()

	tracer.Flush(nil)
	tracer.SendMetrics(nil)

	assertBreakdownMetrics(t, []model.Metrics{
		transactionDurationMetrics("test", "request", 1, 30*time.Millisecond),
		spanSelfTimeMetrics("test", "request", "app", "", 1, 20*time.Millisecond),
		spanSelfTimeMetrics("test", "request", "db", "mysql", 1, 10*time.Millisecond),
	}, payloadsBreakdownMetrics(transport))
}

func TestBreakdownMetrics_SpanDropped(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	// Dropped spans still attribute to breakdown metrics.
	tracer.SetMaxSpans(1)

	t0 := time.Now()
	tx := tracer.StartTransactionOptions("test", "request", apm.TransactionOptions{Start: t0})
	var spans []*apm.Span
	for i := 0; i < 2; i++ {
		span := tx.StartSpanOptions("whatever", "db.mysql", apm.SpanOptions{Start: t0.Add(10 * time.Millisecond)})
		spans = append(spans, span)
	}
	for _, span := range spans {
		span.Duration = 10 * time.Millisecond // t0 + 20ms
		span.End()
	}
	tx.Duration = 30 * time.Millisecond
	tx.End()

	tracer.Flush(nil)
	tracer.SendMetrics(nil)

	assertBreakdownMetrics(t, []model.Metrics{
		transactionDurationMetrics("test", "request", 1, 30*time.Millisecond),
		spanSelfTimeMetrics("test", "request", "app", "", 1, 20*time.Millisecond),
		spanSelfTimeMetrics("test", "request", "db", "mysql", 2, 20*time.Millisecond),
	}, payloadsBreakdownMetrics(transport))
}

func TestBreakdownMetrics_MetricsLimit(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	var logger apmtest.RecordLogger
	tracer.SetLogger(&logger)

	for i := 0; i < 3000; i++ {
		tx := tracer.StartTransaction(fmt.Sprintf("%d", i), "request")
		tx.End()
	}
	tracer.Flush(nil)
	tracer.SendMetrics(nil)

	// Make sure there's just one warning logged.
	var warnings []apmtest.LogRecord
	for _, record := range logger.Records {
		if record.Level == "warning" {
			warnings = append(warnings, record)
		}
	}
	require.Len(t, warnings, 1)
	assert.Regexp(t, "The limit of 1000 breakdown (.|\n)*", warnings[0].Message)

	// There should be 1000 breakdown metrics keys buckets retained
	// in-memory. Transaction count and duration metrics piggy-back
	// on the self_time for the "app" bucket, so some of those buckets
	// may generate multiple metricsets on the wire.
	metrics := payloadsBreakdownMetrics(transport)
	assert.Len(t, metrics, 2000)
}

func TestBreakdownMetrics_TransactionDropped(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	// Send transactions until one gets dropped, so we can check that
	// the dropped transactions are included in the breakdown.
	var count int
	for tracer.Stats().TransactionsDropped == 0 {
		for i := 0; i < 1000; i++ {
			tx := tracer.StartTransaction("test", "request")
			tx.Duration = 10 * time.Millisecond
			tx.End()
			count++
		}
	}

	tracer.Flush(nil)
	tracer.SendMetrics(nil)

	assertBreakdownMetrics(t, []model.Metrics{
		transactionDurationMetrics("test", "request", count, time.Duration(count)*10*time.Millisecond),
		spanSelfTimeMetrics("test", "request", "app", "", count, time.Duration(count)*10*time.Millisecond),
	}, payloadsBreakdownMetrics(transport))
}

func TestBreakdownMetrics_Disabled(t *testing.T) {
	os.Setenv("ELASTIC_APM_BREAKDOWN_METRICS", "false")
	defer os.Unsetenv("ELASTIC_APM_BREAKDOWN_METRICS")

	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	t0 := time.Now()
	tx := tracer.StartTransactionOptions("test", "request", apm.TransactionOptions{Start: t0})
	span := tx.StartSpanOptions("whatever", "db.mysql", apm.SpanOptions{Start: t0.Add(10 * time.Millisecond)})
	span.Duration = 10 * time.Millisecond // t0 + 20ms
	span.End()
	tx.Duration = 30 * time.Millisecond
	tx.End()

	tracer.Flush(nil)
	tracer.SendMetrics(nil)

	expect := transactionDurationMetrics("test", "request", 1, 30*time.Millisecond)
	expect.Samples["transaction.breakdown.count"] = model.Metric{}
	assertBreakdownMetrics(t, []model.Metrics{expect}, payloadsBreakdownMetrics(transport))
}

//                                 total self type
//  ██████████████████████████████    30   30 transaction
//           10        20        30
func TestBreakdownMetrics_AcceptanceTest1(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	tx := tracer.StartTransaction("test", "request")
	tx.Duration = 30 * time.Millisecond
	tx.End()

	tracer.Flush(nil)
	tracer.SendMetrics(nil)

	assertBreakdownMetrics(t, []model.Metrics{
		transactionDurationMetrics("test", "request", 1, 30*time.Millisecond),
		spanSelfTimeMetrics("test", "request", "app", "", 1, 30*time.Millisecond),
	}, payloadsBreakdownMetrics(transport))
}

//                                 total self type
//  ██████████░░░░░░░░░░██████████    30   20 transaction
//  └─────────██████████              10   10 db.mysql
//           10        20        30
func TestBreakdownMetrics_AcceptanceTest2(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	t0 := time.Now()
	tx := tracer.StartTransactionOptions("test", "request", apm.TransactionOptions{Start: t0})
	span := tx.StartSpanOptions("whatever", "db.mysql", apm.SpanOptions{Start: t0.Add(10 * time.Millisecond)})
	span.Duration = 10 * time.Millisecond // t0 + 20ms
	span.End()
	tx.Duration = 30 * time.Millisecond // t0 + 30ms
	tx.End()

	tracer.Flush(nil)
	tracer.SendMetrics(nil)

	assertBreakdownMetrics(t, []model.Metrics{
		transactionDurationMetrics("test", "request", 1, 30*time.Millisecond),
		spanSelfTimeMetrics("test", "request", "app", "", 1, 20*time.Millisecond),
		spanSelfTimeMetrics("test", "request", "db", "mysql", 1, 10*time.Millisecond),
	}, payloadsBreakdownMetrics(transport))
}

//                                 total self type
//  ██████████░░░░░░░░░░██████████    30   20 transaction
//  └─────────██████████              10   10 app
//           10        20        30
func TestBreakdownMetrics_AcceptanceTest3(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	t0 := time.Now()
	tx := tracer.StartTransactionOptions("test", "request", apm.TransactionOptions{Start: t0})
	span := tx.StartSpanOptions("whatever", "app", apm.SpanOptions{
		Start: t0.Add(10 * time.Millisecond),
	})
	span.Duration = 10 * time.Millisecond // t0 + 20ms
	span.End()
	tx.Duration = 30 * time.Millisecond // t0 + 30ms
	tx.End()

	tracer.Flush(nil)
	tracer.SendMetrics(nil)

	assertBreakdownMetrics(t, []model.Metrics{
		transactionDurationMetrics("test", "request", 1, 30*time.Millisecond),
		spanSelfTimeMetrics("test", "request", "app", "", 2, 30*time.Millisecond),
	}, payloadsBreakdownMetrics(transport))
}

//                                 total self type
//  ██████████░░░░░░░░░░██████████    30   20 transaction
//  └─────────██████████              10   10 db.mysql
//  └─────────██████████              10   10 db.mysql
//           10        20        30
func TestBreakdownMetrics_AcceptanceTest4(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	t0 := time.Now()
	tx := tracer.StartTransactionOptions("test", "request", apm.TransactionOptions{Start: t0})
	span1 := tx.StartSpanOptions("whatever", "db.mysql", apm.SpanOptions{Start: t0.Add(10 * time.Millisecond)})
	span2 := tx.StartSpanOptions("whatever", "db.mysql", apm.SpanOptions{Start: t0.Add(10 * time.Millisecond)})
	span1.Duration = 10 * time.Millisecond // t0 + 20ms
	span2.Duration = 10 * time.Millisecond // t0 + 20ms
	span1.End()
	span2.End()
	tx.Duration = 30 * time.Millisecond // t0 + 30ms
	tx.End()

	tracer.Flush(nil)
	tracer.SendMetrics(nil)

	assertBreakdownMetrics(t, []model.Metrics{
		transactionDurationMetrics("test", "request", 1, 30*time.Millisecond),
		spanSelfTimeMetrics("test", "request", "app", "", 1, 20*time.Millisecond),
		spanSelfTimeMetrics("test", "request", "db", "mysql", 2, 20*time.Millisecond),
	}, payloadsBreakdownMetrics(transport))
}

//                                 total self type
//  ██████████░░░░░░░░░░░░░░░█████    30   15 transaction
//  └─────────██████████              10   10 db.mysql
//  └──────────────██████████         10   10 db.mysql
//           10        20        30
func TestBreakdownMetrics_AcceptanceTest5(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	t0 := time.Now()
	tx := tracer.StartTransactionOptions("test", "request", apm.TransactionOptions{Start: t0})
	span1 := tx.StartSpanOptions("whatever", "db.mysql", apm.SpanOptions{Start: t0.Add(10 * time.Millisecond)})
	span2 := tx.StartSpanOptions("whatever", "db.mysql", apm.SpanOptions{Start: t0.Add(15 * time.Millisecond)})
	span1.Duration = 10 * time.Millisecond // t0 + 20ms
	span2.Duration = 10 * time.Millisecond // t0 + 25ms
	span1.End()
	span2.End()
	tx.Duration = 30 * time.Millisecond // t0 + 30ms
	tx.End()

	tracer.Flush(nil)
	tracer.SendMetrics(nil)

	assertBreakdownMetrics(t, []model.Metrics{
		transactionDurationMetrics("test", "request", 1, 30*time.Millisecond),
		spanSelfTimeMetrics("test", "request", "app", "", 1, 15*time.Millisecond),
		spanSelfTimeMetrics("test", "request", "db", "mysql", 2, 20*time.Millisecond),
	}, payloadsBreakdownMetrics(transport))
}

//                                 total self type
//  █████░░░░░░░░░░░░░░░░░░░░█████    30   15 transaction
//  └────██████████                   10   10 db.mysql
//  └──────────────██████████         10   10 db.mysql
//           10        20        30
func TestBreakdownMetrics_AcceptanceTest6(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	t0 := time.Now()
	tx := tracer.StartTransactionOptions("test", "request", apm.TransactionOptions{Start: t0})
	span1 := tx.StartSpanOptions("whatever", "db.mysql", apm.SpanOptions{Start: t0.Add(5 * time.Millisecond)})
	span1.Duration = 10 * time.Millisecond // t0 + 15ms
	span1.End()
	span2 := tx.StartSpanOptions("whatever", "db.mysql", apm.SpanOptions{Start: t0.Add(15 * time.Millisecond)})
	span2.Duration = 10 * time.Millisecond // t0 + 25ms
	span2.End()
	tx.Duration = 30 * time.Millisecond // t0 + 30ms
	tx.End()

	tracer.Flush(nil)
	tracer.SendMetrics(nil)

	assertBreakdownMetrics(t, []model.Metrics{
		transactionDurationMetrics("test", "request", 1, 30*time.Millisecond),
		spanSelfTimeMetrics("test", "request", "app", "", 1, 10*time.Millisecond),
		spanSelfTimeMetrics("test", "request", "db", "mysql", 2, 20*time.Millisecond),
	}, payloadsBreakdownMetrics(transport))
}

//                                 total self type
//  ██████████░░░░░█████░░░░░█████    30   15 transaction
//  └─────────█████                    5    5 db.mysql
//  └───────────────────█████          5    5 db.mysql
//           10        20        30
func TestBreakdownMetrics_AcceptanceTest7(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	t0 := time.Now()
	tx := tracer.StartTransactionOptions("test", "request", apm.TransactionOptions{Start: t0})
	span1 := tx.StartSpanOptions("whatever", "db.mysql", apm.SpanOptions{Start: t0.Add(10 * time.Millisecond)})
	span1.Duration = 5 * time.Millisecond // t0 + 15ms
	span1.End()
	span2 := tx.StartSpanOptions("whatever", "db.mysql", apm.SpanOptions{Start: t0.Add(20 * time.Millisecond)})
	span2.Duration = 5 * time.Millisecond // t0 + 25ms
	span2.End()
	tx.Duration = 30 * time.Millisecond // t0 + 30ms
	tx.End()

	tracer.Flush(nil)
	tracer.SendMetrics(nil)

	assertBreakdownMetrics(t, []model.Metrics{
		transactionDurationMetrics("test", "request", 1, 30*time.Millisecond),
		spanSelfTimeMetrics("test", "request", "app", "", 1, 20*time.Millisecond),
		spanSelfTimeMetrics("test", "request", "db", "mysql", 2, 10*time.Millisecond),
	}, payloadsBreakdownMetrics(transport))
}

//                                 total self type
//  ██████████░░░░░░░░░░██████████    30   20 transaction
//  └─────────█████░░░░░              10    5 app
//            └────██████████         10   10 db.mysql
//           10        20        30
func TestBreakdownMetrics_AcceptanceTest8(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	t0 := time.Now()
	tx := tracer.StartTransactionOptions("test", "request", apm.TransactionOptions{Start: t0})
	ctx := apm.ContextWithTransaction(context.Background(), tx)

	span1, ctx := apm.StartSpanOptions(ctx, "whatever", "app", apm.SpanOptions{Start: t0.Add(10 * time.Millisecond)})
	span2, ctx := apm.StartSpanOptions(ctx, "whatever", "db.mysql", apm.SpanOptions{Start: t0.Add(15 * time.Millisecond)})
	span1.Duration = 10 * time.Millisecond // t0 + 20ms
	span2.Duration = 10 * time.Millisecond // t0 + 25ms
	span1.End()
	span2.End()
	tx.Duration = 30 * time.Millisecond // t0 + 30ms
	tx.End()

	tracer.Flush(nil)
	tracer.SendMetrics(nil)

	assertBreakdownMetrics(t, []model.Metrics{
		transactionDurationMetrics("test", "request", 1, 30*time.Millisecond),
		spanSelfTimeMetrics("test", "request", "app", "", 2, 25*time.Millisecond),
		spanSelfTimeMetrics("test", "request", "db", "mysql", 1, 10*time.Millisecond),
	}, payloadsBreakdownMetrics(transport))
}

//                                 total self type
//  ██████████░░░░░░░░░░              20   20 transaction
//  └─────────██████████░░░░░░░░░░    20   10 app
//            └─────────██████████    10   10 db.mysql
//           10        20        30
func TestBreakdownMetrics_AcceptanceTest9(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	t0 := time.Now()
	tx := tracer.StartTransactionOptions("test", "request", apm.TransactionOptions{Start: t0})
	ctx := apm.ContextWithTransaction(context.Background(), tx)

	span1, ctx := apm.StartSpanOptions(ctx, "whatever", "app", apm.SpanOptions{Start: t0.Add(10 * time.Millisecond)})
	tx.Duration = 20 * time.Millisecond // t0 + 20ms
	tx.End()
	span2, ctx := apm.StartSpanOptions(ctx, "whatever", "db.mysql", apm.SpanOptions{Start: t0.Add(20 * time.Millisecond)})
	span1.Duration = 20 * time.Millisecond // t0 + 30ms
	span2.Duration = 10 * time.Millisecond // t0 + 30ms
	span1.End()
	span2.End()

	tracer.Flush(nil)
	tracer.SendMetrics(nil)

	assertBreakdownMetrics(t, []model.Metrics{
		// The db.mysql span should not be included in breakdown,
		// as it started after the transaction ended. The explicit
		// "app" span should not be included in the self_time value,
		// it should only have been used for subtracting from the
		// transaction's duration.
		transactionDurationMetrics("test", "request", 1, 20*time.Millisecond),
		spanSelfTimeMetrics("test", "request", "app", "", 1, 10*time.Millisecond),
	}, payloadsBreakdownMetrics(transport))
}

//                                 total self type
//  ██████████░░░░░░░░░░              20   10 transaction
//  └─────────████████████████████    20   20 db.mysql
//           10        20        30
func TestBreakdownMetrics_AcceptanceTest10(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	t0 := time.Now()
	tx := tracer.StartTransactionOptions("test", "request", apm.TransactionOptions{Start: t0})
	span := tx.StartSpanOptions("whatever", "db.mysql", apm.SpanOptions{Start: t0.Add(10 * time.Millisecond)})
	tx.Duration = 20 * time.Millisecond // t0 + 20ms
	tx.End()
	span.Duration = 20 * time.Millisecond // t0 + 30ms
	span.End()

	tracer.Flush(nil)
	tracer.SendMetrics(nil)

	assertBreakdownMetrics(t, []model.Metrics{
		// The db.mysql span should not be included in breakdown,
		// as it ended after the transaction ended.
		transactionDurationMetrics("test", "request", 1, 20*time.Millisecond),
		spanSelfTimeMetrics("test", "request", "app", "", 1, 10*time.Millisecond),
	}, payloadsBreakdownMetrics(transport))
}

//                                 total self type
//  ██████████                        10   10 transaction
//  └───────────────────██████████    10   10 db.mysql
//           10        20        30
func TestBreakdownMetrics_AcceptanceTest11(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	t0 := time.Now()
	tx := tracer.StartTransactionOptions("test", "request", apm.TransactionOptions{Start: t0})
	tx.Duration = 10 * time.Millisecond // t0 + 10ms
	tx.End()
	span := tx.StartSpanOptions("whatever", "db.mysql", apm.SpanOptions{Start: t0.Add(20 * time.Millisecond)})
	span.Duration = 10 * time.Millisecond // t0 + 30ms
	span.End()

	tracer.Flush(nil)
	tracer.SendMetrics(nil)

	assertBreakdownMetrics(t, []model.Metrics{
		// The db.mysql span should not be included in breakdown,
		// as it started and ended after the transaction ended.
		transactionDurationMetrics("test", "request", 1, 10*time.Millisecond),
		spanSelfTimeMetrics("test", "request", "app", "", 1, 10*time.Millisecond),
	}, payloadsBreakdownMetrics(transport))
}

func transactionDurationMetrics(txName, txType string, count int, sum time.Duration) model.Metrics {
	return model.Metrics{
		Transaction: model.MetricsTransaction{
			Type: txType,
			Name: txName,
		},
		Samples: map[string]model.Metric{
			"transaction.breakdown.count": {Value: float64(count)},
			"transaction.duration.count":  {Value: float64(count)},
			"transaction.duration.sum.us": {Value: sum.Seconds() * 1000000},
		},
	}
}

func spanSelfTimeMetrics(txName, txType, spanType, spanSubtype string, count int, sum time.Duration) model.Metrics {
	return model.Metrics{
		Transaction: model.MetricsTransaction{
			Type: txType,
			Name: txName,
		},
		Span: model.MetricsSpan{
			Type:    spanType,
			Subtype: spanSubtype,
		},
		Samples: map[string]model.Metric{
			"span.self_time.count":  {Value: float64(count)},
			"span.self_time.sum.us": {Value: sum.Seconds() * 1000000},
		},
	}
}

func assertBreakdownMetrics(t *testing.T, expect []model.Metrics, metrics []model.Metrics) {
	for i := range metrics {
		metrics[i].Timestamp = model.Time{}
	}
	assert.ElementsMatch(t, expect, metrics)
}

func payloadsBreakdownMetrics(t *transporttest.RecorderTransport) []model.Metrics {
	p := t.Payloads()
	ms := make([]model.Metrics, 0, len(p.Metrics))
	for _, m := range p.Metrics {
		if m.Transaction.Type == "" {
			continue
		}
		ms = append(ms, m)
	}
	return ms
}
