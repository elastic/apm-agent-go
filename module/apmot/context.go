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

package apmot

import (
	"time"

	opentracing "github.com/opentracing/opentracing-go"

	"go.elastic.co/apm"
)

type spanContext struct {
	tracer *otTracer // for origin of tx/span

	tx           *apm.Transaction
	traceContext apm.TraceContext
	startTime    time.Time
}

// TraceContext returns the trace context for the transaction or span
// associated with this span context.
func (s *spanContext) TraceContext() apm.TraceContext {
	return s.traceContext
}

// Transaction returns the transaction associated with this span context.
func (s *spanContext) Transaction() *apm.Transaction {
	return s.tx
}

// ForeachBaggageItem is a no-op; we do not support baggage propagation.
func (*spanContext) ForeachBaggageItem(handler func(k, v string) bool) {}

func parentSpanContext(refs []opentracing.SpanReference) (*spanContext, bool) {
	for _, ref := range refs {
		if !isValidSpanRef(ref) {
			continue
		}
		if ctx, ok := ref.ReferencedContext.(*spanContext); ok {
			return ctx, ok
		}
		if apmSpanContext, ok := ref.ReferencedContext.(interface {
			Transaction() *apm.Transaction
			TraceContext() apm.TraceContext
		}); ok {
			// The span context is (probably) one of the
			// native Elastic APM span/transaction wrapper
			// types. Synthesize a spanContext so we can
			// automatically correlate the events created
			// through our native API and the OpenTracing API.
			spanContext := &spanContext{
				tx:           apmSpanContext.Transaction(),
				traceContext: apmSpanContext.TraceContext(),
			}
			return spanContext, true
		}
	}
	return nil, false
}

// NOTE(axw) we currently support only "child-of" span references, but we make
// it possible to override them in order to appease the OT test harness in one
// specific test case: TestStartSpanWithParent, which tests both child-of and
// follows-from.

var isValidSpanRef = isChildOfSpanRef

func isChildOfSpanRef(ref opentracing.SpanReference) bool {
	return ref.Type == opentracing.ChildOfRef
}
