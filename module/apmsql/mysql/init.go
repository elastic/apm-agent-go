package apmmysql

import (
	"github.com/go-sql-driver/mysql"

	"go.elastic.co/apm/module/apmsql"
)

func init() {
	apmsql.Register("mysql", &mysql.MySQLDriver{}, apmsql.WithDSNParser(ParseDSN))
}
