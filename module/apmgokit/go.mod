module go.elastic.co/apm/module/apmgokit

require (
	github.com/go-kit/kit v0.8.0
	github.com/go-logfmt/logfmt v0.4.0 // indirect
	github.com/go-stack/stack v1.8.0 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.0.0
	github.com/stretchr/testify v1.4.0
	go.elastic.co/apm v1.12.0
	go.elastic.co/apm/module/apmgrpc v1.12.0
	go.elastic.co/apm/module/apmhttp v1.12.0
	golang.org/x/net v0.0.0-20200226121028-0de0cce0169b
	google.golang.org/grpc v1.17.0
)

replace go.elastic.co/apm => ../..

replace go.elastic.co/apm/module/apmgrpc => ../apmgrpc

replace go.elastic.co/apm/module/apmhttp => ../apmhttp

go 1.13
