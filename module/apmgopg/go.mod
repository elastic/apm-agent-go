module go.elastic.co/apm/module/apmgopg

require (
	github.com/go-pg/pg v8.0.4+incompatible
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/onsi/ginkgo v1.8.0 // indirect
	github.com/onsi/gomega v1.5.0 // indirect
	github.com/stretchr/testify v1.5.1
	go.elastic.co/apm v1.9.0
	go.elastic.co/apm/module/apmsql v1.9.0
	mellium.im/sasl v0.2.1 // indirect
)

replace go.elastic.co/apm => ../..

replace go.elastic.co/apm/module/apmsql => ../apmsql

go 1.13
