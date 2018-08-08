// Package apmgormmysql imports the gorm mysql dialect package,
// and also registers the mysql driver with apmsql.
package apmgormmysql

import (
	_ "github.com/jinzhu/gorm/dialects/mysql" // import the mysql dialect

	_ "github.com/elastic/apm-agent-go/module/apmsql/mysql" // register mysql with apmsql
)
