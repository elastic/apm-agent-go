package apmpq

import (
	"github.com/lib/pq"

	"github.com/elastic/apm-agent-go/module/apmsql"
)

func init() {
	apmsql.Register("postgres", &pq.Driver{}, apmsql.WithDSNParser(ParseDSN))
}
