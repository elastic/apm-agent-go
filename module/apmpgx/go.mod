module go.elastic.co/apm/module/apmpgx/v2

go 1.15

require (
	github.com/jackc/pgx/v4 v4.17.0
	github.com/stretchr/testify v1.8.4
	go.elastic.co/apm/module/apmsql/v2 v2.4.2
	go.elastic.co/apm/v2 v2.4.2
)

replace go.elastic.co/apm/v2 => ../..

replace go.elastic.co/apm/module/apmsql/v2 => ../apmsql
