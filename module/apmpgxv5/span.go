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

	"go.elastic.co/apm/v2"
)

type spanType string

const (
	querySpanType   spanType = "db.postgresql.query"
	connectSpanType spanType = "db.postgresql.connect"
	copySpanType    spanType = "db.postgresql.copy"
	batchSpanType   spanType = "db.postgresql.batch"
)

const (
	databaseName           = "postgresql"
	databaseType           = "sql"
	destinationServiceType = "db"
)

var action = map[spanType]string{
	querySpanType:   "query",
	connectSpanType: "connect",
	copySpanType:    "copy",
	batchSpanType:   "batch",
}

func startSpan(ctx context.Context, name string, spanType spanType, conn *pgx.ConnConfig, opts apm.SpanOptions) (*apm.Span, context.Context, bool) {
	span, spanCtx := apm.StartSpanOptions(ctx, name, string(spanType), opts)

	if span.Dropped() {
		span.End()
		return nil, ctx, false
	}

	if conn != nil {
		span.Context.SetDatabase(apm.DatabaseSpanContext{
			Instance:  conn.Database,
			Statement: name,
			Type:      databaseType,
			User:      conn.User,
		})

		span.Context.SetDestinationAddress(conn.Host, int(conn.Port))
	}

	span.Action = action[spanType]
	span.Context.SetServiceTarget(apm.ServiceTargetSpanContext{
		Name: databaseName,
		Type: destinationServiceType,
	})
	span.Context.SetDestinationService(apm.DestinationServiceSpanContext{
		Name:     "",
		Resource: "postgresql",
	})

	return span, spanCtx, true
}

func endSpan(ctx context.Context, err error) {
	span := apm.SpanFromContext(ctx)
	if span == nil {
		return
	}

	defer span.End()

	if span.Dropped() {
		return
	}

	if err != nil {
		e := apm.CaptureError(ctx, err)
		e.SetSpan(span)
		e.Send()
	}

	return
}
