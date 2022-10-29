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

package apmpgxv5

import (
	"context"
	"github.com/jackc/pgx/v5"
	"go.elastic.co/apm/v2"
	"time"
)

// BatchTracer traces SendBatch
type BatchTracer struct{}

var _ pgx.BatchTracer = (*BatchTracer)(nil)

const (
	batchSpanType = "db.postgresql.batch"
)

func (b BatchTracer) TraceBatchStart(ctx context.Context, conn *pgx.Conn, _ pgx.TraceBatchStartData) context.Context {
	span, apmCtx := apm.StartSpanOptions(ctx, "BATCH", batchSpanType, apm.SpanOptions{
		Start:    time.Now(),
		ExitSpan: false,
	})

	if span.Dropped() {
		span.End()
		return nil // todo: should be discussed
	}
	span.Action = "batch"

	return apmCtx
}

func (b BatchTracer) TraceBatchQuery(ctx context.Context, conn *pgx.Conn, data pgx.TraceBatchQueryData) {
	span, apmCtx := apm.StartSpanOptions(ctx, data.SQL, querySpanType, apm.SpanOptions{
		Start:    time.Now(),
		ExitSpan: false,
	})
	defer span.End()

	span.Action = "batch"
	span.Context.SetDatabase(apm.DatabaseSpanContext{
		Instance:  conn.Config().Database,
		Statement: data.SQL,
		Type:      "sql",
		User:      conn.Config().User,
	})
	span.Context.SetDestinationAddress(conn.Config().Host, int(conn.Config().Port))

	if apmErr := apm.CaptureError(apmCtx, data.Err); apmErr != nil {
		apmErr.SetSpan(span)
		apmErr.Send()
		return
	}
}

func (b BatchTracer) TraceBatchEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceBatchEndData) {
	span := apm.SpanFromContext(ctx)
	defer span.End()

	if span.Dropped() {
		span.End()
		return
	}

	span.Context.SetDestinationAddress(conn.Config().Host, int(conn.Config().Port))
	span.Action = "batch"

	if apmErr := apm.CaptureError(ctx, data.Err); apmErr != nil {
		apmErr.SetSpan(span)
		apmErr.Send()
		return
	}
}
