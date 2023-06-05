module go.elastic.co/apm/module/apmbeego/v2

require (
	github.com/astaxie/beego v1.10.1
	github.com/stretchr/testify v1.8.4
	go.elastic.co/apm/module/apmhttp/v2 v2.4.2
	go.elastic.co/apm/module/apmsql/v2 v2.4.2
	go.elastic.co/apm/v2 v2.4.2
)

replace go.elastic.co/apm/v2 => ../..

replace go.elastic.co/apm/module/apmhttp/v2 => ../apmhttp

replace go.elastic.co/apm/module/apmsql/v2 => ../apmsql

go 1.15
