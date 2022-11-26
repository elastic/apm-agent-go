module go.elastic.co/apm/module/apmgormv2/v2

require (
	github.com/mattn/go-sqlite3 v1.14.16 // indirect
	github.com/stretchr/testify v1.8.0
	go.elastic.co/apm/module/apmsql/v2 v2.2.0
	go.elastic.co/apm/v2 v2.2.0
	golang.org/x/crypto v0.3.0 // indirect
	gorm.io/driver/mysql v1.0.2
	gorm.io/driver/postgres v1.4.5
	gorm.io/driver/sqlite v1.4.3
	gorm.io/driver/sqlserver v1.4.1
	gorm.io/gorm v1.24.2
)

replace go.elastic.co/apm/v2 => ../..

replace go.elastic.co/apm/module/apmsql/v2 => ../apmsql

go 1.15
