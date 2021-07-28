module go.elastic.co/apm/module/apmchi

require (
	github.com/go-chi/chi v1.5.1
	github.com/stretchr/testify v1.4.0
	go.elastic.co/apm v1.13.0
	go.elastic.co/apm/module/apmhttp v1.13.0
)

replace go.elastic.co/apm => ../..

replace go.elastic.co/apm/module/apmhttp => ../apmhttp

go 1.13
