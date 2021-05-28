module go.elastic.co/apm/module/apmgormv2

require (
	github.com/stretchr/testify v1.6.1
	go.elastic.co/apm v1.12.0
	go.elastic.co/apm/module/apmsql v1.12.0
	gorm.io/driver/mysql v1.0.2
	gorm.io/driver/postgres v1.0.2
	gorm.io/driver/sqlite v1.1.4-0.20200928065301-698e250a3b0d
	gorm.io/gorm v1.20.2
)

replace go.elastic.co/apm => ../..

replace go.elastic.co/apm/module/apmsql => ../apmsql

go 1.13
