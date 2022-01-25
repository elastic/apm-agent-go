module go.elastic.co/apm/module/apmechov4/v2

require (
	github.com/labstack/echo/v4 v4.6.1
	github.com/pkg/errors v0.8.1
	github.com/stretchr/testify v1.6.1
	go.elastic.co/apm/module/apmhttp/v2 v2.0.0
	go.elastic.co/apm/v2 v2.0.0
)

replace go.elastic.co/apm/v2 => ../..

replace go.elastic.co/apm/module/apmhttp/v2 => ../apmhttp

go 1.15

exclude (
	gopkg.in/yaml.v2 v2.2.1
	gopkg.in/yaml.v2 v2.2.2
	gopkg.in/yaml.v2 v2.2.4
	gopkg.in/yaml.v2 v2.2.7
)
