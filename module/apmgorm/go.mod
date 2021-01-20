module go.elastic.co/apm/module/apmgorm

require (
	cloud.google.com/go v0.40.0 // indirect
	github.com/jinzhu/gorm v1.9.10
	github.com/pkg/errors v0.8.1
	github.com/stretchr/testify v1.5.1
	go.elastic.co/apm v1.10.0
	go.elastic.co/apm/module/apmsql v1.10.0
)

replace go.elastic.co/apm => ../..

replace go.elastic.co/apm/module/apmsql => ../apmsql

go 1.13
