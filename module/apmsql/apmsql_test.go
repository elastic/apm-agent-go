package apmsql_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.elastic.co/apm/apmtest"
	"go.elastic.co/apm/model"
	"go.elastic.co/apm/module/apmsql"
	_ "go.elastic.co/apm/module/apmsql/sqlite3"
)

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
	_, spans, _ := apmtest.WithTransaction(func(ctx context.Context) {
		_, err := db.ExecContext(ctx, "CREATE TABLE foo (bar INT)")
		require.NoError(t, err)
	})
	require.Len(t, spans, 1)
	assert.Equal(t, "CREATE", spans[0].Name)
	assert.Equal(t, "db", spans[0].Type)
	assert.Equal(t, "sqlite3", spans[0].Subtype)
	assert.Equal(t, "exec", spans[0].Action)
}

func TestQueryContext(t *testing.T) {
	db, err := apmsql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	_, err = db.Exec("CREATE TABLE foo (bar INT)")
	require.NoError(t, err)

	_, spans, _ := apmtest.WithTransaction(func(ctx context.Context) {
		rows, err := db.QueryContext(ctx, "SELECT * FROM foo")
		require.NoError(t, err)
		rows.Close()
	})
	require.Len(t, spans, 1)

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
