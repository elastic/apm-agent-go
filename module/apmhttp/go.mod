module go.elastic.co/apm/module/apmhttp/v2

require (
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.6.1
	go.elastic.co/apm/v2 v2.4.0
	golang.org/x/net v0.0.0-20211015210444-4f30a5c0130f
)

replace go.elastic.co/apm/v2 => ../..

go 1.15
