package apmsqlite3_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.elastic.co/apm/module/apmsql"
	apmsqlite3 "go.elastic.co/apm/module/apmsql/sqlite3"
)

func TestParseDSN(t *testing.T) {
	assert.Equal(t, apmsql.DSNInfo{Database: "test.db"}, apmsqlite3.ParseDSN("test.db"))
	assert.Equal(t, apmsql.DSNInfo{Database: ":memory:"}, apmsqlite3.ParseDSN(":memory:"))
	assert.Equal(t, apmsql.DSNInfo{Database: "file:test.db"}, apmsqlite3.ParseDSN("file:test.db?cache=shared&mode=memory"))
}
