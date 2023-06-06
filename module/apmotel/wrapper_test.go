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

package apmotel

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/trace"

	"go.elastic.co/apm/v2"
	"go.elastic.co/apm/v2/transport/transporttest"
)

func TestLinkAgentToOtel(t *testing.T) {
	apmTracer, _ := transporttest.NewRecorderTracer()
	_, err := NewTracerProvider(WithAPMTracer(apmTracer))
	assert.NoError(t, err)

	ctx := context.Background()
	tx := apmTracer.StartTransaction("test1", "test")
	ctx = apm.ContextWithTransaction(ctx, tx)

	apmTx := apm.TransactionFromContext(ctx)
	otelSpan := trace.SpanFromContext(ctx)

	assert.Equal(t, [16]byte(apmTx.TraceContext().Trace), [16]byte(otelSpan.SpanContext().TraceID()))
	assert.Equal(t, [8]byte(apmTx.TraceContext().Span), [8]byte(otelSpan.SpanContext().SpanID()))
}

func TestLinkOtelToAgent(t *testing.T) {
	apmTracer, _ := transporttest.NewRecorderTracer()
	tp, err := NewTracerProvider(WithAPMTracer(apmTracer))
	assert.NoError(t, err)

	ctx := context.Background()
	ctx, _ = tp.Tracer("").Start(ctx, "test")

	apmTx := apm.TransactionFromContext(ctx)
	otelSpan := trace.SpanFromContext(ctx)

	assert.Equal(t, [16]byte(apmTx.TraceContext().Trace), [16]byte(otelSpan.SpanContext().TraceID()))
	assert.Equal(t, [8]byte(apmTx.TraceContext().Span), [8]byte(otelSpan.SpanContext().SpanID()))
}
