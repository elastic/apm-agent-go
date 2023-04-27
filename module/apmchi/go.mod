module go.elastic.co/apm/module/apmchi/v2

require (
	github.com/go-chi/chi v1.5.1
	github.com/stretchr/testify v1.6.1
	go.elastic.co/apm/module/apmhttp/v2 v2.4.1
	go.elastic.co/apm/v2 v2.4.1
)

replace go.elastic.co/apm/v2 => ../..

replace go.elastic.co/apm/module/apmhttp/v2 => ../apmhttp

go 1.15
