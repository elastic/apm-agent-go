package apmpq_test

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.elastic.co/apm/apmtest"
	"go.elastic.co/apm/model"
	"go.elastic.co/apm/module/apmsql"
	_ "go.elastic.co/apm/module/apmsql/pq"
)

func TestQueryContext(t *testing.T) {
	if os.Getenv("PGHOST") == "" {
		t.Skipf("PGHOST not specified")
	}

	db, err := apmsql.Open("postgres", "user=postgres password=hunter2 dbname=test_db sslmode=disable")
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
	assert.Equal(t, "db.postgresql.query", spans[0].Type)
	assert.Equal(t, &model.SpanContext{
		Database: &model.DatabaseSpanContext{
			Instance:  "test_db",
			Statement: "SELECT * FROM foo",
			Type:      "sql",
			User:      "postgres",
		},
	}, spans[0].Context)
}
