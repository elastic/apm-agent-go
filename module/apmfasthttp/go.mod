module go.elastic.co/apm/module/apmfasthttp

go 1.13

require (
	github.com/savsgio/gotils v0.0.0-20210316171653-c54912823645
	github.com/valyala/bytebufferpool v1.0.0
	github.com/valyala/fasthttp v1.23.0
	go.elastic.co/apm v1.11.0
	go.elastic.co/apm/module/apmhttp v1.11.0
)

replace (
	go.elastic.co/apm => ../..
	go.elastic.co/apm/module/apmhttp => ../apmhttp
)
