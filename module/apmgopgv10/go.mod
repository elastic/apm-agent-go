module go.elastic.co/apm/module/apmgopgv10/v2

require (
	github.com/go-pg/pg/v10 v10.7.3
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.8.4
	go.elastic.co/apm/module/apmsql/v2 v2.4.7
	go.elastic.co/apm/v2 v2.4.7
)

require (
	github.com/armon/go-radix v1.0.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/elastic/go-sysinfo v1.7.1 // indirect
	github.com/elastic/go-windows v1.0.0 // indirect
	github.com/go-pg/zerochecker v0.2.0 // indirect
	github.com/google/go-cmp v0.5.4 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/joeshaw/multierror v0.0.0-20140124173710-69b34d4ec901 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/procfs v0.0.0-20190425082905-87a4384529e0 // indirect
	github.com/tmthrgd/go-hex v0.0.0-20190904060850-447a3041c3bc // indirect
	github.com/vmihailenco/bufpool v0.1.11 // indirect
	github.com/vmihailenco/msgpack/v5 v5.0.0 // indirect
	github.com/vmihailenco/tagparser v0.1.2 // indirect
	go.elastic.co/fastjson v1.1.0 // indirect
	go.opentelemetry.io/otel v0.14.0 // indirect
	golang.org/x/crypto v0.17.0 // indirect
	golang.org/x/sys v0.15.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	howett.net/plist v0.0.0-20181124034731-591f970eefbb // indirect
	mellium.im/sasl v0.3.1 // indirect
)

replace go.elastic.co/apm/v2 => ../..

replace go.elastic.co/apm/module/apmsql/v2 => ../apmsql

go 1.19
