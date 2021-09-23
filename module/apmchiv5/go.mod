module go.elastic.co/apm/module/apmchiv5

require (
	github.com/go-chi/chi/v5 v5.0.2
	github.com/stretchr/testify v1.6.1
	go.elastic.co/apm v1.14.0
	go.elastic.co/apm/module/apmhttp v1.14.0
)

replace go.elastic.co/apm => ../..

replace go.elastic.co/apm/module/apmhttp => ../apmhttp

go 1.14
