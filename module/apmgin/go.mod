module go.elastic.co/apm/module/apmgin/v2

require (
	github.com/gin-gonic/gin v1.9.0
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.8.1
	go.elastic.co/apm/module/apmhttp/v2 v2.4.1
	go.elastic.co/apm/v2 v2.4.1
)

replace go.elastic.co/apm/v2 => ../..

replace go.elastic.co/apm/module/apmhttp/v2 => ../apmhttp

go 1.15
