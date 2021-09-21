module go.elastic.co/apm/module/apmprometheus

require (
	github.com/pkg/errors v0.8.1
	github.com/prometheus/client_golang v0.9.2
	github.com/prometheus/client_model v0.0.0-20180712105110-5c3871d89910
	github.com/stretchr/testify v1.6.1
	go.elastic.co/apm v1.14.0
	gopkg.in/yaml.v2 v2.2.2 // indirect
)

replace go.elastic.co/apm => ../..

go 1.13
