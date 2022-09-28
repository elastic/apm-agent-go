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

package apmsql // import "go.elastic.co/apm/module/apmsql/v2"

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"sync"

	"go.elastic.co/apm/v2"
	"go.elastic.co/apm/v2/sqlutil"
)

// DriverPrefix should be used as a driver name prefix when
// registering via sql.Register.
const DriverPrefix = "apm/"

var (
	driversMu sync.RWMutex
	drivers   = make(map[string]*tracingDriver)
)

// Register registers a traced version of the given driver.
//
// The name and driver values should be the same as given to
// sql.Register: the name of the driver (e.g. "postgres"), and
// the driver (e.g. &github.com/lib/pq.Driver{}).
func Register(name string, driver driver.Driver, opts ...WrapOption) {
	driversMu.Lock()
	defer driversMu.Unlock()

	wrapped := newTracingDriver(driver, opts...)
	sql.Register(DriverPrefix+name, wrapped)
	drivers[name] = wrapped
}

// Open opens a database with the given driver and data source names,
// as in sql.Open. The driver name should be one registered via the
// Register function in this package.
func Open(driverName, dataSourceName string) (*sql.DB, error) {
	return sql.Open(DriverPrefix+driverName, dataSourceName)
}

// Wrap wraps a database/sql/driver.Driver such that
// the driver's database methods are traced. The tracer
// will be obtained from the context supplied to methods
// that accept it.
func Wrap(driver driver.Driver, opts ...WrapOption) driver.Driver {
	return newTracingDriver(driver, opts...)
}

func newTracingDriver(driver driver.Driver, opts ...WrapOption) *tracingDriver {
	d := &tracingDriver{
		Driver: driver,
	}
	for _, opt := range opts {
		opt(d)
	}
	if d.driverName == "" {
		d.driverName = sqlutil.DriverName(driver)
	}
	if d.dsnParser == nil {
		d.dsnParser = genericDSNParser
	}
	// store span types to avoid repeat allocations
	d.connectSpanType = d.formatSpanType("connect")
	d.pingSpanType = d.formatSpanType("ping")
	d.prepareSpanType = d.formatSpanType("prepare")
	d.querySpanType = d.formatSpanType("query")
	d.execSpanType = d.formatSpanType("exec")
	return d
}

// DriverDSNParser returns the DSNParserFunc for the registered driver.
// If there is no such registered driver, the parser function that is
// returned will return empty DSNInfo structures.
func DriverDSNParser(driverName string) DSNParserFunc {
	driversMu.RLock()
	driver := drivers[driverName]
	defer driversMu.RUnlock()

	if driver == nil {
		return genericDSNParser
	}
	return driver.dsnParser
}

// WrapOption is an option that can be supplied to Wrap.
type WrapOption func(*tracingDriver)

// WithDriverName returns a WrapOption which sets the underlying
// driver name to the specified value. If WithDriverName is not
// supplied to Wrap, the driver name will be inferred from the
// driver supplied to Wrap.
func WithDriverName(name string) WrapOption {
	return func(d *tracingDriver) {
		d.driverName = name
	}
}

// WithDSNParser returns a WrapOption which sets the function to
// use for parsing the data source name. If WithDSNParser is not
// supplied to Wrap, the function to use will be inferred from
// the driver name.
func WithDSNParser(f DSNParserFunc) WrapOption {
	return func(d *tracingDriver) {
		d.dsnParser = f
	}
}

type tracingDriver struct {
	driver.Driver
	driverName string
	dsnParser  DSNParserFunc

	connectSpanType string
	execSpanType    string
	pingSpanType    string
	prepareSpanType string
	querySpanType   string
}

func (d *tracingDriver) formatSpanType(suffix string) string {
	return fmt.Sprintf("db.%s.%s", d.driverName, suffix)
}

// querySignature returns the value to use in Span.Name for
// a database query.
func (d *tracingDriver) querySignature(query string) string {
	return QuerySignature(query)
}

// Unwrap returns the wrapped database/sql/driver.Driver.
func (d *tracingDriver) Unwrap() driver.Driver {
	return d.Driver
}

func (d *tracingDriver) Open(name string) (driver.Conn, error) {
	conn, err := d.Driver.Open(name)
	if err != nil {
		return nil, err
	}
	return newConn(conn, d, d.dsnParser(name)), nil
}

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
	span, ctx := apm.StartSpanOptions(ctx, "connect", d.driver.connectSpanType, apm.SpanOptions{
		ExitSpan: true,
	})
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
