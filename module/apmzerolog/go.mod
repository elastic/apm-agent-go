module go.elastic.co/apm/module/apmzerolog/v2

require (
	github.com/elastic/go-licenser v0.4.0 // indirect
	github.com/elastic/go-sysinfo v1.7.1 // indirect
	github.com/google/go-cmp v0.5.7 // indirect
	github.com/jcchavezs/porto v0.4.0 // indirect
	github.com/pkg/errors v0.8.1
	github.com/rs/zerolog v1.14.3
	github.com/stretchr/testify v1.6.1
	go.elastic.co/apm/v2 v2.0.0
)

replace go.elastic.co/apm/v2 => ../..

go 1.15

exclude (
	gopkg.in/yaml.v2 v2.2.1
	gopkg.in/yaml.v2 v2.2.2
	gopkg.in/yaml.v2 v2.2.4
	gopkg.in/yaml.v2 v2.2.7
)
