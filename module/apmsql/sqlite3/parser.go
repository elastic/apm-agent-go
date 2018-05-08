package apmsqlite3

import (
	"strings"

	"github.com/elastic/apm-agent-go/module/apmsql"
)

// ParseDSN parses the sqlite3 datasource name.
func ParseDSN(name string) apmsql.DSNInfo {
	if pos := strings.IndexRune(name, '?'); pos >= 0 {
		name = name[:pos]
	}
	return apmsql.DSNInfo{
		Database: name,
	}
}
