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
)

var (
	// ErrUnsupportedPgxVersion is indicating that data doesn't contain value for "time" key. This field appeared in pgx v4.17
	ErrUnsupportedPgxVersion = errors.New("this version of pgx is unsupported, please upgrade to v4.17+")

	// ErrInvalidType is indicating that field type doesn't meet expected one
	ErrInvalidType = errors.New("invalid field type")
)

// tracer struct contains pgx.ConnConfig inside and pgx.Logger implementation.
type tracer struct {
	// cfg is the pgx.ConnConfig which used for setting metadata in spans such as host, port, etc.
	cfg *pgx.ConnConfig

	// logger used for writing data to log. If it's nil, then data won't be written to log, and only spans will be created.
	logger pgx.Logger
}

// Instrument is getting pgx.ConnConfig and wrap logger into tracer.
// It's safe to pass nil logger into pgx.ConnConfig, if so, then only spans will be created
func Instrument(cfg *pgx.ConnConfig) {
	cfg.Logger = &tracer{
		cfg:    cfg,
		logger: cfg.Logger,
	}
}

// Log is getting type of SQL expression from msg and run suitable trace.
// If logger in tracer struct isn't nil, than log will be
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

	span, ok := t.startSpan(ctx, apmsql.QuerySignature(statement), querySpanType, data)
	if !ok {
		return
	}
	defer span.End()
}

// CopyTrace traces copy queries and creates spans for them.
func (t *tracer) CopyTrace(ctx context.Context, data map[string]interface{}) {
	tableName, ok := data["tableName"].(pgx.Identifier)
	if !ok {
		return
	}

	span, ok := t.startSpan(ctx,
		fmt.Sprintf("COPY TO %s", strings.Join(tableName, ", ")),
		copySpanType,
		data,
	)
	if !ok {
		return
	}
	defer span.End()
}

// BatchTrace traces batch execution and creates spans for the whole batch.
func (t *tracer) BatchTrace(ctx context.Context, data map[string]interface{}) {
	span, ok := t.startSpan(ctx, "BATCH", batchSpanType, data)
	if !ok {
		return
	}
	defer span.End()

	if batchLen, ok := data["batchLen"].(int); ok {
		span.Context.SetLabel("batch.length", batchLen)
	}
}

func (t *tracer) startSpan(ctx context.Context, spanName, spanType string, data map[string]interface{}) (*apm.Span, bool) {
	stop := time.Now()

	duration, ok := data["time"].(time.Duration)
	if !ok {
		apm.CaptureError(ctx, ErrUnsupportedPgxVersion).Send()
		return nil, false
	}

	span, _ := apm.StartSpanOptions(ctx, spanName, spanType, apm.SpanOptions{
		Start:    stop.Add(-duration),
		ExitSpan: true,
	})

	if span.Dropped() {
		span.End()
		return nil, false
	}

	span.Duration = duration
	span.Context.SetDatabase(apm.DatabaseSpanContext{
		Instance:  t.cfg.Database,
		Statement: spanName,
		Type:      "sql",
		User:      t.cfg.User,
	})

	span.Context.SetDestinationAddress(t.cfg.Host, int(t.cfg.Port))

	if err, ok := data["err"].(error); ok {
		e := apm.CaptureError(ctx, err)
		e.Timestamp = stop
		e.SetSpan(span)
		e.Send()
	}

	return span, true
}
