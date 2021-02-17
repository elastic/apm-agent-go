module apmgodog

go 1.13

require (
	github.com/cucumber/godog v0.8.1
	go.elastic.co/apm v1.11.0
	go.elastic.co/apm/module/apmgrpc v1.11.0
	go.elastic.co/apm/module/apmhttp v1.11.0
	google.golang.org/grpc v1.17.0
)

replace go.elastic.co/apm => ../..

replace go.elastic.co/apm/module/apmgrpc => ../../module/apmgrpc

replace go.elastic.co/apm/module/apmhttp => ../../module/apmhttp
