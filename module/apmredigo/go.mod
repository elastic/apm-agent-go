module go.elastic.co/apm/module/apmredigo

require (
	github.com/gomodule/redigo v2.0.0+incompatible
	github.com/stretchr/testify v1.3.0
	go.elastic.co/apm v1.5.0
)

replace go.elastic.co/apm => ../..

go 1.13
