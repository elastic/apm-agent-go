module go.elastic.co/apm/module/apmgopgv10/v2

require (
	github.com/go-pg/pg/v10 v10.7.3
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.6.1
	go.elastic.co/apm/module/apmsql/v2 v2.2.0
	go.elastic.co/apm/v2 v2.2.0
)

replace go.elastic.co/apm/v2 => ../..

replace go.elastic.co/apm/module/apmsql/v2 => ../apmsql

go 1.15
