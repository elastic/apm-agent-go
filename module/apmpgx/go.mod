module go.elastic.co/apm/module/apmpgx/v2

go 1.18

require (
	github.com/jackc/pgx/v4 v4.17.0
	github.com/stretchr/testify v1.8.0
	go.elastic.co/apm/module/apmsql/v2 v2.0.0
	go.elastic.co/apm/v2 v2.1.0
)

replace go.elastic.co/apm/v2 => ../..
