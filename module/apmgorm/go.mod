module go.elastic.co/apm/module/apmgorm

require (
	cloud.google.com/go v0.34.0 // indirect
	github.com/denisenkom/go-mssqldb v0.0.0-20181014144952-4e0d7dc8888f // indirect
	github.com/erikstmartin/go-testdb v0.0.0-20160219214506-8d10e4a1bae5 // indirect
	github.com/gofrs/uuid v3.1.0+incompatible // indirect
	github.com/jinzhu/gorm v1.9.2
	github.com/jinzhu/inflection v0.0.0-20180308033659-04140366298a // indirect
	github.com/jinzhu/now v0.0.0-20181116074157-8ec929ed50c3 // indirect
	github.com/pkg/errors v0.8.1
	github.com/stretchr/testify v1.3.0
	go.elastic.co/apm v1.4.0
	go.elastic.co/apm/module/apmsql v1.4.0
	golang.org/x/crypto v0.0.0-20181203042331-505ab145d0a9 // indirect
)

replace go.elastic.co/apm => ../..

replace go.elastic.co/apm/module/apmsql => ../apmsql
