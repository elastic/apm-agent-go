module go.elastic.co/apm/module/apmsql/v2

require (
	github.com/denisenkom/go-mssqldb v0.12.3
	github.com/go-sql-driver/mysql v1.5.0
	github.com/jackc/pgx/v4 v4.9.0
	github.com/lib/pq v1.3.0
	github.com/mattn/go-sqlite3 v1.10.0
	github.com/stretchr/testify v1.7.0
	go.elastic.co/apm/v2 v2.3.0
	golang.org/x/sys v0.1.0 // indirect
)

replace go.elastic.co/apm/v2 => ../..

go 1.15
