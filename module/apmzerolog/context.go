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

package apmzerolog

import (
	"context"

	"github.com/rs/zerolog"

	"go.elastic.co/apm"
)

const (
	// SpanIDFieldName is the field name for the span ID.
	SpanIDFieldName = "span.id"

	// TraceIDFieldName is the field name for the trace ID.
	TraceIDFieldName = "trace.id"

	// TransactionIDFieldName is the field name for the transaction ID.
	TransactionIDFieldName = "transaction.id"
)

// TraceContextHook returns a zerolog.Hook that will add any trace context
// contained in ctx to log events.
func TraceContextHook(ctx context.Context) zerolog.Hook {
	return traceContextHook{ctx}
}

type traceContextHook struct {
	ctx context.Context
}

func (h traceContextHook) Run(e *zerolog.Event, level zerolog.Level, message string) {
	tx := apm.TransactionFromContext(h.ctx)
	if tx == nil {
		return
	}
	traceContext := tx.TraceContext()
	e.Hex(TraceIDFieldName, traceContext.Trace[:])
	e.Hex(TransactionIDFieldName, traceContext.Span[:])
	if span := apm.SpanFromContext(h.ctx); span != nil {
		spanTraceContext := span.TraceContext()
		e.Hex("span.id", spanTraceContext.Span[:])
	}
}
