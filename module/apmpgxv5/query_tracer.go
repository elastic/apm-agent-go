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

// QueryTracer traces Query, QueryRow, and Exec
type QueryTracer struct{}

var _ pgx.QueryTracer = (*QueryTracer)(nil)

const (
	querySpanType = "db.postgresql.queryQueue"
)

type values struct {
	start     time.Time
	statement string
}

func (q QueryTracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	span, apmCtx := apm.StartSpanOptions(ctx, data.SQL, querySpanType, apm.SpanOptions{
		ExitSpan: false,
	})

	newCtx := context.WithValue(apmCtx, "data", values{
		start:     time.Now(),
		statement: data.SQL,
	})

	return apm.ContextWithSpan(newCtx, span)
}

func (q QueryTracer) TraceQueryEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryEndData) {
	v, ok := ctx.Value("data").(values)
	if !ok {
		return
	}

	span := apm.SpanFromContext(ctx)
	defer span.End()

	span.Duration = time.Now().Sub(v.start)

	if span.Dropped() {
		return
	}

	// todo: https://go.dev/play/p/45WtOgYBc7p
	span.Context.SetDatabase(apm.DatabaseSpanContext{
		Instance:  conn.Config().Database,
		Statement: v.statement,
		Type:      "sql",
		User:      conn.Config().User,
	})
	span.Context.SetDestinationAddress(conn.Config().Host, int(conn.Config().Port))
	span.Context.SetServiceTarget(apm.ServiceTargetSpanContext{
		Name: "postgresql",
		Type: "db",
	})

	if apmErr := apm.CaptureError(ctx, data.Err); apmErr != nil {
		apmErr.SetSpan(span)
		apmErr.Send()
	}

	return
}
