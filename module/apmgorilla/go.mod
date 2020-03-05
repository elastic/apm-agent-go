module go.elastic.co/apm/module/apmgorilla

require (
	github.com/gorilla/context v1.1.1 // indirect
	github.com/gorilla/mux v1.6.2
	github.com/stretchr/testify v1.4.0
	go.elastic.co/apm v1.7.1
	go.elastic.co/apm/module/apmhttp v1.7.1
)

replace go.elastic.co/apm => ../..

replace go.elastic.co/apm/module/apmhttp => ../apmhttp

go 1.13
