module go.elastic.co/apm/module/apmpgx/v2

go 1.15

require (
	github.com/jackc/pgx/v5 v5.0.4
	github.com/stretchr/testify v1.8.0
	go.elastic.co/apm/v2 v2.1.0
)

replace go.elastic.co/apm/v2 => ../..
