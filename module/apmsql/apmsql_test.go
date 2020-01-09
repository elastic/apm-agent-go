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
	"database/sql/driver"
	"testing"

	sqlite3 "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.elastic.co/apm/apmtest"
	"go.elastic.co/apm/model"
	"go.elastic.co/apm/module/apmsql"
	_ "go.elastic.co/apm/module/apmsql/sqlite3"
)

func init() {
	apmsql.Register("sqlite3_test", &sqlite3TestDriver{})
}

func TestPingContext(t *testing.T) {
	db, err := apmsql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	db.Ping() // connect
	_, spans, _ := apmtest.WithTransaction(func(ctx context.Context) {
		err := db.PingContext(ctx)
		assert.NoError(t, err)
	})
	require.Len(t, spans, 1)
	assert.Equal(t, "ping", spans[0].Name)
	assert.Equal(t, "db", spans[0].Type)
	assert.Equal(t, "sqlite3", spans[0].Subtype)
	assert.Equal(t, "ping", spans[0].Action)
}

func TestExecContext(t *testing.T) {
	db, err := apmsql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	db.Ping() // connect
	const N = 5
	_, spans, _ := apmtest.WithTransaction(func(ctx context.Context) {
		_, err := db.ExecContext(ctx, "CREATE TABLE foo (bar INT)")
		require.NoError(t, err)
		for i := 0; i < N; i++ {
			result, err := db.ExecContext(ctx, "INSERT INTO foo VALUES (?)", i)
			require.NoError(t, err)

			rowsAffected, err := result.RowsAffected()
			require.NoError(t, err)
			assert.Equal(t, int64(1), rowsAffected)
		}
		result, err := db.ExecContext(ctx, "DELETE FROM foo")
		require.NoError(t, err)
		rowsAffected, err := result.RowsAffected()
		require.NoError(t, err)
		assert.Equal(t, int64(N), rowsAffected)
	})
	require.Len(t, spans, 2+N)

	int64ptr := func(n int64) *int64 { return &n }

	assert.Equal(t, "CREATE", spans[0].Name)
	assert.Equal(t, "db", spans[0].Type)
	assert.Equal(t, "sqlite3", spans[0].Subtype)
	assert.Equal(t, "exec", spans[0].Action)
	assert.Equal(t, &model.SpanContext{
		Database: &model.DatabaseSpanContext{
			Instance: ":memory:",
			// Ideally RowsAffected would not be returned for DDL
			// statements, but this is on the driver; it should
			// be returning database/sql/driver.ResultNoRows for
			// DDL statements, in which case we'll omit this.
			RowsAffected: int64ptr(0),
			Statement:    "CREATE TABLE foo (bar INT)",
			Type:         "sql",
		},
	}, spans[0].Context)

	for i := 0; i < N; i++ {
		span := spans[i+1]
		assert.Equal(t, "INSERT INTO foo", span.Name)
		assert.Equal(t, &model.SpanContext{
			Database: &model.DatabaseSpanContext{
				Instance:     ":memory:",
				RowsAffected: int64ptr(1),
				Statement:    "INSERT INTO foo VALUES (?)",
				Type:         "sql",
			},
		}, span.Context)
	}

	assert.Equal(t, "DELETE FROM foo", spans[N+1].Name)
	assert.Equal(t, &model.SpanContext{
		Database: &model.DatabaseSpanContext{
			Instance:     ":memory:",
			RowsAffected: int64ptr(N),
			Statement:    "DELETE FROM foo",
			Type:         "sql",
		},
	}, spans[N+1].Context)
}

func TestQueryContext(t *testing.T) {
	db, err := apmsql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	_, err = db.Exec("CREATE TABLE foo (bar INT)")
	require.NoError(t, err)

	_, spans, errors := apmtest.WithTransaction(func(ctx context.Context) {
		rows, err := db.QueryContext(ctx, "SELECT * FROM foo")
		require.NoError(t, err)
		rows.Close()
	})
	require.Len(t, spans, 1)
	assert.Empty(t, errors)

	assert.NotNil(t, spans[0].ID)
	assert.Equal(t, "SELECT FROM foo", spans[0].Name)
	assert.Equal(t, "db", spans[0].Type)
	assert.Equal(t, "sqlite3", spans[0].Subtype)
	assert.Equal(t, "query", spans[0].Action)
	assert.Equal(t, &model.SpanContext{
		Database: &model.DatabaseSpanContext{
			Instance:  ":memory:",
			Statement: "SELECT * FROM foo",
			Type:      "sql",
		},
	}, spans[0].Context)
}

func TestPrepareContext(t *testing.T) {
	db, err := apmsql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	db.Ping() // connect
	_, spans, _ := apmtest.WithTransaction(func(ctx context.Context) {
		stmt, err := db.PrepareContext(ctx, "CREATE TABLE foo (bar INT)")
		require.NoError(t, err)
		defer stmt.Close()
		_, err = stmt.Exec()
		require.NoError(t, err)
	})
	require.Len(t, spans, 1)
	assert.Equal(t, "CREATE", spans[0].Name)
	assert.Equal(t, "db", spans[0].Type)
	assert.Equal(t, "sqlite3", spans[0].Subtype)
	assert.Equal(t, "prepare", spans[0].Action)
}

