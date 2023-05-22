module go.elastic.co/apm/module/apmgormv2/v2

require (
	github.com/golang-sql/civil v0.0.0-20220223132316-b832511892a9 // indirect
	github.com/jackc/pgproto3/v2 v2.3.1 // indirect
	github.com/jackc/pgtype v1.12.0 // indirect
	github.com/jackc/pgx/v4 v4.16.1 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/mattn/go-sqlite3 v1.14.16 // indirect
	github.com/stretchr/testify v1.8.0
	go.elastic.co/apm/module/apmsql/v2 v2.4.2
	go.elastic.co/apm/v2 v2.4.2
	gorm.io/driver/mysql v1.0.2
	gorm.io/driver/postgres v1.3.4
	gorm.io/driver/sqlite v1.3.1
	gorm.io/driver/sqlserver v1.3.2
	gorm.io/gorm v1.23.3
)

replace go.elastic.co/apm/v2 => ../..

replace go.elastic.co/apm/module/apmsql/v2 => ../apmsql

go 1.15
