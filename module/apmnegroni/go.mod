module go.elastic.co/apm/module/apmnegroni/v2

go 1.15

require (
	github.com/stretchr/testify v1.6.1
	github.com/urfave/negroni v1.0.0
	go.elastic.co/apm/module/apmhttp/v2 v2.3.0
	go.elastic.co/apm/v2 v2.3.0
)

replace go.elastic.co/apm/v2 => ../..

replace go.elastic.co/apm/module/apmhttp/v2 => ../apmhttp
