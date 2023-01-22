module go.elastic.co/apm/module/apmpgxv5/v2

go 1.18

require (
	github.com/jackc/pgx/v5 v5.0.4
	github.com/stretchr/testify v1.8.0
	go.elastic.co/apm/module/apmsql/v2 v2.2.0
	go.elastic.co/apm/v2 v2.2.0
)

replace go.elastic.co/apm/v2 => ../..

replace go.elastic.co/apm/module/apmsql/v2 => ../apmsql
