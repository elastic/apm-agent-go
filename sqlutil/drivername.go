// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package sqlutil // import "go.elastic.co/apm/v2/sqlutil"

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

		if strings.HasPrefix(t.PkgPath(), "github.com/jackc/pgx/") {
			return "postgresql"
		}

		if strings.HasPrefix(t.PkgPath(), "github.com/denisenkom/go-mssqldb") {
			return "sqlserver"
		}
	}
	// TODO include the package path of the driver in context
	// so we can easily determine how the rules above should
	// be updated.
	return "generic"
}
