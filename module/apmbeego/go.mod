module go.elastic.co/apm/module/apmbeego

require (
	github.com/astaxie/beego v1.11.1
	github.com/stretchr/testify v1.3.0
	go.elastic.co/apm v1.5.0
	go.elastic.co/apm/module/apmhttp v1.5.0
	go.elastic.co/apm/module/apmsql v1.5.0
	golang.org/x/crypto v0.0.0-20190701094942-4def268fd1a4 // indirect
)

replace go.elastic.co/apm => ../..

replace go.elastic.co/apm/module/apmhttp => ../apmhttp

replace go.elastic.co/apm/module/apmsql => ../apmsql
