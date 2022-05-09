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
	querySpanType = "db.postgresql.query"
	copySpanType  = "db.postgresql.copy"
	batchSpanType = "db.postgresql.batch"

	postgresql = "postgresql"
)

var ErrUnsupportedPgxVersion = errors.New("this version of pgx is unsupported for tracing, please upgrade")

type Tracer struct {
	logger pgx.Logger
}

func NewTracer(logger pgx.Logger) *Tracer {
	return &Tracer{logger: logger}
}

func (t *Tracer) Log(ctx context.Context, level pgx.LogLevel, msg string, data map[string]interface{}) {
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

func (t *Tracer) QueryTrace(ctx context.Context, data map[string]interface{}) {
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

func (t *Tracer) CopyTrace(ctx context.Context, data map[string]interface{}) {
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

func (t *Tracer) BatchTrace(ctx context.Context, data map[string]interface{}) {
	stop := time.Now()

	var batchLen int
	if _, ok := data["batchLen"]; ok {
		batchLen = data["batchLen"].(int)
	}

	span, _ := apm.StartSpanOptions(ctx, "BATCH", batchSpanType, apm.SpanOptions{
		Start: stop.Add(-data["time"].(time.Duration)),
	})

	span.Duration = data["time"].(time.Duration)
	span.Context.SetLabel("batch.length", batchLen)
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
