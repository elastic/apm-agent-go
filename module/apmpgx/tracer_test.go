package apmpgx

import (
	"context"
	"github.com/jackc/pgx/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.elastic.co/apm/v2/apmtest"
	"testing"
	"time"
)

func TestQueryTrace(t *testing.T) {
	tracer := NewTracer(nil)

	_, spans, errors := apmtest.WithTransaction(func(ctx context.Context) {
		tracer.QueryTrace(ctx, map[string]interface{}{
			"time": 3 * time.Millisecond,
			"sql":  "SELECT * FROM foo.bar",
		})
	})

	assert.Equal(t, "db", spans[0].Type)
	assert.Equal(t, "postgresql", spans[0].Subtype)
	assert.Equal(t, "query", spans[0].Action)
	assert.Equal(t, "SELECT FROM foo.bar", spans[0].Name)

	require.Len(t, errors, 0)
}

func TestCopyTrace(t *testing.T) {
	tracer := NewTracer(nil)

	_, spans, errors := apmtest.WithTransaction(func(ctx context.Context) {
		tracer.CopyTrace(ctx, map[string]interface{}{
			"time":        3 * time.Millisecond,
			"tableName":   pgx.Identifier{"foo"},
			"columnNames": pgx.Identifier{"id,name,age"},
		})
	})

	assert.Equal(t, "db", spans[0].Type)
	assert.Equal(t, "postgresql", spans[0].Subtype)
	assert.Equal(t, "copyFrom", spans[0].Action)
	assert.Equal(t, "COPY TO foo", spans[0].Name)

	require.Len(t, errors, 0)
}
