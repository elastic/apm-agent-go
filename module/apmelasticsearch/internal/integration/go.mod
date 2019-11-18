module go.elastic.co/apm/module/apmelasticsearch/internal/integration

require (
	github.com/fortytw2/leaktest v1.3.0 // indirect
	github.com/mailru/easyjson v0.0.0-20180823135443-60711f1a8329 // indirect
	github.com/olivere/elastic v6.2.16+incompatible
	github.com/stretchr/testify v1.3.0
	go.elastic.co/apm v1.6.0
	go.elastic.co/apm/module/apmelasticsearch v1.6.0
)

replace go.elastic.co/apm => ../../../..

replace go.elastic.co/apm/module/apmelasticsearch => ../..

replace go.elastic.co/apm/module/apmhttp => ../../../apmhttp

go 1.13
