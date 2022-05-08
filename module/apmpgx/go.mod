module go.elastic.co/apm/module/apmpgx/v2

go 1.18

require (
	github.com/jackc/pgx/v4 v4.16.1
	go.elastic.co/apm/module/apmsql/v2 v2.0.0
	go.elastic.co/apm/v2 v2.0.0
)

replace go.elastic.co/apm/v2 => ../..

replace go.elastic.co/apm/module/apmsql/v2 => ../apmsql
