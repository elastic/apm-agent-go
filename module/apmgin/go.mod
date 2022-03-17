module go.elastic.co/apm/module/apmgin/v2

require (
	github.com/gin-gonic/gin v1.7.2
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.6.1
	go.elastic.co/apm/module/apmhttp/v2 v2.0.0
	go.elastic.co/apm/v2 v2.0.0
)

replace go.elastic.co/apm/v2 => ../..

replace go.elastic.co/apm/module/apmhttp/v2 => ../apmhttp

go 1.15
