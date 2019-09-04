module go.elastic.co/apm/module/apmgorm

require (
	cloud.google.com/go v0.40.0 // indirect
	github.com/jinzhu/gorm v1.9.10
	github.com/pkg/errors v0.8.1
	github.com/stretchr/testify v1.3.0
	go.elastic.co/apm v1.5.0
	go.elastic.co/apm/module/apmsql v1.5.0
	golang.org/x/crypto v0.0.0-20190701094942-4def268fd1a4 // indirect
	google.golang.org/appengine v1.6.1 // indirect
)

replace go.elastic.co/apm => ../..

replace go.elastic.co/apm/module/apmsql => ../apmsql

go 1.13
