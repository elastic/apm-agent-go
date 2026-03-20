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

package apmpgx // import "go.elastic.co/apm/module/apmpgx/v2"

import (
	"context"
	"errors"
	"fmt"

	"go.elastic.co/apm/module/apmsql/v2"
	"go.elastic.co/apm/v2"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/tracelog"
)

const (
	//querySpanType is setting action for query expression trace in APM server.
	querySpanType = "db.postgresql.query"

	//querySpanType is setting action for copy expression trace in APM server.
	copySpanType = "db.postgresql.copy"

	//querySpanType is setting action for batch expression trace in APM server.
	batchSpanType = "db.postgresql.batch"
)

var (
	// ErrUnsupportedPgxVersion is indicating that pgx version is unsupported
	ErrUnsupportedPgxVersion = errors.New("this version of pgx is unsupported")

	// ErrInvalidType is indicating that field type doesn't meet expected one
	ErrInvalidType = errors.New("invalid field type")
)

// tracer struct contains pgx.ConnConfig inside and pgx.QueryTracer implementation.
type tracer struct {
	// cfg is the pgx.ConnConfig which used for setting metadata in spans such as host, port, etc.
	cfg *pgx.ConnConfig

	// parentTracer is the original tracer if one was set. If it's nil, then only spans will be created.
	parentTracer pgx.QueryTracer

	// parentLogger is the original logger if one was set. If it's nil, then no additional logging is done.
	parentLogger tracelog.Logger
}

// Instrument wraps a pgx.ConnConfig with an APM tracer.
// It's safe to pass a config with no existing tracer, in which case only spans will be created.
// An optional parent logger can be provided to delegate log calls to (pass nil if not needed).
func Instrument(cfg *pgx.ConnConfig, parentLogger ...tracelog.Logger) {
	originalTracer := cfg.Tracer
	var logger tracelog.Logger
	if len(parentLogger) > 0 {
		logger = parentLogger[0]
	}
	cfg.Tracer = &tracer{
		cfg:          cfg,
		parentTracer: originalTracer,
		parentLogger: logger,
	}
}

// Log dispatches to the appropriate trace method based on msg type.
// This method maintains backwards compatibility with the pgx v4 Logger interface.
func (t *tracer) Log(ctx context.Context, level tracelog.LogLevel, msg string, data map[string]interface{}) {
	if t.parentLogger != nil {
		t.parentLogger.Log(ctx, level, msg, data)
	}

	switch msg {
	case "Query", "Exec":
		t.QueryTrace(ctx, data)
	case "CopyFrom":
		t.CopyTrace(ctx, data)
	case "SendBatch":
		t.BatchTrace(ctx, data)
	}
}

// QueryTrace traces query and creates spans for them.
func (t *tracer) QueryTrace(ctx context.Context, data map[string]interface{}) {
	statement, ok := data["sql"].(string)
	if !ok {
		apm.CaptureError(ctx,
			fmt.Errorf("%w: expect string, got: %T",
				ErrInvalidType,
				data["sql"],
			),
		).Send()
		return
	}

	t.startSpan(ctx, apmsql.QuerySignature(statement), querySpanType, statement, data)
}

// CopyTrace traces copy queries and creates spans for them.
func (t *tracer) CopyTrace(ctx context.Context, data map[string]interface{}) {
	tableName, ok := data["tableName"].(pgx.Identifier)
	if !ok {
		return
	}

	var columnNames []string

	switch data["columnNames"].(type) {
	case pgx.Identifier:
		columnNames = data["columnNames"].(pgx.Identifier)
	case []string:
		columnNames = data["columnNames"].([]string)
	default:
		return
	}

	statement := "COPY TO " + tableName[0] + "(" + columnNames[0] + ")"
	spanName := "COPY TO " + tableName[0]

	t.startSpan(ctx, spanName, copySpanType, statement, data)
}

// BatchTrace traces batch execution and creates spans for the whole batch.
func (t *tracer) BatchTrace(ctx context.Context, data map[string]interface{}) {
	t.startSpan(ctx, "BATCH", batchSpanType, "", data)
}

// startSpan is a helper that creates and manages spans.
func (t *tracer) startSpan(ctx context.Context, spanName, spanType, statement string, data map[string]interface{}) {
	span, _ := apm.StartSpanOptions(ctx, spanName, spanType, apm.SpanOptions{
		ExitSpan: true,
	})

	if span.Dropped() {
		span.End()
		return
	}

	span.Context.SetDatabase(apm.DatabaseSpanContext{
		Instance:  t.cfg.Database,
		Statement: statement,
		Type:      "sql",
		User:      t.cfg.User,
	})

	span.Context.SetDestinationAddress(t.cfg.Host, int(t.cfg.Port))
	span.Context.SetServiceTarget(apm.ServiceTargetSpanContext{
		Type: "postgresql",
		Name: t.cfg.Database,
	})
	span.Context.SetDestinationService(apm.DestinationServiceSpanContext{
		Name:     "postgresql",
		Resource: "postgresql",
	})

	if batchLen, ok := data["batchLen"].(int); ok {
		span.Context.SetLabel("batch.length", batchLen)
	}

	if err, ok := data["err"].(error); ok {
		e := apm.CaptureError(ctx, err)
		e.SetSpan(span)
		e.Send()
		span.Outcome = "failure"
	} else {
		span.Outcome = "success"
	}

	span.End()
}

// TraceQueryStart is called at the beginning of Query, QueryRow, and Exec calls.
func (t *tracer) TraceQueryStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	if t.parentTracer != nil {
		ctx = t.parentTracer.TraceQueryStart(ctx, conn, data)
	}

	statement := data.SQL
	spanName := apmsql.QuerySignature(statement)

	span, _ := apm.StartSpanOptions(ctx, spanName, querySpanType, apm.SpanOptions{
		ExitSpan: true,
	})

	if span.Dropped() {
		span.End()
		return ctx
	}

	span.Context.SetDatabase(apm.DatabaseSpanContext{
		Instance:  t.cfg.Database,
		Statement: statement,
		Type:      "sql",
		User:      t.cfg.User,
	})

	span.Context.SetDestinationAddress(t.cfg.Host, int(t.cfg.Port))
	span.Context.SetServiceTarget(apm.ServiceTargetSpanContext{
		Type: "postgresql",
		Name: t.cfg.Database,
	})
	span.Context.SetDestinationService(apm.DestinationServiceSpanContext{
		Name:     "postgresql",
		Resource: "postgresql",
	})

	return context.WithValue(ctx, spanKey, span)
}

// TraceQueryEnd is called at the end of Query, QueryRow, and Exec calls.
func (t *tracer) TraceQueryEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryEndData) {
	span := t.extractSpan(ctx)
	if span != nil {
		if data.Err != nil {
			e := apm.CaptureError(ctx, data.Err)
			e.SetSpan(span)
			e.Send()
			span.Outcome = "failure"
		} else {
			span.Outcome = "success"
		}
		span.End()
	}

	if t.parentTracer != nil {
		t.parentTracer.TraceQueryEnd(ctx, conn, data)
	}
}

func (t *tracer) extractSpan(ctx context.Context) *apm.Span {
	span, ok := ctx.Value(spanKey).(*apm.Span)
	if !ok {
		return nil
	}
	return span
}

// Context key for storing the span
type contextKey string

const spanKey contextKey = "apmSpan"
