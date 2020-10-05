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

package apmgormv2

import (
	"database/sql"
	"github.com/pkg/errors"
	"go.elastic.co/apm/module/apmsql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
)

// Open returns a *gorm.DB for the given dialect and arguments.
// The returned *gorm.DB will have callbacks registered with
// RegisterCallbacks, such that CRUD operations will be reported
// as spans.
//
// Open accepts the following signatures:
//  - a gorm.Dialect
//  - a *gorm.Config
//
// If a driver and datasource name are supplied, and the appropriate
// apmgorm/dialects package has been imported (or the driver has
// otherwise been registered with apmsql), then the datasource name
// will be parsed for inclusion in the span context.
func Open(dialect gorm.Dialector, config *gorm.Config) (*gorm.DB, error) {
	if err := initDialect(dialect); err != nil {
		return nil, errors.WithStack(err)
	}

	db, err := gorm.Open(dialect, config)

	if err != nil {
		return nil, errors.WithStack(err)
	}

	if dialect.Name() == (sqlite.Dialector{}.Name()) {
		dsn, err := extractDsn(dialect)
		if err != nil {
			return nil, err
		}
		conn, err := apmsql.Open(dialect.Name(), dsn)
		if err != nil {
			return nil, err
		}
		if oldConn, ok := db.ConnPool.(*sql.DB); ok {
			_ = oldConn.Close()
		}
		db.ConnPool = conn
	}

	if err := db.Use(NewPlugin()); err != nil {
		return nil, err
	}
	return db, nil
}

// Extracts dsn with given gorm.Dialector
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

func initDialect(dialect gorm.Dialector) error {
	dsn, err := extractDsn(dialect)
	if err != nil {
		return err
	}
	conn, err := apmsql.Open(dialect.Name(), dsn)
	if err != nil {
		return err
	}
	switch dialect.Name() {
	case mysql.Dialector{}.Name():
		if driver, ok := dialect.(mysql.Dialector); !ok {
			return errors.New("unable to cast dialect")
		} else {
			driver.Conn = conn
		}
	case postgres.Dialector{}.Name():
		if driver, ok := dialect.(postgres.Dialector); !ok {
			return errors.New("unable to cast dialect")
		} else {
			driver.Conn = conn
		}
	case sqlite.Dialector{}.Name():
		return nil
	}
	return errors.New("invalid dialect")
}
