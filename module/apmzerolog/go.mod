module go.elastic.co/apm/module/apmzerolog/v2

require (
	github.com/pkg/errors v0.8.1
	github.com/rs/zerolog v1.14.3
	github.com/stretchr/testify v1.6.1
	go.elastic.co/apm/v2 v2.0.0
	gopkg.in/yaml.v2 v2.2.2 // indirect
)

replace go.elastic.co/apm/v2 => ../..

go 1.13
