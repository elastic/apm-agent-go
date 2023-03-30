module apmelasticsearch_integration

require (
	github.com/elastic/go-elasticsearch/v7 v7.5.0
	github.com/fortytw2/leaktest v1.3.0 // indirect
	github.com/mailru/easyjson v0.0.0-20180823135443-60711f1a8329 // indirect
	github.com/olivere/elastic v6.2.16+incompatible
	github.com/stretchr/testify v1.6.1
	go.elastic.co/apm/module/apmelasticsearch/v2 v2.1.0
	go.elastic.co/apm/v2 v2.3.0
)

replace go.elastic.co/apm/v2 => ../../../..

replace go.elastic.co/apm/module/apmelasticsearch/v2 => ../..

replace go.elastic.co/apm/module/apmhttp/v2 => ../../../apmhttp

go 1.13
