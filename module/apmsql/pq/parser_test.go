package apmpq_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/apm-agent-go/module/apmsql/pq"
)

func TestParseDSNURL(t *testing.T) {
	info := apmpq.ParseDSN("postgresql://user:pass@localhost/dbinst")
	assert.Equal(t, "dbinst", info.Database)
	assert.Equal(t, "user", info.User)
}

func TestParseDSNConnectionString(t *testing.T) {
	info := apmpq.ParseDSN("dbname=foo\\ bar user='baz'")
	assert.Equal(t, "foo bar", info.Database)
	assert.Equal(t, "baz", info.User)
}

func TestParseDSNEnv(t *testing.T) {
	os.Setenv("PGDATABASE", "dbinst")
	os.Setenv("PGUSER", "bob")
	defer os.Unsetenv("PGDATABASE")
	defer os.Unsetenv("PGUSER")

	info := apmpq.ParseDSN("postgres://")
	assert.Equal(t, "dbinst", info.Database)
	assert.Equal(t, "bob", info.User)
}
