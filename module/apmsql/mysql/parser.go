package apmmysql

import (
	"github.com/go-sql-driver/mysql"

	"go.elastic.co/apm/module/apmsql"
)

// ParseDSN parses the given go-sql-driver/mysql datasource name.
func ParseDSN(name string) apmsql.DSNInfo {
	cfg, err := mysql.ParseDSN(name)
	if err != nil {
		// mysql.Open will fail with the same error,
		// so just return a zero value.
		return apmsql.DSNInfo{}
	}
	return apmsql.DSNInfo{
		Database: cfg.DBName,
		User:     cfg.User,
	}
}
