module go.elastic.co/apm/module/apmzap/v2

require (
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.8.4
	go.elastic.co/apm/v2 v2.4.2
	go.uber.org/atomic v1.3.2 // indirect
	go.uber.org/multierr v1.1.0 // indirect
	go.uber.org/zap v1.9.1
)

replace go.elastic.co/apm/v2 => ../..

go 1.15
