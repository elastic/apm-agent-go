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

package apmpgxv5_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.elastic.co/apm/module/apmpgxv5/v2"
	"go.elastic.co/apm/v2"
	"go.elastic.co/apm/v2/apmtest"
)

func Test_Connect(t *testing.T) {
	host := os.Getenv("PGHOST")
	if host == "" {
		t.Skipf("PGHOST not specified")
	}

	testcases := []struct {
		name      string
		dsn       string
		expectErr bool
	}{
		{
			name:      "CONNECT span, success",
			expectErr: false,
			dsn:       "postgres://postgres:hunter2@%s:5432/test_db",
		},
		{
			name:      "CONNECT span, failure",
			expectErr: true,
			dsn:       "postgres://postgres:hunter2@%s:5432/non_existing_db",
		},
	}

	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := pgx.ParseConfig(fmt.Sprintf(tt.dsn, host))
			require.NoError(t, err)

			apmpgxv5.Instrument(cfg)

			_, spans, errs := apmtest.WithUncompressedTransaction(func(ctx context.Context) {
				_, _ = pgx.ConnectConfig(ctx, cfg)
			})

			assert.NotNil(t, spans[0].ID)

			if tt.expectErr {
				require.Len(t, errs, 1)
				assert.Equal(t, "failure", spans[0].Outcome)
				assert.Equal(t, "CONNECT", spans[0].Name)
			} else {
				assert.Equal(t, "success", spans[0].Outcome)
				assert.Equal(t, "CONNECT", spans[0].Name)
			}
		})
	}
}

func Test_Connect_DroppedSpanKeepsCallerSpanOpen(t *testing.T) {
	cfg, err := pgx.ParseConfig("postgres://postgres:hunter2@127.0.0.1:5432/test_db")
	require.NoError(t, err)

	tracer := apmtest.NewRecordingTracer()
	defer tracer.Close()
	tracer.SetMaxSpans(1)

	tx := tracer.StartTransaction("name", "type")
	callerSpan, ctx := apm.StartSpan(apm.ContextWithTransaction(context.Background(), tx), "caller", "custom")

	var connectTracer apmpgxv5.ConnectTracer
	connectCtx := connectTracer.TraceConnectStart(ctx, pgx.TraceConnectStartData{ConnConfig: cfg})
	connectTracer.TraceConnectEnd(connectCtx, pgx.TraceConnectEndData{})

	callerSpanDuration := 10 * time.Millisecond
	time.Sleep(callerSpanDuration)
	callerSpan.End()
	tx.End()
	tracer.Flush(nil)

	spans := tracer.Payloads().Spans
	require.Len(t, spans, 1)
	assert.Equal(t, "caller", spans[0].Name)
	assert.GreaterOrEqual(t, spans[0].Duration, float64(callerSpanDuration.Milliseconds()))
}
