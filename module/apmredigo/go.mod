module go.elastic.co/apm/module/apmredigo/v2

require (
	github.com/gomodule/redigo v1.8.2
	github.com/stretchr/testify v1.8.4
	go.elastic.co/apm/v2 v2.4.2
)

replace go.elastic.co/apm/v2 => ../..

go 1.15
