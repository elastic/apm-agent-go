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
		tracer, err := apm.NewTracer("apmhttp_test", "0.1")
		require.NoError(b, err)
		tracer.Transport = httpTransport
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
