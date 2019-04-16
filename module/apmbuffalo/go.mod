module go.elastic.co/apm/module/apmbuffalo

require (
	github.com/gobuffalo/buffalo v0.14.3
	github.com/stretchr/testify v1.3.0
	go.elastic.co/apm v1.3.0
	go.elastic.co/apm/module/apmhttp v1.3.0
)

replace go.elastic.co/apm => ../..

replace go.elastic.co/apm/module/apmhttp => ../apmhttp
