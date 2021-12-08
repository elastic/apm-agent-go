module go.elastic.co/apm/module/apmzerolog

require (
	github.com/pkg/errors v0.8.1
	github.com/rs/zerolog v1.14.3
	github.com/stretchr/testify v1.6.1
	go.elastic.co/apm v1.15.0
	gopkg.in/yaml.v2 v2.2.2 // indirect
)

replace go.elastic.co/apm => ../..

go 1.13
