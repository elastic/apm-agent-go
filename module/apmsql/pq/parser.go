package apmpq

import (
	"go.elastic.co/apm/internal/pgutils"
	"go.elastic.co/apm/module/apmsql"
)

// ParseDSN is proxy to pgutils.ParseDSN to maintain api compatibility
func ParseDSN(name string) apmsql.DSNInfo  {
	return pgutils.ParseDSN(name)
}
