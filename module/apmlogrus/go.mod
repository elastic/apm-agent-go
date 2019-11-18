module go.elastic.co/apm/module/apmlogrus

require (
	github.com/pkg/errors v0.8.1
	github.com/sirupsen/logrus v1.2.0
	github.com/stretchr/testify v1.3.0
	go.elastic.co/apm v1.6.0
	golang.org/x/crypto v0.0.0-20190701094942-4def268fd1a4 // indirect
)

replace go.elastic.co/apm => ../..

go 1.13
