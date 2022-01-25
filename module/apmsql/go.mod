module go.elastic.co/apm/module/apmsql/v2

require (
	github.com/elastic/go-licenser v0.4.0 // indirect
	github.com/elastic/go-sysinfo v1.7.1 // indirect
	github.com/go-sql-driver/mysql v1.5.0
	github.com/google/go-cmp v0.5.7 // indirect
	github.com/jackc/pgx/v4 v4.9.0
	github.com/jcchavezs/porto v0.4.0 // indirect
	github.com/lib/pq v1.3.0
	github.com/mattn/go-sqlite3 v1.10.0
	github.com/stretchr/testify v1.6.1
	go.elastic.co/apm/v2 v2.0.0
)

replace go.elastic.co/apm/v2 => ../..

go 1.15

exclude (
	gopkg.in/yaml.v2 v2.2.1
	gopkg.in/yaml.v2 v2.2.2
	gopkg.in/yaml.v2 v2.2.4
	gopkg.in/yaml.v2 v2.2.7
)
