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
	"sync"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.elastic.co/apm"
	"go.elastic.co/apm/apmtest"
	"go.elastic.co/apm/model"
)

func TestContextStartSpanTransactionEnded(t *testing.T) {
	tracer := apmtest.DiscardTracer
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 1000; i++ {
				tx := tracer.StartTransaction("name", "type")
				ctx := apm.ContextWithTransaction(context.Background(), tx)
				tx.End()
				apm.CaptureError(ctx, errors.New("boom")).Send()
				span, _ := apm.StartSpan(ctx, "name", "type")
				assert.False(t, span.Dropped())
				span.End()
			}
		}()
	}
	tracer.Flush(nil)
	wg.Wait()
}

func TestContextStartSpanSpanEnded(t *testing.T) {
	tracer := apmtest.DiscardTracer
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 1000; i++ {
				tx := tracer.StartTransaction("name", "type")
				ctx := apm.ContextWithTransaction(context.Background(), tx)
				span1, ctx := apm.StartSpan(ctx, "name", "type")
				span1.End()
				apm.CaptureError(ctx, errors.New("boom")).Send()
				span2, _ := apm.StartSpan(ctx, "name", "type")
				assert.False(t, span2.Dropped())
				span2.End()
				tx.End()
			}
		}()
	}
	tracer.Flush(nil)
	wg.Wait()
}

func TestContextStartSpanOptions(t *testing.T) {
	txTimestamp := time.Now().Add(-time.Hour)
	tx, spans, _ := apmtest.WithTransactionOptions(apm.TransactionOptions{
		Start: txTimestamp,
	}, func(ctx context.Context) {
		span0, ctx := apm.StartSpanOptions(ctx, "span0", "type", apm.SpanOptions{
			Start: txTimestamp.Add(time.Minute),
		})
		assert.False(t, span0.Dropped())
		defer span0.End()

		// span1 should use span0 as its parent, as span0 has not been ended yet.
		span1, ctx := apm.StartSpanOptions(ctx, "span1", "type", apm.SpanOptions{})
		assert.False(t, span1.Dropped())
		span1TraceContext := span1.TraceContext()
		span1.End()

		// span2 should not be dropped. The parent span in ctx should be
		// completely disregarded, since Parent is specified in options.
		span2, ctx := apm.StartSpanOptions(ctx, "span2", "type", apm.SpanOptions{
			Parent: span1TraceContext,
		})
		assert.False(t, span2.Dropped())
		span2.End()

		// span3 should not be dropped. Even though the parent in ctx has been ended,
		// we still record the span to allow for fire-and-forget type patterns.
		span3, ctx := apm.StartSpanOptions(ctx, "span3", "type", apm.SpanOptions{})
		assert.False(t, span3.Dropped())
		span3.End()
	})

	require.Len(t, spans, 4)
	assert.Equal(t, "span0", spans[3].Name) // span 0 ended last
	assert.Equal(t, "span1", spans[0].Name)
	assert.Equal(t, "span2", spans[1].Name)
	assert.Equal(t, "span3", spans[2].Name)

	assert.Equal(t, tx.ID, spans[3].ParentID)
	assert.Equal(t, spans[3].ID, spans[0].ParentID)
	assert.Equal(t, spans[0].ID, spans[1].ParentID)
	assert.Equal(t, spans[1].ID, spans[2].ParentID)

	span0Start := time.Time(tx.Timestamp).Add(time.Minute)
	assert.Equal(t, model.Time(span0Start), spans[3].Timestamp)
}

func TestDetachedContext(t *testing.T) {
	funcB := func(ctx context.Context) chan chan error {
		chch := make(chan chan error)
		go func() {
			ch := <-chch
			defer close(ch)
			span, ctx := apm.StartSpan(ctx, "funcB", "custom")
			defer span.End()
			ch <- ctx.Err()
		}()
		return chch
	}

	funcA := func(ctx context.Context) chan chan error {
		span, ctx := apm.StartSpan(ctx, "funcA", "custom")
		defer span.End()
		return funcB(apm.DetachedContext(ctx))
	}

	tx, spans, _ := apmtest.WithTransaction(func(ctx context.Context) {
		// Call funcA and immediately cancel its context after it returns.
		ctx, cancel := context.WithCancel(ctx)
		chch := funcA(ctx)
		cancel()

		// Sending a channel to the goroutine spawned by funcB will cause
		// it to start a new span from the context passed by funcA. The
		// "funcA" span in the context is ended, and the context of funcA
		// is canceled; but the context of funcB is not canceled because
		// it was passed a "detached context".
		ch := make(chan error)
		chch <- ch
		err := <-ch
		assert.NoError(t, err)

		// wait for ch to be closed, at which point we know that funcB's
		// span has ended.
		<-ch
	})
	require.Len(t, spans, 2)

	assert.Equal(t, tx.ID, spans[0].ParentID)
	assert.Equal(t, spans[0].ID, spans[1].ParentID)
	for _, span := range spans {
		assert.Equal(t, tx.ID, span.TransactionID)
		assert.Equal(t, tx.TraceID, span.TraceID)
	}
}
