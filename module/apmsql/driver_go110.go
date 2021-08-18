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

//go:build go1.10
// +build go1.10

package apmsql // import "go.elastic.co/apm/module/apmsql"

import (
	"context"
	"database/sql/driver"

	"go.elastic.co/apm"
)

func (d *tracingDriver) OpenConnector(name string) (driver.Connector, error) {
	if dc, ok := d.Driver.(driver.DriverContext); ok {
		oc, err := dc.OpenConnector(name)
		if err != nil {
			return nil, err
		}
		return &driverConnector{oc.Connect, d, name}, nil
	}
	connect := func(context.Context) (driver.Conn, error) {
		return d.Driver.Open(name)
	}
	return &driverConnector{connect, d, name}, nil
}

type driverConnector struct {
	connect func(context.Context) (driver.Conn, error)
	driver  *tracingDriver
	name    string
}

func (d *driverConnector) Connect(ctx context.Context) (driver.Conn, error) {
	span, ctx := apm.StartSpan(ctx, "connect", d.driver.connectSpanType)
	defer span.End()
	dsnInfo := d.driver.dsnParser(d.name)
	if !span.Dropped() {
		span.Context.SetDatabase(apm.DatabaseSpanContext{
			Instance: dsnInfo.Database,
			Type:     "sql",
			User:     dsnInfo.User,
		})
	}
	conn, err := d.connect(ctx)
	if err != nil {
		return nil, err
	}
	return newConn(conn, d.driver, dsnInfo), nil
}

func (d *driverConnector) Driver() driver.Driver {
	return d.driver
}
