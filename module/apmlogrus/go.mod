module go.elastic.co/apm/module/apmlogrus

require (
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.2.0
	github.com/stretchr/testify v1.4.0
	go.elastic.co/apm v1.11.0
)

replace go.elastic.co/apm => ../..

go 1.13
