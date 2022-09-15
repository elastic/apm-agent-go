package apmpgx_test

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.elastic.co/apm/module/apmpgx/v2"
	"go.elastic.co/apm/v2/apmtest"
	"testing"
	"time"
)

type testLogger struct{}

func (t *testLogger) Log(_ context.Context, _ pgx.LogLevel, _ string, _ map[string]interface{}) {}

func TestLog(t *testing.T) {
	testcases := []struct {
		name, msg string
		logger    pgx.Logger
	}{
		{
			name:   "QUERY trace",
			msg:    "Query",
			logger: nil,
		},
		{
			name:   "EXEC trace",
			msg:    "Exec",
			logger: nil,
		},
		{
			name:   "COPY FROM trace",
			msg:    "CopyFrom",
			logger: nil,
		},
		{
			name:   "logger test",
			msg:    "Query",
			logger: &testLogger{},
		},
	}

	for _, test := range testcases {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			cfg := &pgx.ConnConfig{
				Logger: nil,
			}

			apmpgx.Instrument(cfg)

			cfg.Logger.Log(context.TODO(), pgx.LogLevelNone, test.msg, nil)
		})
	}
}

func TestQueryTrace(t *testing.T) {
	cfg := &pgx.ConnConfig{
		Logger: nil,
	}

	apmpgx.Instrument(cfg)

	testcases := []struct {
		name      string
		expectErr bool
		data      map[string]interface{}
	}{
		{
			name:      "QUERY span, no error",
			expectErr: false,
			data: map[string]interface{}{
				"time": 3 * time.Millisecond,
				"sql":  "SELECT * FROM foo.bar",
			},
		},
		{
			name:      "QUERY span, empty data",
			expectErr: true,
			data:      nil,
		},
		{
			name:      "QUERY span, error in data object",
			expectErr: true,
			data: map[string]interface{}{
				"time": 3 * time.Millisecond,
				"sql":  "SELECT * FROM foo.bar",
				"err":  errors.New("test error"),
			},
		},
	}

	for _, test := range testcases {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, spans, errs := apmtest.WithTransaction(func(ctx context.Context) {
				cfg.Logger.Log(ctx, pgx.LogLevelNone, "Query", test.data)
			})

			if test.expectErr {
				require.Len(t, errs, 1)
			} else {
				assert.Equal(t, "db", spans[0].Type)
				assert.Equal(t, "postgresql", spans[0].Subtype)
				assert.Equal(t, "query", spans[0].Action)
				assert.Equal(t, "SELECT FROM foo.bar", spans[0].Name)

				assert.Equal(t, 0, len(spans[0].Stacktrace))

				require.Len(t, errs, 0)
			}
		})
	}
}

func TestCopyTrace(t *testing.T) {
	cfg := &pgx.ConnConfig{
		Logger: nil,
	}

	apmpgx.Instrument(cfg)

	testcases := []struct {
		name      string
		expectErr bool
		data      map[string]interface{}
	}{
		{
			name:      "COPY span, no error",
			expectErr: false,
			data: map[string]interface{}{
				"time":        3 * time.Millisecond,
				"tableName":   pgx.Identifier{"foo"},
				"columnNames": pgx.Identifier{"id,name,age"},
			},
		},
		{
			name:      "COPY span, empty data",
			expectErr: true,
			data:      nil,
		},
		{
			name:      "COPY span, error in data object",
			expectErr: true,
			data: map[string]interface{}{
				"time":        3 * time.Millisecond,
				"tableName":   pgx.Identifier{"foo"},
				"columnNames": pgx.Identifier{"id,name,age"},
				"err":         errors.New("test error"),
			},
		},
	}

	for _, test := range testcases {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, spans, errs := apmtest.WithTransaction(func(ctx context.Context) {
				cfg.Logger.Log(ctx, pgx.LogLevelNone, "CopyFrom", test.data)
			})

			if test.expectErr && test.data != nil {
				require.Len(t, errs, 1)
			}

			if !test.expectErr {
				assert.Equal(t, "db", spans[0].Type)
				assert.Equal(t, "postgresql", spans[0].Subtype)
				assert.Equal(t, "copy", spans[0].Action)
				assert.Equal(t, "COPY TO foo", spans[0].Name)

				assert.Equal(t, len(spans[0].Stacktrace), 0)

				require.Len(t, errs, 0)
			}
		})
	}
}

func TestBatchTrace(t *testing.T) {
	cfg := &pgx.ConnConfig{
		Logger: nil,
	}

	apmpgx.Instrument(cfg)

	testcases := []struct {
		name      string
		expectErr bool
		data      map[string]interface{}
	}{
		{
			name:      "BATCH span, no error",
			expectErr: false,
			data: map[string]interface{}{
				"time":     3 * time.Millisecond,
				"batchLen": 5,
			},
		},
		{
			name:      "BATCH span, no batch len",
			expectErr: false,
			data: map[string]interface{}{
				"time": 3 * time.Millisecond,
			},
		},
		{
			name:      "BATCH span, error in data object",
			expectErr: true,
			data: map[string]interface{}{
				"time":     3 * time.Millisecond,
				"batchLen": 5,
				"err":      errors.New("test error"),
			},
		},
	}

	for _, test := range testcases {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, spans, errs := apmtest.WithTransaction(func(ctx context.Context) {
				cfg.Logger.Log(ctx, pgx.LogLevelNone, "SendBatch", test.data)
			})

			if test.expectErr {
				require.Len(t, errs, 1)
			} else {
				assert.Equal(t, "db", spans[0].Type)
				assert.Equal(t, "postgresql", spans[0].Subtype)
				assert.Equal(t, "batch", spans[0].Action)
				assert.Equal(t, "BATCH", spans[0].Name)

				assert.Equal(t, len(spans[0].Stacktrace), 0)

				if len(spans[0].Context.Tags) > 0 {
					assert.Equal(t, float64(5), spans[0].Context.Tags[0].Value)
				}

				require.Len(t, errs, 0)
			}
		})
	}
}
