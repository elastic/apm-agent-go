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

//go:build go1.9
// +build go1.9

package apmzap // import "go.elastic.co/apm/module/apmzap"

import (
	"context"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"go.elastic.co/apm"
)

const (
	// FieldKeyTraceID is the field key for the trace ID.
	FieldKeyTraceID = "trace.id"

	// FieldKeyTransactionID is the field key for the transaction ID.
	FieldKeyTransactionID = "transaction.id"

	// FieldKeySpanID is the field key for the span ID.
	FieldKeySpanID = "span.id"
)

// TraceContext returns zap.Fields containing the trace context
// of the transaction and span contained in ctx, if any.
func TraceContext(ctx context.Context) []zapcore.Field {
	tx := apm.TransactionFromContext(ctx)
	if tx == nil {
		return nil
	}
	traceContext := tx.TraceContext()
	fields := []zapcore.Field{
		zap.Stringer(FieldKeyTraceID, traceContext.Trace),
		zap.Stringer(FieldKeyTransactionID, traceContext.Span),
	}
	if span := apm.SpanFromContext(ctx); span != nil {
		fields = append(fields, zap.Stringer(FieldKeySpanID, span.TraceContext().Span))
	}
	return fields
}
