module go.elastic.co/apm/module/apmzerolog

require (
	github.com/pkg/errors v0.9.1
	github.com/rs/zerolog v1.14.3
	github.com/stretchr/testify v1.4.0
	go.elastic.co/apm v1.8.0
)

replace go.elastic.co/apm => ../..

go 1.13
