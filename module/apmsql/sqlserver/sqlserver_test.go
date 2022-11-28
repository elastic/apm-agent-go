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

package apmsqlserver_test

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.elastic.co/apm/module/apmsql/v2"
	_ "go.elastic.co/apm/module/apmsql/v2/sqlserver"
	"go.elastic.co/apm/v2/apmtest"
	"go.elastic.co/apm/v2/model"
)

var sqlserverHost = os.Getenv("SQLSERVER_HOST")

func TestQueryContext(t *testing.T) {
	if sqlserverHost == "" {
		t.Skipf("SQLSERVER_HOST not specified")
	}

	db, err := apmsql.Open("sqlserver", "sqlserver://sa:hunter2@"+sqlserverHost+":1433?database=test_db")
	require.NoError(t, err)
	defer db.Close()

	_, err = db.Exec(
		`IF NOT EXISTS (SELECT * FROM sysobjects WHERE name='foo' and xtype='U')
			CREATE TABLE foo (bar INT)`)
	require.NoError(t, err)

	_, spans, _ := apmtest.WithTransaction(func(ctx context.Context) {
		rows, err := db.QueryContext(ctx, "SELECT * FROM foo")
		require.NoError(t, err)
		rows.Close()
	})
	require.Len(t, spans, 1)

	assert.NotNil(t, spans[0].ID)
	assert.Equal(t, "SELECT FROM foo", spans[0].Name)
	assert.Equal(t, "sqlserver", spans[0].Subtype)
	assert.Equal(t, &model.SpanContext{
		Destination: &model.DestinationSpanContext{
			Address: sqlserverHost,
			Port:    1433,
			Service: &model.DestinationServiceSpanContext{
				Type:     "db",
				Name:     "sqlserver",
				Resource: "sqlserver",
			},
		},
		Service: &model.ServiceSpanContext{
			Target: &model.ServiceTargetSpanContext{
				Type: "sqlserver",
				Name: "test_db",
			},
		},
		Database: &model.DatabaseSpanContext{
			Instance:  "test_db",
			Statement: "SELECT * FROM foo",
			Type:      "sql",
			User:      "sa",
		},
	}, spans[0].Context)
}
