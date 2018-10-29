package apmsql

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"sync"

	"go.elastic.co/apm/internal/sqlutil"
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
	return sqlutil.QuerySignature(query)
}

func (d *tracingDriver) Open(name string) (driver.Conn, error) {
	conn, err := d.Driver.Open(name)
	if err != nil {
		return nil, err
	}
	return newConn(conn, d, d.dsnParser(name)), nil
}
