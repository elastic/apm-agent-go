module go.elastic.co/apm/module/apmhttprouter

require (
	github.com/julienschmidt/httprouter v1.2.0
	github.com/stretchr/testify v1.2.2
	go.elastic.co/apm v1.1.1
	go.elastic.co/apm/module/apmhttp v1.1.1
	golang.org/x/net v0.0.0-20181220203305-927f97764cc3 // indirect
)

replace go.elastic.co/apm => ../../

replace go.elastic.co/apm/module/apmhttp => ../apmhttp
