package apmpgx

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		t.Run(test.name, func(t *testing.T) {
			tracer := NewTracer(test.logger)
			tracer.Log(context.TODO(), pgx.LogLevelNone, test.msg, nil)
		})
	}
}

func TestQueryTrace(t *testing.T) {
	tracer := NewTracer(nil)

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
		t.Run(test.name, func(t *testing.T) {
			_, spans, errs := apmtest.WithTransaction(func(ctx context.Context) {
				tracer.QueryTrace(ctx, test.data)
			})

			if test.expectErr {
				require.Len(t, errs, 1)
			} else {
				assert.Equal(t, "db", spans[0].Type)
				assert.Equal(t, "postgresql", spans[0].Subtype)
				assert.Equal(t, "query", spans[0].Action)
				assert.Equal(t, "SELECT FROM foo.bar", spans[0].Name)

				require.Len(t, errs, 0)
			}
		})
	}
}

func TestCopyTrace(t *testing.T) {
	tracer := NewTracer(nil)

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
		t.Run(test.name, func(t *testing.T) {
			_, spans, errs := apmtest.WithTransaction(func(ctx context.Context) {
				tracer.CopyTrace(ctx, test.data)
			})

			if test.expectErr {
				require.Len(t, errs, 1)
			} else {
				assert.Equal(t, "db", spans[0].Type)
				assert.Equal(t, "postgresql", spans[0].Subtype)
				assert.Equal(t, "copy", spans[0].Action)
				assert.Equal(t, "COPY TO foo", spans[0].Name)

				require.Len(t, errs, 0)
			}
		})
	}
}
