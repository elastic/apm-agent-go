module go.elastic.co/apm/module/apmrestful/v2

require (
	github.com/emicklei/go-restful v2.9.6+incompatible
	github.com/json-iterator/go v1.1.5 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.1 // indirect
	github.com/stretchr/testify v1.6.1
	go.elastic.co/apm/module/apmhttp/v2 v2.0.0
	go.elastic.co/apm/v2 v2.0.0
)

replace go.elastic.co/apm/v2 => ../..

replace go.elastic.co/apm/module/apmhttp/v2 => ../apmhttp

go 1.15
