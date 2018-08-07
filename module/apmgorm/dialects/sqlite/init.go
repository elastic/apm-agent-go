// Package apmgormsqlite imports the gorm sqlite dialect package,
// and also registers the sqlite3 driver with apmsql.
package apmgormsqlite

import (
	_ "github.com/jinzhu/gorm/dialects/sqlite" // import the sqlite dialect

	_ "github.com/elastic/apm-agent-go/module/apmsql/sqlite3" // register sqlite3 with apmsql
)
