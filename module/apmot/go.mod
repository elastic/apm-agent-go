module go.elastic.co/apm/module/apmot

require (
	github.com/opentracing/opentracing-go v1.1.0
	github.com/stretchr/testify v1.3.0
	go.elastic.co/apm v1.4.0
	go.elastic.co/apm/module/apmhttp v1.4.0
	golang.org/x/net v0.0.0-20181220203305-927f97764cc3 // indirect
)

replace go.elastic.co/apm => ../..

replace go.elastic.co/apm/module/apmhttp => ../apmhttp
