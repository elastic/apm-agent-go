module go.elastic.co/apm/module/apmgopg

require (
	github.com/go-pg/pg/v10 v10.7.3
	github.com/stretchr/testify v1.6.1
	go.elastic.co/apm v1.9.0
	go.elastic.co/apm/module/apmsql v1.9.0
)

replace go.elastic.co/apm => ../..

replace go.elastic.co/apm/module/apmsql => ../apmsql

go 1.13
