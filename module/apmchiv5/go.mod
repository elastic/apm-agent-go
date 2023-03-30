module go.elastic.co/apm/module/apmchiv5/v2

require (
	github.com/go-chi/chi/v5 v5.0.2
	github.com/stretchr/testify v1.6.1
	go.elastic.co/apm/module/apmhttp/v2 v2.3.0
	go.elastic.co/apm/v2 v2.3.0
)

replace go.elastic.co/apm/v2 => ../..

replace go.elastic.co/apm/module/apmhttp/v2 => ../apmhttp

go 1.15
