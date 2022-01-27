module go.elastic.co/apm/module/apmhttp/v2

require (
	github.com/pkg/errors v0.8.1
	github.com/stretchr/testify v1.6.1
	go.elastic.co/apm/v2 v2.0.0
	golang.org/x/net v0.0.0-20200226121028-0de0cce0169b
	golang.org/x/text v0.3.2 // indirect
	gopkg.in/yaml.v2 v2.2.2 // indirect
)

replace go.elastic.co/apm/v2 => ../..

go 1.13
