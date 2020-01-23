module go.elastic.co/apm/module/apmgopg

require (
	github.com/go-pg/pg/v9 v9.1.1
	github.com/stretchr/testify v1.4.0
	go.elastic.co/apm v1.7.0
	go.elastic.co/apm/module/apmsql v1.7.0
)

replace go.elastic.co/apm => ../..

replace go.elastic.co/apm/module/apmsql => ../apmsql

go 1.13
