module go.elastic.co/apm/module/apmfiber/v2

require (
	github.com/gofiber/fiber/v2 v2.18.0
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.7.0
	github.com/valyala/fasthttp v1.34.0
	go.elastic.co/apm/module/apmfasthttp/v2 v2.4.2
	go.elastic.co/apm/module/apmhttp/v2 v2.4.2
	go.elastic.co/apm/v2 v2.4.2
)

replace go.elastic.co/apm/v2 => ../..

replace go.elastic.co/apm/module/apmhttp/v2 => ../apmhttp

replace go.elastic.co/apm/module/apmfasthttp/v2 => ../apmfasthttp

go 1.15
