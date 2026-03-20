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

package apmpgx_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/tracelog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.elastic.co/apm/module/apmpgx/v2"
	"go.elastic.co/apm/v2/apmtest"
)

func TestInstrument(t *testing.T) {
	cfg, err := pgx.ParseConfig("postgres://postgres@localhost/test")
	require.NoError(t, err)

	originalTracer := cfg.Tracer
	apmpgx.Instrument(cfg)

	// Verify that the tracer was instrumented
	assert.NotNil(t, cfg.Tracer)
	assert.NotEqual(t, originalTracer, cfg.Tracer)
}

func TestTraceQueryStart(t *testing.T) {
	cfg, err := pgx.ParseConfig("postgres://testuser@localhost:5432/testdb")
	require.NoError(t, err)

	apmpgx.Instrument(cfg)

	testcases := []struct {
		name string
		sql  string
	}{
		{
			name: "simple SELECT",
			sql:  "SELECT * FROM foo.bar",
		},
		{
			name: "INSERT",
			sql:  "INSERT INTO foo (id) VALUES (1)",
		},
	}

	for _, test := range testcases {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, spans, errs := apmtest.WithTransaction(func(ctx context.Context) {
				data := pgx.TraceQueryStartData{
					SQL: test.sql,
				}
				ctx = cfg.Tracer.TraceQueryStart(ctx, nil, data)
				// Must call TraceQueryEnd to complete the span
				cfg.Tracer.TraceQueryEnd(ctx, nil, pgx.TraceQueryEndData{
					Err: nil,
				})
			})

			require.Len(t, spans, 1)
			assert.Equal(t, "db", spans[0].Type)
			assert.Equal(t, "postgresql", spans[0].Subtype)
			assert.Equal(t, "testdb", spans[0].Context.Database.Instance)
			assert.Equal(t, "testuser", spans[0].Context.Database.User)
			assert.Equal(t, test.sql, spans[0].Context.Database.Statement)
			assert.Len(t, errs, 0)
		})
	}
}

func TestTraceQueryEnd(t *testing.T) {
	cfg, err := pgx.ParseConfig("postgres://testuser@localhost:5432/testdb")
	require.NoError(t, err)

	apmpgx.Instrument(cfg)

	testcases := []struct {
		name          string
		sql           string
		withError     bool
		expectOutcome string
	}{
		{
			name:          "successful query",
			sql:           "SELECT * FROM foo",
			withError:     false,
			expectOutcome: "success",
		},
		{
			name:          "failed query",
			sql:           "SELECT * FROM foo",
			withError:     true,
			expectOutcome: "failure",
		},
	}

	for _, test := range testcases {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, spans, errs := apmtest.WithTransaction(func(ctx context.Context) {
				traceCtx := cfg.Tracer.TraceQueryStart(ctx, nil, pgx.TraceQueryStartData{
					SQL: test.sql,
				})

				queryErr := errors.New("query failed")
				if !test.withError {
					queryErr = nil
				}

				cfg.Tracer.TraceQueryEnd(traceCtx, nil, pgx.TraceQueryEndData{
					Err: queryErr,
				})
			})

			assert.Len(t, spans, 1)
			assert.Equal(t, test.expectOutcome, spans[0].Outcome)

			if test.withError {
				assert.Len(t, errs, 1)
			} else {
				assert.Len(t, errs, 0)
			}
		})
	}
}

func TestPublicLogAPI(t *testing.T) {
	cfg, err := pgx.ParseConfig("postgres://testuser@localhost:5432/testdb")
	require.NoError(t, err)

	apmpgx.Instrument(cfg)

	testcases := []struct {
		name       string
		msg        string
		data       map[string]interface{}
		expectName string
	}{
		{
			name: "Query message",
			msg:  "Query",
			data: map[string]interface{}{
				"sql": "SELECT * FROM users",
			},
			expectName: "SELECT FROM users",
		},
		{
			name: "Exec message",
			msg:  "Exec",
			data: map[string]interface{}{
				"sql": "INSERT INTO users (name) VALUES ($1)",
			},
			expectName: "INSERT INTO users",
		},
		{
			name: "CopyFrom message",
			msg:  "CopyFrom",
			data: map[string]interface{}{
				"tableName":   pgx.Identifier{"users"},
				"columnNames": pgx.Identifier{"id", "name"},
			},
			expectName: "COPY TO users",
		},
		{
			name: "SendBatch message",
			msg:  "SendBatch",
			data: map[string]interface{}{
				"batchLen": 5,
			},
			expectName: "BATCH",
		},
	}

	for _, test := range testcases {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			tracer := cfg.Tracer
			_, spans, _ := apmtest.WithTransaction(func(ctx context.Context) {
				// Cast to access public Log method
				if logTracer, ok := tracer.(interface {
					Log(context.Context, tracelog.LogLevel, string, map[string]interface{})
				}); ok {
					logTracer.Log(ctx, tracelog.LogLevelInfo, test.msg, test.data)
				}
			})

			require.Len(t, spans, 1)
			assert.Equal(t, test.expectName, spans[0].Name)
			assert.Equal(t, "db", spans[0].Type)
			assert.Equal(t, "postgresql", spans[0].Subtype)
			assert.Equal(t, "success", spans[0].Outcome)
		})
	}
}
