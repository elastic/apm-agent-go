module go.elastic.co/apm/module/apmot/v2

require (
	github.com/opentracing/opentracing-go v1.1.0
	github.com/stretchr/testify v1.6.1
	go.elastic.co/apm/module/apmhttp/v2 v2.3.0
	go.elastic.co/apm/v2 v2.3.0
)

replace go.elastic.co/apm/v2 => ../..

replace go.elastic.co/apm/module/apmhttp/v2 => ../apmhttp

go 1.15
