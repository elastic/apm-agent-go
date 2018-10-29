// Package apmgormpostgres imports the gorm postgres dialect package,
// and also registers the lib/pq driver with apmsql.
package apmgormpostgres

import (
	_ "github.com/jinzhu/gorm/dialects/postgres" // import the postgres dialect

	_ "go.elastic.co/apm/module/apmsql/pq" // register lib/pq with apmsql
)
