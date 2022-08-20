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

package apmpgx

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.elastic.co/apm/module/apmsql/v2"
	"go.elastic.co/apm/v2"

	"github.com/jackc/pgx/v4"
)

const (
	//querySpanType is setting action for query expression trace in APM server.
	querySpanType = "db.postgresql.query"

	//querySpanType is setting action for copy expression trace in APM server.
	copySpanType = "db.postgresql.copy"

	//querySpanType is setting action for batch expression trace in APM server.
	batchSpanType = "db.postgresql.batch"

	//postgresql is subtype which indicates database type in trace.
	postgresql = "postgresql"
)

// ErrUnsupportedPgxVersion is indicating that data doesn't contain value for "time" key.
// This fields appeared in pgx v4.17
var ErrUnsupportedPgxVersion = errors.New("this version of pgx is unsupported, please upgrade to v4.17")

// tracer is an implementation of pgx.Logger.
type tracer struct {
	// logger is the pgx.Logger to use for writing data to log.
	// If logger is nil, then data won't be written to log, and only spans will be created.
	logger pgx.Logger
}

// NewTracer returns a new tracer which creates spans for pgx queries.
// It is safe to pass nil logger to constructor.
func NewTracer(logger pgx.Logger) *tracer {
	return &tracer{logger: logger}
}

// Log is getting type of SQL expression from msg and run suitable trace.
// If logger was provided in NewTracer constructor, than expression will be
// written to your logger that implements pgx.Logger interface.
func (t *tracer) Log(ctx context.Context, level pgx.LogLevel, msg string, data map[string]interface{}) {
	if t.logger != nil {
		t.logger.Log(ctx, level, msg, data)
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
	stop := time.Now()

	if _, ok := data["time"]; !ok {
		apm.CaptureError(ctx, ErrUnsupportedPgxVersion).Send()
		return
	}

	span, _ := apm.StartSpanOptions(ctx, apmsql.QuerySignature(data["sql"].(string)), querySpanType, apm.SpanOptions{
		Start: stop.Add(-data["time"].(time.Duration)),
	})

	span.Duration = data["time"].(time.Duration)
	span.Context.SetDatabase(apm.DatabaseSpanContext{
		Type:      postgresql,
		Statement: data["sql"].(string),
	})

	if _, ok := data["err"]; ok {
		e := apm.CaptureError(ctx, data["err"].(error))
		e.Timestamp = stop
		e.Send()
	}

	span.End()
}

// CopyTrace traces copy queries and creates spans for them.
func (t *tracer) CopyTrace(ctx context.Context, data map[string]interface{}) {
	stop := time.Now()

	if _, ok := data["time"]; !ok {
		apm.CaptureError(ctx, ErrUnsupportedPgxVersion).Send()
		return
	}

	span, _ := apm.StartSpanOptions(ctx, fmt.Sprintf("COPY TO %s", strings.Join(data["tableName"].(pgx.Identifier), ", ")),
		copySpanType, apm.SpanOptions{
			Start: stop.Add(-data["time"].(time.Duration)),
		})

	span.Duration = data["time"].(time.Duration)
	span.Context.SetDatabase(apm.DatabaseSpanContext{
		Type: postgresql,
	})

	if _, ok := data["err"]; ok {
		e := apm.CaptureError(ctx, data["err"].(error))
		e.Timestamp = stop
		e.Send()
	}

	span.End()
}

// BatchTrace traces batch execution and creates spans for the whole batch.
func (t *tracer) BatchTrace(ctx context.Context, data map[string]interface{}) {
	stop := time.Now()

	span, _ := apm.StartSpanOptions(ctx, "BATCH", batchSpanType, apm.SpanOptions{
		Start: stop.Add(-data["time"].(time.Duration)),
	})

	if _, ok := data["batchLen"]; ok {
		span.Context.SetLabel("batch.length", data["batchLen"].(int))
	}

	span.Duration = data["time"].(time.Duration)
	span.Context.SetDatabase(apm.DatabaseSpanContext{
		Type: postgresql,
	})

	if _, ok := data["err"]; ok {
		e := apm.CaptureError(ctx, data["err"].(error))
		e.Timestamp = stop
		e.Send()
	}

	span.End()
}
