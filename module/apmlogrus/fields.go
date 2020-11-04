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

package apmlogrus // import "go.elastic.co/apm/module/apmlogrus"

import (
	"context"

	"github.com/sirupsen/logrus"

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

// TraceContext returns a logrus.Fields containing the trace
// context of the transaction and span contained in ctx, if any.
func TraceContext(ctx context.Context) logrus.Fields {
	tx := apm.TransactionFromContext(ctx)
	if tx == nil {
		return nil
	}
	traceContext := tx.TraceContext()
	fields := logrus.Fields{
		FieldKeyTraceID:       traceContext.Trace,
		FieldKeyTransactionID: traceContext.Span,
	}
	if span := apm.SpanFromContext(ctx); span != nil {
		fields[FieldKeySpanID] = span.TraceContext().Span
	}
	return fields
}
