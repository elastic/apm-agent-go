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

package apmpgxv5_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.elastic.co/apm/module/apmpgxv5/v2"
	"go.elastic.co/apm/v2/apmtest"
	"go.elastic.co/apm/v2/model"
)

type stmt struct {
	query  string
	action string
}

func TestBatchTrace(t *testing.T) {
	host := os.Getenv("PGHOST")
	if host == "" {
		t.Skipf("PGHOST not specified")
	}

	cfg, err := pgx.ParseConfig(fmt.Sprintf("postgres://postgres:hunter2@%s:5432/test_db", host))
	require.NoError(t, err)

	ctx := context.TODO()

	apmpgxv5.Instrument(cfg)

	conn, err := pgx.ConnectConfig(ctx, cfg)
	require.NoError(t, err)

	_, err = conn.Exec(ctx, "CREATE TABLE IF NOT EXISTS foo (bar INT)")
	require.NoError(t, err)

	testcases := []struct {
		name       string
		expectErr  bool
		queryQueue []string
		expStmt    map[string]stmt
	}{
		{
			name:      "BATCH spans, success",
			expectErr: false,
			queryQueue: []string{
				"SELECT * FROM foo WHERE bar = 1",
				"SELECT bar FROM foo WHERE bar = 1",
			},
			expStmt: map[string]stmt{
				"BATCH": {
					query:  "BATCH",
					action: "batch",
				},
				"SELECT * FROM foo WHERE bar = 1": {
					query:  "SELECT FROM foo",
					action: "query",
				},
				"SELECT bar FROM foo WHERE bar = 1": {
					query:  "SELECT FROM foo",
					action: "query",
				},
			},
		},
		{
			name:      "BATCH spans, error",
			expectErr: true,
			queryQueue: []string{
				"SELECT * FROM foo WHERE bar = 1",
				"SELECT bar FROM foo2",
			},
			expStmt: map[string]stmt{
				"BATCH": {
					query:  "BATCH",
					action: "batch",
				},
				"SELECT * FROM foo WHERE bar = 1": {
					query:  "SELECT FROM foo",
					action: "query",
				},
				"SELECT bar FROM foo2 WHERE bar = 1": {
					query:  "SELECT FROM foo",
					action: "query",
				},
			},
		},
	}

	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			_, spans, errs := apmtest.WithUncompressedTransaction(func(ctx context.Context) {
				batch := &pgx.Batch{}

				for _, query := range tt.queryQueue {
					batch.Queue(query)
				}

				br := conn.SendBatch(ctx, batch)
				defer func() {
					_ = br.Close()
				}()
			})

			if tt.expectErr {
				require.Len(t, errs, 2)
				assert.Equal(t, "failure", spans[0].Outcome)
			} else {
				for i := range tt.queryQueue {
					expectedStatement := tt.expStmt[tt.queryQueue[i]]

					assert.Equal(t, "success", spans[i].Outcome)
					assert.Equal(t, "db", spans[i].Type)
					assert.Equal(t, "postgresql", spans[i].Subtype)
					assert.Equal(t, expectedStatement.action, spans[i].Action)
					assert.Equal(t, expectedStatement.query, spans[i].Name)

					assert.Equal(t, &model.SpanContext{
						Destination: &model.DestinationSpanContext{
							Address: cfg.Host,
							Port:    int(cfg.Port),
							Service: &model.DestinationServiceSpanContext{
								Type:     "db",
								Name:     "",
								Resource: "postgresql",
							},
						},
						Service: &model.ServiceSpanContext{
							Target: &model.ServiceTargetSpanContext{
								Type: "db",
								Name: "postgresql",
							},
						},
						Database: &model.DatabaseSpanContext{
							Instance:  cfg.Database,
							Statement: expectedStatement.query,
							Type:      "sql",
							User:      cfg.User,
						},
					}, spans[i].Context)
				}
			}
		})
	}
}
