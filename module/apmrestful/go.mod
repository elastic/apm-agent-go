module go.elastic.co/apm/module/apmrestful

require (
	github.com/emicklei/go-restful v2.8.0+incompatible
	github.com/json-iterator/go v1.1.5 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.1 // indirect
	github.com/stretchr/testify v1.2.2
	go.elastic.co/apm v1.4.0
	go.elastic.co/apm/module/apmhttp v1.4.0
	golang.org/x/net v0.0.0-20181220203305-927f97764cc3 // indirect
)

replace go.elastic.co/apm => ../..

replace go.elastic.co/apm/module/apmhttp => ../apmhttp
