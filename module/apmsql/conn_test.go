package apmsql_test

import (
	"context"
	"database/sql/driver"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"go.elastic.co/apm/module/apmsql"
	_ "go.elastic.co/apm/module/apmsql/sqlite3"
)

func TestConnUnwrap(t *testing.T) {
	db, err := apmsql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	type Unwrapper interface {
		Unwrap() driver.Conn
	}

	ctx := context.Background()
	conn, err := db.Conn(ctx)
	_ = conn.Raw(func(driverConn interface{}) error {
		unwrappedConn := driverConn.(Unwrapper)
		require.Equal(t, "*sqlite3.SQLiteConn", fmt.Sprintf("%T", unwrappedConn.Unwrap()))
		return nil
	})
}
