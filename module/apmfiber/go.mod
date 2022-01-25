module go.elastic.co/apm/module/apmfiber/v2

require (
	github.com/gofiber/fiber/v2 v2.18.0
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.7.0
	github.com/valyala/fasthttp v1.29.0
	go.elastic.co/apm/module/apmfasthttp/v2 v2.0.0
	go.elastic.co/apm/module/apmhttp/v2 v2.0.0
	go.elastic.co/apm/v2 v2.0.0
)

replace go.elastic.co/apm/v2 => ../..

replace go.elastic.co/apm/module/apmhttp/v2 => ../apmhttp

replace go.elastic.co/apm/module/apmfasthttp/v2 => ../apmfasthttp

go 1.15

exclude (
	gopkg.in/yaml.v2 v2.2.1
	gopkg.in/yaml.v2 v2.2.2
	gopkg.in/yaml.v2 v2.2.4
	gopkg.in/yaml.v2 v2.2.7
)
