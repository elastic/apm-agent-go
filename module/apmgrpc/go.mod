module go.elastic.co/apm/module/apmgrpc/v2

require (
	github.com/golang/protobuf v1.2.0
	github.com/grpc-ecosystem/go-grpc-middleware v1.0.0
	github.com/stretchr/testify v1.6.1
	go.elastic.co/apm/module/apmhttp/v2 v2.4.1
	go.elastic.co/apm/v2 v2.4.1
	golang.org/x/net v0.0.0-20211015210444-4f30a5c0130f
	google.golang.org/grpc v1.17.0
)

replace go.elastic.co/apm/v2 => ../..

replace go.elastic.co/apm/module/apmhttp/v2 => ../apmhttp

go 1.15
