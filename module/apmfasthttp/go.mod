module go.elastic.co/apm/module/apmfasthttp/v2

go 1.15

require (
	github.com/stretchr/testify v1.6.1
	github.com/valyala/bytebufferpool v1.0.0
	github.com/valyala/fasthttp v1.33.0
	go.elastic.co/apm/module/apmhttp/v2 v2.0.0
	go.elastic.co/apm/v2 v2.0.0
)

replace (
	go.elastic.co/apm/module/apmhttp/v2 => ../apmhttp
	go.elastic.co/apm/v2 => ../..
)
