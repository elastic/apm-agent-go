module go.elastic.co/apm/module/apmelasticsearch/v2

require (
	github.com/stretchr/testify v1.6.1
	go.elastic.co/apm/module/apmhttp/v2 v2.0.0
	go.elastic.co/apm/v2 v2.0.0
	golang.org/x/net v0.0.0-20210805182204-aaa1db679c0d
)

replace go.elastic.co/apm/v2 => ../..

replace go.elastic.co/apm/module/apmhttp/v2 => ../apmhttp

go 1.15
