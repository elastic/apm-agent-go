module go.elastic.co/apm/module/apmgopg

require (
	github.com/go-pg/pg v8.0.4+incompatible
	github.com/jinzhu/inflection v0.0.0-20180308033659-04140366298a // indirect
	github.com/onsi/ginkgo v1.8.0 // indirect
	github.com/onsi/gomega v1.5.0 // indirect
	github.com/stretchr/testify v1.3.0
	go.elastic.co/apm v1.4.0
	go.elastic.co/apm/module/apmsql v1.4.0
	mellium.im/sasl v0.2.1 // indirect
)

replace go.elastic.co/apm => ../..

replace go.elastic.co/apm/module/apmsql => ../apmsql
