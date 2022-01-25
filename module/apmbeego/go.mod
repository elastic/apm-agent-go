module go.elastic.co/apm/module/apmbeego/v2

require (
	github.com/astaxie/beego v1.11.1
	github.com/stretchr/testify v1.6.1
	go.elastic.co/apm/module/apmhttp/v2 v2.0.0
	go.elastic.co/apm/module/apmsql/v2 v2.0.0
	go.elastic.co/apm/v2 v2.0.0
	gopkg.in/yaml.v2 v2.4.0 // indirect
)

replace go.elastic.co/apm/v2 => ../..

replace go.elastic.co/apm/module/apmhttp/v2 => ../apmhttp

replace go.elastic.co/apm/module/apmsql/v2 => ../apmsql

go 1.15

exclude (
	gopkg.in/yaml.v2 v2.2.1
	gopkg.in/yaml.v2 v2.2.2
	gopkg.in/yaml.v2 v2.2.4
	gopkg.in/yaml.v2 v2.2.7
)
