module go.elastic.co/apm/module/apmgin

require (
	github.com/gin-gonic/gin v1.4.0
	github.com/pkg/errors v0.8.1
	github.com/stretchr/testify v1.3.0
	go.elastic.co/apm v1.5.0
	go.elastic.co/apm/module/apmhttp v1.5.0
)

replace go.elastic.co/apm => ../..

replace go.elastic.co/apm/module/apmhttp => ../apmhttp

go 1.13
