package apmpgxv5_test

import (
	"context"
	"fmt"
	"github.com/gvencadze/apm-agent-go/module/apmpgxv5"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.elastic.co/apm/v2/apmtest"
	"os"
	"testing"
)

// TODO: add tests for connect
func Test_Connect(t *testing.T) {
	host := os.Getenv("PGHOST")
	if host == "" {
		t.Skipf("PGHOST not specified")
	}

	testcases := []struct {
		name      string
		dsn       string
		expectErr bool
	}{
		{
			name:      "CONNECT span, success",
			expectErr: false,
			dsn:       "postgres://postgres:hunter2@%s:5432/test_db",
		},
		{
			name:      "CONNECT span, failure",
			expectErr: true,
			dsn:       "postgres://postgres:hunter2@%s:5432/non_existing_db",
		},
	}

	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := pgx.ParseConfig(fmt.Sprintf(tt.dsn, host))
			require.NoError(t, err)

			apmpgxv5.Instrument(cfg)

			_, spans, errs := apmtest.WithTransaction(func(ctx context.Context) {
				_, _ = pgx.ConnectConfig(ctx, cfg)
			})

			assert.NotNil(t, spans[0].ID)

			if tt.expectErr {
				require.Len(t, errs, 1)
				assert.Equal(t, "failure", spans[0].Outcome)
				assert.Equal(t, "connect", spans[0].Name)
			} else {
				assert.Equal(t, "success", spans[0].Outcome)
				assert.Equal(t, "connect", spans[0].Name)
			}
		})
	}
}
