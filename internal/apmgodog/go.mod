module apmgodog

go 1.15

require (
	github.com/cucumber/godog v0.12.2
	go.elastic.co/apm/module/apmgrpc/v2 v2.0.0
	go.elastic.co/apm/module/apmhttp/v2 v2.0.0
	go.elastic.co/apm/v2 v2.0.0
	go.elastic.co/fastjson v1.1.0
	google.golang.org/grpc v1.36.0
	google.golang.org/grpc/examples v0.0.0-20220124233804-5b3768235a1d
)

replace go.elastic.co/apm/v2 => ../..

replace go.elastic.co/apm/module/apmgrpc/v2 => ../../module/apmgrpc

replace go.elastic.co/apm/module/apmhttp/v2 => ../../module/apmhttp

exclude (
	gopkg.in/yaml.v2 v2.0.0-20170812160011-eb3733d160e7
	gopkg.in/yaml.v2 v2.2.1
	gopkg.in/yaml.v2 v2.2.2
	gopkg.in/yaml.v2 v2.2.4
	gopkg.in/yaml.v2 v2.2.7
)
