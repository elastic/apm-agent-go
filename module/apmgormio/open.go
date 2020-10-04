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

// +build go1.9

package apmgormio

import (
	"github.com/pkg/errors"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"go.elastic.co/apm/module/apmsql"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
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
func Open(dialect gorm.Dialector, config *gorm.Config) (*gorm.DB, error) {
	dsn, err := extractDsn(dialect)
	if err != nil {
		return nil, err
	}

	db, err := gorm.Open(dialect, config)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	registerCallbacks(db, apmsql.DriverDSNParser(dialect.Name())(dsn))
	return db, nil
}

func extractDsn(dialect gorm.Dialector) (string, error) {
	switch dialect.Name() {
	case mysql.Dialector{}.Name():
		if driver, ok := dialect.(mysql.Dialector); !ok {
			return "", errors.New("unable to cast dialect")
		} else {
			return driver.DSN, nil
		}
	case postgres.Dialector{}.Name():
		if driver, ok := dialect.(postgres.Dialector); !ok {
			return "", errors.New("unable to cast dialect")
		} else {
			return driver.DSN, nil
		}
	case sqlite.Dialector{}.Name():
		if driver, ok := dialect.(sqlite.Dialector); !ok {
			return "", errors.New("unable to cast dialect")
		} else {
			return driver.DSN, nil
		}
	}
	return "", errors.New("dialect not supported")
}
