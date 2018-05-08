package apmmysql_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/apm-agent-go/module/apmsql"
	"github.com/elastic/apm-agent-go/module/apmsql/mysql"
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
