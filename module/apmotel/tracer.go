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

//go:build go1.18
// +build go1.18

package apmotel // import "go.elastic.co/apm/module/apmotel/v2"

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/trace"

	"go.elastic.co/apm/v2"
)

type tracer struct {
	provider *tracerProvider
}

func newTracer(p *tracerProvider) *tracer {
	return &tracer{p}
}

// Start forwards the call to APM Agent
func (t *tracer) Start(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	config := trace.NewSpanStartConfig(opts...)

	startTime := config.Timestamp()
	if startTime.IsZero() {
		startTime = time.Now()
	}

	if p := trace.SpanFromContext(ctx); !config.NewRoot() && p != nil {
		if apmSpan, ok := p.(*span); ok {
			// This is a child span. Create a span, not a transaction
			opts := apm.SpanOptions{
				Parent: apmSpan.span.TraceContext(),
				Start:  config.Timestamp(),
			}

			apmTx := apmSpan.tx
			apmSpan := apmTx.StartSpanOptions(spanName, "", opts)

			s := &span{
				provider: t.provider,

				attributes: config.Attributes(),
				startTime:  startTime,

				tx:   apmTx,
				span: apmSpan,
			}
			return trace.ContextWithSpan(ctx, s), s
		}
	}

	apmTx := t.provider.apmTracer.StartTransactionOptions(spanName, "", apm.TransactionOptions{})
	s := &span{
		provider: t.provider,

		attributes: config.Attributes(),
		startTime:  startTime,

		tx: apmTx,
	}
	return trace.ContextWithSpan(ctx, s), s
}
