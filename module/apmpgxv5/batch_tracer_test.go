package apmpgxv5_test

import (
	"context"
	"fmt"
	"github.com/gvencadze/apm-agent-go/module/apmpgxv5"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.elastic.co/apm/v2/apmtest"
	"go.elastic.co/apm/v2/model"
	_ "go.elastic.co/apm/v2/model"
	"os"
	"testing"
)

func TestBatchTrace(t *testing.T) {
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
		name       string
		expectErr  bool
		queryQueue []string
	}{
		{
			name:      "BATCH spans, success",
			expectErr: false,
			queryQueue: []string{
				"SELECT * FROM foo WHERE bar = 1",
				"SELECT bar FROM foo WHERE bar = 1",
			},
		},
		{
			name:      "BATCH spans, error",
			expectErr: true,
			queryQueue: []string{
				"SELECT * FROM foo WHERE bar = 1",
				"SELECT bar FROM foo2",
			},
		},
	}

	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			_, spans, errs := apmtest.WithTransaction(func(ctx context.Context) {
				batch := &pgx.Batch{}

				for _, query := range tt.queryQueue {
					batch.Queue(query)
				}

				br := conn.SendBatch(ctx, batch)
				defer func() {
					_ = br.Close()
				}()
			})

			if tt.expectErr {
				require.Len(t, errs, 2)
				assert.Equal(t, "failure", spans[0].Outcome)
			} else {
				for i, expectedStmt := range tt.queryQueue {
					assert.Equal(t, "success", spans[i].Outcome)
					assert.Equal(t, "db", spans[i].Type)
					assert.Equal(t, "postgresql", spans[i].Subtype)
					assert.Equal(t, "batch", spans[i].Action)
					assert.Equal(t, expectedStmt, spans[i].Name)

					assert.Equal(t, &model.SpanContext{
						Destination: &model.DestinationSpanContext{
							Address: cfg.Host,
							Port:    int(cfg.Port),
						},
						Database: &model.DatabaseSpanContext{
							Instance:  cfg.Database,
							Statement: expectedStmt,
							Type:      "sql",
							User:      cfg.User,
						},
					}, spans[i].Context)
				}
			}
		})
	}
}
