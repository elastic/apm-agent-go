package sqlutil

import (
	"database/sql/driver"
	"reflect"
	"strings"
)

// DriverName returns the name of the driver, based on its type.
// If the driver name cannot be deduced, DriverName will return
// "generic".
func DriverName(d driver.Driver) string {
	t := reflect.TypeOf(d)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	switch t.Name() {
	case "SQLiteDriver":
		return "sqlite3"
	case "MySQLDriver":
		return "mysql"
	case "Driver":
		// Check suffix in case of vendoring.
		if strings.HasSuffix(t.PkgPath(), "github.com/lib/pq") {
			return "postgresql"
		}
	}
	// TODO include the package path of the driver in context
	// so we can easily determine how the rules above should
	// be updated.
	return "generic"
}
