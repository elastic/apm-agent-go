module go.elastic.co/apm/module/apmelasticsearch

require (
	github.com/stretchr/testify v1.2.2
	go.elastic.co/apm v1.4.0
	go.elastic.co/apm/module/apmhttp v1.4.0
	golang.org/x/net v0.0.0-20181213202711-891ebc4b82d6
)

replace go.elastic.co/apm => ../..

replace go.elastic.co/apm/module/apmhttp => ../apmhttp
