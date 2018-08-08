// +build gofuzz

package apmsql_test

import (
	"strings"

	"github.com/elastic/apm-agent-go/module/apmsql"
)

func Fuzz(data []byte) int {
	sql := string(data)
	sig := apmsql.QuerySignature(sql)
	if sig == "" {
		return -1
	}
	prefixes := [...]string{
		"CALL ",
		"DELETE FROM ",
		"INSERT INTO ",
		"REPLACE INTO ",
		"SELECT FROM ",
		"UPDATE ",
	}
	for _, p := range prefixes {
		if strings.HasPrefix(sig, p) {
			// Give priority to input that is parsed
			// successfully, and doesn't just result
			// in the fallback.
			return 1
		}
	}
	return 0
}
