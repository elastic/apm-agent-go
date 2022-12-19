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
	"fmt"
	"github.com/jackc/pgx/v5"
	"go.elastic.co/apm/v2"
	"strings"
	"time"
)

// CopyFromTracer traces CopyFrom
type CopyFromTracer struct{}

var _ pgx.CopyFromTracer = (*CopyFromTracer)(nil)

func (c CopyFromTracer) TraceCopyFromStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceCopyFromStartData) context.Context {
	span, spanCtx, ok := startSpan(ctx, fmt.Sprintf(
		"COPY TO %s(%s)",
		strings.Join(data.TableName, ", "),
		strings.Join(data.ColumnNames, ", "),
	), copySpanType, conn.Config(), apm.SpanOptions{
		Start:    time.Now(),
		ExitSpan: false,
	})
	if !ok {
		return nil
	}

	return apm.ContextWithSpan(spanCtx, span)
}

func (c CopyFromTracer) TraceCopyFromEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceCopyFromEndData) {
	endSpan(ctx, data)
	return
}
