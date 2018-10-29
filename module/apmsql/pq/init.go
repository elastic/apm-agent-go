package apmpq

import (
	"github.com/lib/pq"

	"go.elastic.co/apm/module/apmsql"
)

func init() {
	apmsql.Register("postgres", &pq.Driver{}, apmsql.WithDSNParser(ParseDSN))
}
