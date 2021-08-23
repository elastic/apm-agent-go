module go.elastic.co/apm/module/apmfasthttp

go 1.13

require (
	github.com/stretchr/testify v1.6.1
	github.com/valyala/bytebufferpool v1.0.0
	github.com/valyala/fasthttp v1.26.0
	go.elastic.co/apm v1.13.1
	go.elastic.co/apm/module/apmhttp v1.13.1
)

replace (
	go.elastic.co/apm => ../..
	go.elastic.co/apm/module/apmhttp => ../apmhttp
)
