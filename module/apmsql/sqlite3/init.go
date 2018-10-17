package apmsqlite3

import (
	"github.com/mattn/go-sqlite3"

	"go.elastic.co/apm/module/apmsql"
)

func init() {
	apmsql.Register("sqlite3", &sqlite3.SQLiteDriver{}, apmsql.WithDSNParser(ParseDSN))
}
