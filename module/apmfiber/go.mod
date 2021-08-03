module go.elastic.co/apm/module/apmfiber

require (
	github.com/gofiber/fiber/v2 v2.15.0
	github.com/pkg/errors v0.8.1
	github.com/stretchr/testify v1.6.1
	github.com/valyala/fasthttp v1.26.0
	go.elastic.co/apm v1.13.0
	go.elastic.co/apm/module/apmfasthttp v1.13.0
	go.elastic.co/apm/module/apmhttp v1.13.0
)

replace go.elastic.co/apm => ../..

replace go.elastic.co/apm/module/apmhttp => ../apmhttp

replace go.elastic.co/apm/module/apmfasthttp => ../apmfasthttp

go 1.16
