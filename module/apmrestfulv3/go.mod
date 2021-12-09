module go.elastic.co/apm/module/apmrestfulv3/v2

require (
	github.com/emicklei/go-restful/v3 v3.5.1
	github.com/json-iterator/go v1.1.11 // indirect
	github.com/stretchr/testify v1.6.1
	go.elastic.co/apm/module/apmhttp/v2 v2.0.0
	go.elastic.co/apm/v2 v2.0.0
)

replace go.elastic.co/apm/v2 => ../..

replace go.elastic.co/apm/module/apmhttp/v2 => ../apmhttp

go 1.13
