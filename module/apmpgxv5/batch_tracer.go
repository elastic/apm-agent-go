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

package apmpgxv5 // import "go.elastic.co/apm/module/apmpgxv5/v2"

import (
	"context"

	"github.com/jackc/pgx/v5"

	"go.elastic.co/apm/module/apmsql/v2"
	"go.elastic.co/apm/v2"
)

// BatchTracer traces SendBatch
type BatchTracer struct{}

var _ pgx.BatchTracer = (*BatchTracer)(nil)

func (b BatchTracer) TraceBatchStart(ctx context.Context, conn *pgx.Conn, _ pgx.TraceBatchStartData) context.Context {
	span, spanCtx, ok := startSpan(ctx, apmsql.QuerySignature("BATCH"), batchSpanType, conn.Config(),
		apm.SpanOptions{})
	if !ok {
		return ctx
	}

	return apm.ContextWithSpan(spanCtx, span)
}

func (b BatchTracer) TraceBatchQuery(ctx context.Context, conn *pgx.Conn, data pgx.TraceBatchQueryData) {
	span, _, ok := startSpan(ctx, apmsql.QuerySignature(data.SQL), querySpanType, conn.Config(), apm.SpanOptions{
		ExitSpan: true,
	})
	if !ok {
		return
	}

	defer span.End()
}

func (b BatchTracer) TraceBatchEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceBatchEndData) {
	endSpan(ctx, data.Err)
}
