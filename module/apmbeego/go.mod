module go.elastic.co/apm/module/apmbeego

require (
	github.com/astaxie/beego v1.11.1
	github.com/stretchr/testify v1.4.0
	go.elastic.co/apm v1.7.1
	go.elastic.co/apm/module/apmhttp v1.7.1
	go.elastic.co/apm/module/apmsql v1.7.1
	golang.org/x/crypto v0.0.0-20191206172530-e9b2fee46413 // indirect
)

replace go.elastic.co/apm => ../..

replace go.elastic.co/apm/module/apmhttp => ../apmhttp

replace go.elastic.co/apm/module/apmsql => ../apmsql

go 1.13
