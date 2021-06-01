module go.elastic.co/apm/module/apmbeego

require (
	github.com/astaxie/beego v1.11.1
	github.com/stretchr/testify v1.6.1
	go.elastic.co/apm v1.12.0
	go.elastic.co/apm/module/apmhttp v1.12.0
	go.elastic.co/apm/module/apmsql v1.12.0
)

replace go.elastic.co/apm => ../..

replace go.elastic.co/apm/module/apmhttp => ../apmhttp

replace go.elastic.co/apm/module/apmsql => ../apmsql

go 1.13
