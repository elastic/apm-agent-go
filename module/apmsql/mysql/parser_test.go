package apmmysql_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.elastic.co/apm/module/apmsql"
	apmmysql "go.elastic.co/apm/module/apmsql/mysql"
)

func TestParseDSN(t *testing.T) {
	info := apmmysql.ParseDSN("user:pass@/dbname")
	assert.Equal(t, "dbname", info.Database)
	assert.Equal(t, "user", info.User)
}

func TestParseDSNError(t *testing.T) {
	info := apmmysql.ParseDSN("nonsense")
	assert.Equal(t, apmsql.DSNInfo{}, info)
}
