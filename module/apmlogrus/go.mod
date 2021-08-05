module go.elastic.co/apm/module/apmlogrus

require (
	github.com/pkg/errors v0.8.1
	github.com/sirupsen/logrus v1.2.0
	github.com/stretchr/testify v1.6.1
	go.elastic.co/apm v1.13.1
	golang.org/x/crypto v0.0.0-20191206172530-e9b2fee46413 // indirect
	gopkg.in/yaml.v2 v2.2.2 // indirect
)

replace go.elastic.co/apm => ../..

go 1.13
