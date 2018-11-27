// +build go1.10

package apmsql_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.elastic.co/apm/apmtest"
	"go.elastic.co/apm/module/apmsql"
)

func TestConnect(t *testing.T) {
	db, err := apmsql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Note: only in Go 1.10 do we have context during connection.

	_, spans, _ := apmtest.WithTransaction(func(ctx context.Context) {
		err := db.PingContext(ctx)
		assert.NoError(t, err)
	})
	require.Len(t, spans, 2)
	assert.Equal(t, "connect", spans[0].Name)
	assert.Equal(t, "connect", spans[0].Action)
	assert.Equal(t, "ping", spans[1].Name)
	assert.Equal(t, "ping", spans[1].Action)
}
