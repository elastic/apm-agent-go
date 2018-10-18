// Package apmgormmysql imports the gorm mysql dialect package,
// and also registers the mysql driver with apmsql.
package apmgormmysql

import (
	_ "github.com/jinzhu/gorm/dialects/mysql" // import the mysql dialect

	_ "go.elastic.co/apm/module/apmsql/mysql" // register mysql with apmsql
)
