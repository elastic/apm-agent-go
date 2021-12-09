module go.elastic.co/apm/module/apmgorm/v2

require (
	cloud.google.com/go v0.40.0 // indirect
	github.com/jinzhu/gorm v1.9.10
	github.com/pkg/errors v0.8.1
	github.com/stretchr/testify v1.6.1
	go.elastic.co/apm/module/apmsql/v2 v2.0.0
	go.elastic.co/apm/v2 v2.0.0
)

replace go.elastic.co/apm/v2 => ../..

replace go.elastic.co/apm/module/apmsql/v2 => ../apmsql

go 1.13
