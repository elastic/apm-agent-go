module go.elastic.co/apm/module/apmgorm/v2

require (
	github.com/jinzhu/gorm v1.9.10
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.7.0
	go.elastic.co/apm/module/apmsql/v2 v2.4.2
	go.elastic.co/apm/v2 v2.4.2
)

replace go.elastic.co/apm/v2 => ../..

replace go.elastic.co/apm/module/apmsql/v2 => ../apmsql

go 1.15
