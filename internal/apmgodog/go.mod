module apmgodog

go 1.13

require (
	github.com/cucumber/godog v0.12.2
	go.elastic.co/apm v1.15.0
	go.elastic.co/apm/module/apmgrpc v1.15.0
	go.elastic.co/apm/module/apmhttp v1.15.0
	go.elastic.co/fastjson v1.1.0
	google.golang.org/grpc v1.21.1
)

replace go.elastic.co/apm => ../..

replace go.elastic.co/apm/module/apmgrpc => ../../module/apmgrpc

replace go.elastic.co/apm/module/apmhttp => ../../module/apmhttp
