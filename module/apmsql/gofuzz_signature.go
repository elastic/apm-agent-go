// +build gofuzz

package apmsql

import "strings"

func Fuzz(data []byte) int {
	sql := string(data)
	sig := genericQuerySignature(sql)
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
