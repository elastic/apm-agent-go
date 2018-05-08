package apmsqlite3_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/apm-agent-go/module/apmsql"
	"github.com/elastic/apm-agent-go/module/apmsql/sqlite3"
)

func TestParseDSN(t *testing.T) {
	assert.Equal(t, apmsql.DSNInfo{Database: "test.db"}, apmsqlite3.ParseDSN("test.db"))
	assert.Equal(t, apmsql.DSNInfo{Database: ":memory:"}, apmsqlite3.ParseDSN(":memory:"))
	assert.Equal(t, apmsql.DSNInfo{Database: "file:test.db"}, apmsqlite3.ParseDSN("file:test.db?cache=shared&mode=memory"))
}
