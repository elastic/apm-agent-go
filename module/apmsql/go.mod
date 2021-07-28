module go.elastic.co/apm/module/apmsql

require (
	github.com/go-sql-driver/mysql v1.5.0
	github.com/jackc/pgx/v4 v4.9.0
	github.com/lib/pq v1.3.0
	github.com/mattn/go-sqlite3 v1.10.0
	github.com/stretchr/testify v1.5.1
	go.elastic.co/apm v1.13.0
)

replace go.elastic.co/apm => ../..

go 1.13
