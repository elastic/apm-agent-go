module go.elastic.co/apm/module/apmchi

require (
	github.com/go-chi/chi v4.0.2+incompatible
	github.com/stretchr/testify v1.4.0
	go.elastic.co/apm v1.7.1
	go.elastic.co/apm/module/apmhttp v1.7.1
)

replace go.elastic.co/apm => ../..

replace go.elastic.co/apm/module/apmhttp => ../apmhttp

go 1.13
