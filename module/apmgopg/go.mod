module go.elastic.co/apm/module/apmgopg/v2

require (
	github.com/go-pg/pg v8.0.7+incompatible
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/onsi/ginkgo v1.16.5 // indirect
	github.com/onsi/gomega v1.18.1 // indirect
	github.com/stretchr/testify v1.7.0
	go.elastic.co/apm/module/apmsql/v2 v2.3.0
	go.elastic.co/apm/v2 v2.3.0
	mellium.im/sasl v0.2.1 // indirect
)

replace go.elastic.co/apm/v2 => ../..

replace go.elastic.co/apm/module/apmsql/v2 => ../apmsql

go 1.15
