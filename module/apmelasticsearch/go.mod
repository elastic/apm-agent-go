module go.elastic.co/apm/module/apmelasticsearch/v2

require (
	github.com/stretchr/testify v1.6.1
	go.elastic.co/apm/module/apmhttp/v2 v2.1.0
	go.elastic.co/apm/v2 v2.1.0
	golang.org/x/net v0.0.0-20211015210444-4f30a5c0130f
)

replace go.elastic.co/apm/v2 => ../..

replace go.elastic.co/apm/module/apmhttp/v2 => ../apmhttp

go 1.15
