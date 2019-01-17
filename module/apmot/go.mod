module go.elastic.co/apm/module/apmot

require (
	github.com/opentracing/opentracing-go v1.0.2
	github.com/stretchr/testify v1.2.2
	go.elastic.co/apm v1.2.0
	go.elastic.co/apm/module/apmhttp v1.2.0
	golang.org/x/net v0.0.0-20181220203305-927f97764cc3 // indirect
)

replace go.elastic.co/apm => ../..

replace go.elastic.co/apm/module/apmhttp => ../apmhttp
