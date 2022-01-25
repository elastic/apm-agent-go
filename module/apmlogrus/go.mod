module go.elastic.co/apm/module/apmlogrus/v2

require (
	github.com/elastic/go-licenser v0.4.0 // indirect
	github.com/elastic/go-sysinfo v1.7.1 // indirect
	github.com/google/go-cmp v0.5.7 // indirect
	github.com/jcchavezs/porto v0.4.0 // indirect
	github.com/pkg/errors v0.8.1
	github.com/sirupsen/logrus v1.2.0
	github.com/stretchr/testify v1.6.1
	go.elastic.co/apm/v2 v2.0.0
	golang.org/x/crypto v0.0.0-20191206172530-e9b2fee46413 // indirect
	golang.org/x/tools v0.1.8 // indirect
)

replace go.elastic.co/apm/v2 => ../..

go 1.15
