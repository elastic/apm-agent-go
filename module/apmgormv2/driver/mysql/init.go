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

// Package apmmysql imports the gorm mysql dialect package,
// and also registers the mysql driver with apmsql.
package apmmysql // import "go.elastic.co/apm/module/apmgormv2/v2/driver/mysql"

import (
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"go.elastic.co/apm/module/apmsql/v2"

	_ "go.elastic.co/apm/module/apmsql/v2/mysql" // register mysql with apmsql
)

// Open creates a dialect with apmsql
func Open(dsn string) gorm.Dialector {
	driverName := mysql.Dialector{}.Name()
	dialect := &mysql.Dialector{
		Config: &mysql.Config{
			DriverName: apmsql.DriverPrefix + driverName,
			DSN:        dsn,
		},
	}

	return dialect
}
