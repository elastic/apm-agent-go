module go.elastic.co/apm/module/apmgokit

require (
	github.com/go-kit/kit v0.8.0
	github.com/go-logfmt/logfmt v0.4.0 // indirect
	github.com/go-stack/stack v1.8.0 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.0.0
	github.com/stretchr/testify v1.2.2
	go.elastic.co/apm v1.2.0
	go.elastic.co/apm/module/apmgrpc v1.2.0
	go.elastic.co/apm/module/apmhttp v1.2.0
	golang.org/x/net v0.0.0-20181220203305-927f97764cc3
	google.golang.org/grpc v1.17.0
)

replace go.elastic.co/apm => ../..

replace go.elastic.co/apm/module/apmgrpc => ../apmgrpc

replace go.elastic.co/apm/module/apmhttp => ../apmhttp
