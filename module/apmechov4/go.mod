module go.elastic.co/apm/module/apmechov4

require (
	github.com/labstack/echo/v4 v4.0.0
	github.com/pkg/errors v0.8.0
	github.com/stretchr/testify v1.3.0
	go.elastic.co/apm v1.2.0
	go.elastic.co/apm/module/apmhttp v1.2.0
)

replace go.elastic.co/apm => ../..

replace go.elastic.co/apm/module/apmhttp => ../apmhttp
