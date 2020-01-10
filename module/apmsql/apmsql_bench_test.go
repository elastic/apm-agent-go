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

package apmsql_test

import (
	"context"
	"database/sql"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"

	"go.elastic.co/apm"
	"go.elastic.co/apm/module/apmsql"
	_ "go.elastic.co/apm/module/apmsql/sqlite3"
	"go.elastic.co/apm/transport"
)

func BenchmarkStmtQueryContext(b *testing.B) {
	db, err := apmsql.Open("sqlite3", ":memory:")
	require.NoError(b, err)
	defer db.Close()

	_, err = db.Exec("CREATE TABLE foo (bar INT)")
	require.NoError(b, err)

	stmt, err := db.Prepare("SELECT * FROM foo")
	require.NoError(b, err)
	defer stmt.Close()

	b.Run("baseline", func(b *testing.B) {
		benchmarkQueries(b, context.Background(), stmt)
	})
	b.Run("instrumented", func(b *testing.B) {
		invalidServerURL, err := url.Parse("http://testing.invalid:8200")
		if err != nil {
			panic(err)
		}
		httpTransport, err := transport.NewHTTPTransport()
		require.NoError(b, err)
		httpTransport.SetServerURL(invalidServerURL)

		tracer, err := apm.NewTracerOptions(apm.TracerOptions{
			ServiceName:    "apmhttp_test",
			ServiceVersion: "0.1",
			Transport:      httpTransport,
		})
		require.NoError(b, err)
		defer tracer.Close()

		tracer.SetMaxSpans(b.N)
		tx := tracer.StartTransaction("name", "type")
		ctx := apm.ContextWithTransaction(context.Background(), tx)
		benchmarkQueries(b, ctx, stmt)
	})
}

func benchmarkQueries(b *testing.B, ctx context.Context, stmt *sql.Stmt) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rows, err := stmt.QueryContext(ctx)
		require.NoError(b, err)
		rows.Close()
	}
}

func BenchmarkStmtExecContext(b *testing.B) {
	db, err := apmsql.Open("sqlite3", ":memory:")
	require.NoError(b, err)
	defer db.Close()

	_, err = db.Exec("CREATE TABLE foo (bar INT)")
	require.NoError(b, err)

	stmt, err := db.Prepare("DELETE FROM foo")
	require.NoError(b, err)
	defer stmt.Close()

	b.Run("baseline", func(b *testing.B) {
		benchmarkExec(b, context.Background(), stmt)
	})
	b.Run("instrumented", func(b *testing.B) {
		invalidServerURL, err := url.Parse("http://testing.invalid:8200")
		if err != nil {
			panic(err)
		}
		httpTransport, err := transport.NewHTTPTransport()
		require.NoError(b, err)
		httpTransport.SetServerURL(invalidServerURL)

		tracer, err := apm.NewTracerOptions(apm.TracerOptions{
			ServiceName:    "apmhttp_test",
			ServiceVersion: "0.1",
			Transport:      httpTransport,
		})
		require.NoError(b, err)
		defer tracer.Close()

		tracer.SetMaxSpans(b.N)
		tx := tracer.StartTransaction("name", "type")
		ctx := apm.ContextWithTransaction(context.Background(), tx)
		benchmarkExec(b, ctx, stmt)
	})
}

func benchmarkExec(b *testing.B, ctx context.Context, stmt *sql.Stmt) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := stmt.ExecContext(ctx)
		require.NoError(b, err)
	}
}
