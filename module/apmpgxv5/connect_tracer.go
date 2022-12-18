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

// ConnectTracer traces Connect and ConnectConfig
type ConnectTracer struct{}

var _ pgx.ConnectTracer = (*ConnectTracer)(nil)

const (
	connectSpanType = "db.postgresql.connect"
)

type contextKey string

const (
	dataContextKey contextKey = "data"
)

func (c ConnectTracer) TraceConnectStart(ctx context.Context, _ pgx.TraceConnectStartData) context.Context {
	statement := "connect"

	span, apmCtx := apm.StartSpanOptions(ctx, statement, connectSpanType, apm.SpanOptions{
		ExitSpan: false,
	})

	newCtx := context.WithValue(apmCtx, dataContextKey, values{
		start:     time.Now(),
		statement: statement,
	})

	return apm.ContextWithSpan(newCtx, span)
}

func (c ConnectTracer) TraceConnectEnd(ctx context.Context, data pgx.TraceConnectEndData) {
	span := apm.SpanFromContext(ctx)
	defer span.End()

	var (
		statement string
		db        string
		user      string
		host      string
		port      int
	)

	if d, ok := ctx.Value(dataContextKey).(values); ok {
		statement = d.statement
	}

	// TODO: refactor
	if data.Conn != nil {
		db = data.Conn.Config().Database
		user = data.Conn.Config().User
		host = data.Conn.Config().Host
		port = int(data.Conn.Config().Port)
	}

	if span.Dropped() {
		return
	}

	span.Context.SetDatabase(apm.DatabaseSpanContext{
		Instance:  db,
		Statement: statement,
		Type:      "sql",
		User:      user,
	})
	span.Context.SetDestinationAddress(host, port)
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