func TestStmtExecContext(t *testing.T) {
	db, err := apmsql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	_, err = db.Exec("CREATE TABLE foo (bar INT)")
	require.NoError(t, err)

	stmt, err := db.Prepare("DELETE FROM foo WHERE bar < :ceil")
	require.NoError(t, err)
	defer stmt.Close()

	_, spans, _ := apmtest.WithTransaction(func(ctx context.Context) {
		_, err = stmt.ExecContext(ctx, sql.Named("ceil", 999))
		require.NoError(t, err)
	})
	require.Len(t, spans, 1)
	assert.Equal(t, "DELETE FROM foo", spans[0].Name)
	assert.Equal(t, "db", spans[0].Type)
	assert.Equal(t, "sqlite3", spans[0].Subtype)
	assert.Equal(t, "exec", spans[0].Action)
}

func TestStmtQueryContext(t *testing.T) {
	db, err := apmsql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	_, err = db.Exec("CREATE TABLE foo (bar INT)")
	require.NoError(t, err)

	stmt, err := db.Prepare("SELECT * FROM foo")
	require.NoError(t, err)
	defer stmt.Close()

	_, spans, _ := apmtest.WithTransaction(func(ctx context.Context) {
		rows, err := stmt.QueryContext(ctx)
		require.NoError(t, err)
		rows.Close()
	})
	require.Len(t, spans, 1)
	assert.Equal(t, "SELECT FROM foo", spans[0].Name)
	assert.Equal(t, "db", spans[0].Type)
	assert.Equal(t, "sqlite3", spans[0].Subtype)
	assert.Equal(t, "query", spans[0].Action)
}

func TestTxStmtQueryContext(t *testing.T) {
	db, err := apmsql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	_, err = db.Exec("CREATE TABLE foo (bar INT)")
	require.NoError(t, err)

	stmt, err := db.Prepare("SELECT * FROM foo")
	require.NoError(t, err)
	defer stmt.Close()

	_, spans, _ := apmtest.WithTransaction(func(ctx context.Context) {
		tx, err := db.BeginTx(ctx, nil)
		require.NoError(t, err)
		defer tx.Rollback()

		stmt := tx.Stmt(stmt)
		rows, err := stmt.QueryContext(ctx)
		require.NoError(t, err)
		rows.Close()
	})
	require.Len(t, spans, 1)
	assert.Equal(t, "SELECT FROM foo", spans[0].Name)
	assert.Equal(t, "db", spans[0].Type)
	assert.Equal(t, "sqlite3", spans[0].Subtype)
	assert.Equal(t, "query", spans[0].Action)
}

func TestCaptureErrors(t *testing.T) {
	db, err := apmsql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	db.Ping() // connect
	_, spans, errors := apmtest.WithTransaction(func(ctx context.Context) {
		_, err := db.QueryContext(ctx, "SELECT * FROM thin_air")
		require.Error(t, err)
	})
	require.Len(t, spans, 1)
	require.Len(t, errors, 1)
	assert.Equal(t, "SELECT FROM thin_air", spans[0].Name)
	assert.Equal(t, "db", spans[0].Type)
	assert.Equal(t, "sqlite3", spans[0].Subtype)
	assert.Equal(t, "query", spans[0].Action)
	assert.Equal(t, "no such table: thin_air", errors[0].Exception.Message)
}

func TestBadConn(t *testing.T) {
	db, err := apmsql.Open("sqlite3_test", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	defer func() { testQueryContext = nil }()
	testQueryContext = func(context.Context, *sqlite3.SQLiteConn, string, []driver.NamedValue) (driver.Rows, error) {
		return nil, driver.ErrBadConn
	}

	db.Ping() // connect
	_, spans, errors := apmtest.WithTransaction(func(ctx context.Context) {
		_, err := db.QueryContext(ctx, "SELECT * FROM foo")
		require.Error(t, err)
	})

	var attempts int
	for _, span := range spans {
		if span.Name == "SELECT FROM foo" {
			attempts++
		}
	}
	// Two attempts with cached-or-new, followed
	// by one attempt with a new connection.
	assert.Condition(t, func() bool { return attempts == 3 })
	assert.Len(t, errors, 0) // no "bad connection" errors reported
}

func TestContextCanceled(t *testing.T) {
	db, err := apmsql.Open("sqlite3_test", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	defer func() { testQueryContext = nil }()
	testQueryContext = func(ctx context.Context, conn *sqlite3.SQLiteConn, query string, args []driver.NamedValue) (driver.Rows, error) {
		return nil, context.Canceled
	}

	db.Ping() // connect
	_, _, errors := apmtest.WithTransaction(func(ctx context.Context) {
		_, err := db.QueryContext(ctx, "SELECT * FROM foo")
		require.Error(t, err)
	})
	assert.Len(t, errors, 0) // no "context canceled" errors reported
}

type sqlite3TestDriver struct {
	sqlite3.SQLiteDriver
}

func (d *sqlite3TestDriver) Open(name string) (driver.Conn, error) {
	conn, err := d.SQLiteDriver.Open(name)
	if err != nil {
		return conn, err
	}
	return sqlite3TestConn{conn.(*sqlite3.SQLiteConn)}, nil
}

var testQueryContext func(context.Context, *sqlite3.SQLiteConn, string, []driver.NamedValue) (driver.Rows, error)

type sqlite3TestConn struct {
	*sqlite3.SQLiteConn
}

func (d sqlite3TestConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	if testQueryContext != nil {
		return testQueryContext(ctx, d.SQLiteConn, query, args)
	}
	return nil, driver.ErrBadConn
}
