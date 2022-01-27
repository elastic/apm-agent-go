module go.elastic.co/apm/module/apmzap/v2

require (
	github.com/pkg/errors v0.8.1
	github.com/stretchr/testify v1.6.1
	go.elastic.co/apm/v2 v2.0.0
	go.uber.org/atomic v1.3.2 // indirect
	go.uber.org/multierr v1.1.0 // indirect
	go.uber.org/zap v1.9.1
	gopkg.in/yaml.v2 v2.2.2 // indirect
)

replace go.elastic.co/apm/v2 => ../..

go 1.13
