module go.elastic.co/apm/module/apmbuffalo

require (
	github.com/gobuffalo/buffalo v0.14.3
	github.com/gobuffalo/envy v1.6.15
	github.com/gobuffalo/mw-contenttype v0.0.0-20190224202710-36c73cc938f3
	github.com/gobuffalo/x v0.0.0-20190224155809-6bb134105960
	github.com/pkg/errors v0.8.1
	github.com/rs/cors v1.6.0
	github.com/stretchr/testify v1.3.0
	go.elastic.co/apm v1.3.0
	go.elastic.co/apm/module/apmhttp v1.3.0
)

replace go.elastic.co/apm => ../..

replace go.elastic.co/apm/module/apmhttp => ../apmhttp
