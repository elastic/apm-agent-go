// +build go1.9

package apmgorm

import (
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"

	"github.com/elastic/apm-agent-go/module/apmsql"
)

// Open returns a *gorm.DB for the given dialect and arguments.
// The returned *gorm.DB will have callbacks registered with
// RegisterCallbacks, such that CRUD operations will be reported
// as spans.
//
// Open accepts the following signatures:
//  - a datasource name (i.e. the second argument to sql.Open)
//  - a driver name and a datasource name
//  - a *sql.DB, or some other type with the same interface
//
// If a driver and datasource name are supplied, and the appropriate
// apmgorm/dialects package has been imported (or the driver has
// otherwise been registered with apmsql), then the datasource name
// will be parsed for inclusion in the span context.
func Open(dialect string, args ...interface{}) (*gorm.DB, error) {
	var driverName, dsn string
	switch len(args) {
	case 1:
		switch arg0 := args[0].(type) {
		case string:
			driverName = dialect
			dsn = arg0
		}
	case 2:
		driverName, _ = args[0].(string)
		dsn, _ = args[1].(string)
	}
	db, err := gorm.Open(dialect, args...)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	registerCallbacks(db, apmsql.DriverDSNParser(driverName)(dsn))
	return db, nil
}
