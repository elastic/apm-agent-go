module go.elastic.co/apm/module/apmfasthttp

go 1.13

require (
	github.com/stretchr/testify v1.6.1
	github.com/valyala/bytebufferpool v1.0.0
	github.com/valyala/fasthttp v1.32.1-0.20220122035437-436977654aa1
	go.elastic.co/apm v1.15.0
	go.elastic.co/apm/module/apmhttp v1.15.0
)

replace (
	go.elastic.co/apm => ../..
	go.elastic.co/apm/module/apmhttp => ../apmhttp
)
