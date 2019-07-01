module go.elastic.co/apm/module/apmgrpc

require (
	github.com/grpc-ecosystem/go-grpc-middleware v1.0.0
	github.com/stretchr/testify v1.3.0
	go.elastic.co/apm v1.4.0
	go.elastic.co/apm/module/apmhttp v1.4.0
	golang.org/x/net v0.0.0-20181220203305-927f97764cc3
	google.golang.org/grpc v1.17.0
)

replace go.elastic.co/apm => ../..

replace go.elastic.co/apm/module/apmhttp => ../apmhttp
