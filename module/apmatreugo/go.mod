module go.elastic.co/apm/module/apmatreugo

go 1.13

require (
	github.com/savsgio/atreugo/v11 v11.7.2
	github.com/valyala/fasthttp v1.26.0
	go.elastic.co/apm v1.12.0
	go.elastic.co/apm/module/apmfasthttp v1.12.0
)

replace (
	go.elastic.co/apm => ../..
	go.elastic.co/apm/module/apmfasthttp => ../apmfasthttp
)
