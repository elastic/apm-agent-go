module go.elastic.co/apm/module/apmechov4

require (
	github.com/labstack/echo/v4 v4.0.0
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.4.0
	go.elastic.co/apm v1.11.0
	go.elastic.co/apm/module/apmhttp v1.11.0
)

replace go.elastic.co/apm => ../..

replace go.elastic.co/apm/module/apmhttp => ../apmhttp

go 1.13
