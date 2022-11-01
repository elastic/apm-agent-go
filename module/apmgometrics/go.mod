module go.elastic.co/apm/module/apmgometrics/v2

require (
	github.com/rcrowley/go-metrics v0.0.0-20181016184325-3113b8401b8a
	github.com/stretchr/testify v1.6.1
	go.elastic.co/apm/v2 v2.2.0
)

replace go.elastic.co/apm/v2 => ../..

go 1.15
