module go.elastic.co/apm/module/apmfasthttp

go 1.13

require (
	github.com/valyala/bytebufferpool v1.0.0
	github.com/valyala/fasthttp v1.28.0
	go.elastic.co/apm v1.12.0
	go.elastic.co/apm/module/apmhttp v1.12.0
)

replace (
	go.elastic.co/apm => ../..
	go.elastic.co/apm/module/apmhttp => ../apmhttp
)
