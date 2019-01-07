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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.elastic.co/apm"
	"go.elastic.co/apm/model"
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
	startTransactionInvalidTraceContext(t, apm.TraceContext{
		Trace: apm.TraceID{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15},
		// Span is all zeroes, which is invalid.
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
	tracer.Flush(nil)
	payloads := transport.Payloads()
	require.Len(t, payloads.Transactions, 1)
	assert.Equal(t, model.SpanID(parentSpan), payloads.Transactions[0].ParentID)
}

type samplerFunc func(apm.TraceContext) bool

func (f samplerFunc) Sample(t apm.TraceContext) bool {
	return f(t)
}
