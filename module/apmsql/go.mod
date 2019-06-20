module go.elastic.co/apm/module/apmsql

require (
	github.com/go-sql-driver/mysql v1.4.1
	github.com/lib/pq v1.0.0
	github.com/mattn/go-sqlite3 v1.10.0
	github.com/stretchr/testify v1.2.2
	go.elastic.co/apm v1.4.0
	google.golang.org/appengine v1.4.0 // indirect
)

replace go.elastic.co/apm => ../..
