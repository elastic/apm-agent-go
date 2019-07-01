module go.elastic.co/apm/module/apmgorm

require (
	cloud.google.com/go v0.40.0 // indirect
	github.com/jinzhu/gorm v1.9.10
	github.com/pkg/errors v0.8.1
	github.com/stretchr/testify v1.3.0
	go.elastic.co/apm v1.4.0
	go.elastic.co/apm/module/apmsql v1.4.0
	golang.org/x/crypto v0.0.0-20190621222207-cc06ce4a13d4 // indirect
	google.golang.org/appengine v1.6.1 // indirect
)

replace go.elastic.co/apm => ../..

replace go.elastic.co/apm/module/apmsql => ../apmsql
