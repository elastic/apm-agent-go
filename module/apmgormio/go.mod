module go.elastic.co/apm/module/apmgormio

require (
	cloud.google.com/go v0.40.0 // indirect
	github.com/jinzhu/gorm v1.9.10
	github.com/pkg/errors v0.8.1
	github.com/stretchr/testify v1.5.1
	go.elastic.co/apm v1.8.0
	go.elastic.co/apm/module/apmsql v1.8.0
	gorm.io/driver/mysql v1.0.2
	gorm.io/driver/postgres v1.0.2
	gorm.io/driver/sqlite v1.1.3
)

replace go.elastic.co/apm => ../..

replace go.elastic.co/apm/module/apmsql => ../apmsql

go 1.13
