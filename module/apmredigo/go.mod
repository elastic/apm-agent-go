module go.elastic.co/apm/module/apmredigo

require (
	github.com/gomodule/redigo v1.8.2
	github.com/stretchr/testify v1.5.1
	go.elastic.co/apm v1.12.0
)

replace go.elastic.co/apm => ../..

go 1.13
