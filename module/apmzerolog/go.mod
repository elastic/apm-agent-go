module go.elastic.co/apm/module/apmzerolog/v2

require (
	github.com/pkg/errors v0.9.1
	github.com/rs/zerolog v1.14.3
	github.com/stretchr/testify v1.8.4
	go.elastic.co/apm/v2 v2.4.2
)

replace go.elastic.co/apm/v2 => ../..

go 1.15
