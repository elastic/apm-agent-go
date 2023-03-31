module apmgodog

go 1.13

require (
	github.com/cucumber/godog v0.12.2
	go.elastic.co/apm/module/apmgrpc/v2 v2.3.0
	go.elastic.co/apm/module/apmhttp/v2 v2.3.0
	go.elastic.co/apm/v2 v2.3.0
	go.elastic.co/fastjson v1.1.0
	google.golang.org/grpc v1.21.1
)

replace go.elastic.co/apm/v2 => ../..

replace go.elastic.co/apm/module/apmgrpc/v2 => ../../module/apmgrpc

replace go.elastic.co/apm/module/apmhttp/v2 => ../../module/apmhttp
