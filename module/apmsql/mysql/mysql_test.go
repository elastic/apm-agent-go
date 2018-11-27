package apmmysql_test

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.elastic.co/apm/apmtest"
	"go.elastic.co/apm/model"
	"go.elastic.co/apm/module/apmsql"
	_ "go.elastic.co/apm/module/apmsql/mysql"
)

var mysqlHost = os.Getenv("MYSQL_HOST")

func TestQueryContext(t *testing.T) {
	if mysqlHost == "" {
		t.Skipf("MYSQL_HOST not specified")
	}

	db, err := apmsql.Open("mysql", "root:hunter2@tcp("+mysqlHost+")/test_db")
	require.NoError(t, err)
	defer db.Close()

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS foo (bar INT)")
	require.NoError(t, err)

	_, spans, _ := apmtest.WithTransaction(func(ctx context.Context) {
		rows, err := db.QueryContext(ctx, "SELECT * FROM foo")
		require.NoError(t, err)
		rows.Close()
	})
	require.Len(t, spans, 1)

	assert.NotNil(t, spans[0].ID)
	assert.Equal(t, "SELECT FROM foo", spans[0].Name)
	assert.Equal(t, "mysql", spans[0].Subtype)
	assert.Equal(t, &model.SpanContext{
		Database: &model.DatabaseSpanContext{
			Instance:  "test_db",
			Statement: "SELECT * FROM foo",
			Type:      "sql",
			User:      "root",
		},
	}, spans[0].Context)
}
