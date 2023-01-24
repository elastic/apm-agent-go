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

func Test_QueryTrace(t *testing.T) {
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
		name      string
		expectErr bool
		query     string
	}{
		{
			name:      "QUERY span, success",
			expectErr: false,
			query:     "SELECT * FROM foo",
		},
		{
			name:      "QUERY span, error",
			expectErr: true,
			query:     "SELECT * FROM foo2",
		},
	}

	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			_, spans, errs := apmtest.WithUncompressedTransaction(func(ctx context.Context) {
				rows, _ := conn.Query(ctx, tt.query)
				defer rows.Close()
			})

			assert.NotNil(t, spans[0].ID)

			if tt.expectErr {
				require.Len(t, errs, 1)
				assert.Equal(t, "failure", spans[0].Outcome)
			} else {
				assert.Equal(t, "success", spans[0].Outcome)

				assert.Equal(t, "SELECT * FROM foo", spans[0].Name)
				assert.Equal(t, "postgresql", spans[0].Subtype)
				assert.Equal(t, "success", spans[0].Outcome)

				assert.Equal(t, &model.SpanContext{
					Destination: &model.DestinationSpanContext{
						Address: cfg.Host,
						Port:    int(cfg.Port),
						Service: &model.DestinationServiceSpanContext{
							Type:     "db",
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
						Statement: "SELECT * FROM foo",
						Type:      "sql",
						User:      cfg.User,
					},
				}, spans[0].Context)
			}
		})
	}
}
