package apmpgxv5_test

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.elastic.co/apm/module/apmpgxv5/v2"
	"go.elastic.co/apm/v2/apmtest"
	"go.elastic.co/apm/v2/model"
	"os"
	"testing"
)

func Test_QueryTrace(t *testing.T) {
	host := os.Getenv("PGHOST")
	if host == "" {
		host = "localhost"
		//t.Skipf("PGHOST not specified")
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
			_, spans, errs := apmtest.WithTransaction(func(ctx context.Context) {
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
