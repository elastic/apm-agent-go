package apmpq_test

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/apm-agent-go/apmtest"
	"github.com/elastic/apm-agent-go/model"
	"github.com/elastic/apm-agent-go/module/apmsql"
	_ "github.com/elastic/apm-agent-go/module/apmsql/pq"
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

	tx, _ := apmtest.WithTransaction(func(ctx context.Context) {
		rows, err := db.QueryContext(ctx, "SELECT * FROM foo")
		require.NoError(t, err)
		rows.Close()
	})
	require.Len(t, tx.Spans, 1)

	assert.NotNil(t, tx.Spans[0].ID)
	assert.Equal(t, "SELECT FROM foo", tx.Spans[0].Name)
	assert.Equal(t, "db.postgresql.query", tx.Spans[0].Type)
	assert.Equal(t, &model.SpanContext{
		Database: &model.DatabaseSpanContext{
			Instance:  "test_db",
			Statement: "SELECT * FROM foo",
			Type:      "sql",
			User:      "postgres",
		},
	}, tx.Spans[0].Context)
}
