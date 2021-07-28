module go.elastic.co/apm/module/apmnegroni

go 1.13

require (
	github.com/stretchr/testify v1.6.1
	github.com/urfave/negroni v1.0.0
	go.elastic.co/apm v1.13.0
	go.elastic.co/apm/module/apmhttp v1.13.0
)

replace go.elastic.co/apm => ../..

replace go.elastic.co/apm/module/apmhttp => ../apmhttp
