module go.elastic.co/apm/module/apmgokit/v2

require (
	github.com/go-kit/kit v0.8.0
	github.com/go-logfmt/logfmt v0.4.0 // indirect
	github.com/go-stack/stack v1.8.0 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.0.0
	github.com/stretchr/testify v1.6.1
	go.elastic.co/apm/module/apmgrpc/v2 v2.3.0
	go.elastic.co/apm/module/apmhttp/v2 v2.3.0
	go.elastic.co/apm/v2 v2.3.0
	golang.org/x/net v0.0.0-20211015210444-4f30a5c0130f
	google.golang.org/grpc v1.17.0
)

replace go.elastic.co/apm/v2 => ../..

replace go.elastic.co/apm/module/apmgrpc/v2 => ../apmgrpc

replace go.elastic.co/apm/module/apmhttp/v2 => ../apmhttp

go 1.15
