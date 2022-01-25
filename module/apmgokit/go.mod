module go.elastic.co/apm/module/apmgokit/v2

require (
	github.com/go-kit/kit v0.8.0
	github.com/go-logfmt/logfmt v0.4.0 // indirect
	github.com/go-stack/stack v1.8.0 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.0.0
	github.com/stretchr/testify v1.6.1
	go.elastic.co/apm/module/apmgrpc/v2 v2.0.0
	go.elastic.co/apm/module/apmhttp/v2 v2.0.0
	go.elastic.co/apm/v2 v2.0.0
	golang.org/x/net v0.0.0-20210805182204-aaa1db679c0d
	google.golang.org/grpc v1.17.0
)

replace go.elastic.co/apm/v2 => ../..

replace go.elastic.co/apm/module/apmgrpc/v2 => ../apmgrpc

replace go.elastic.co/apm/module/apmhttp/v2 => ../apmhttp

go 1.15

exclude (
	gopkg.in/yaml.v2 v2.2.1
	gopkg.in/yaml.v2 v2.2.2
	gopkg.in/yaml.v2 v2.2.4
	gopkg.in/yaml.v2 v2.2.7
)
