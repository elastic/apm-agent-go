module go.elastic.co/apm/module/apmnegroni

go 1.13

require (
	github.com/stretchr/testify v1.3.0
	github.com/urfave/negroni v1.0.0
	go.elastic.co/apm v1.5.0
	go.elastic.co/apm/module/apmhttp v1.5.0
)

replace go.elastic.co/apm => ../..

replace go.elastic.co/apm/module/apmhttp => ../apmhttp
