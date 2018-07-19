package apmgocql

import (
	"strings"

	"github.com/elastic/apm-agent-go/internal/sqlscanner"
)

// querySignature returns the "signature" for a Cassandra query:
// a high level description of the operation.
//
// For DDL statements (CREATE, DROP, ALTER, etc.), we we only
// report the first keyword, on the grounds that these statements
// are not expected to be common within the hot code paths of
// an application. For DML statements, we extract and include
// the table name in the signature.
func querySignature(query string) string {
	s := sqlscanner.NewScanner(query)
	for s.Scan() {
		if s.Token() != sqlscanner.COMMENT {
			break
		}
	}

	scanUntil := func(until sqlscanner.Token) bool {
		for s.Scan() {
			if s.Token() == until {
				return true
			}
		}
		return false
	}
	scanToken := func(tok sqlscanner.Token) bool {
		for s.Scan() {
			switch s.Token() {
			case tok:
				return true
			case sqlscanner.COMMENT:
			default:
				return false
			}
		}
		return false
	}

	switch s.Token() {
	case sqlscanner.DELETE:
		if !scanUntil(sqlscanner.FROM) {
			break
		}
		if !scanToken(sqlscanner.IDENT) {
			break
		}
		tableName := s.Text()
		for scanToken(sqlscanner.PERIOD) && scanToken(sqlscanner.IDENT) {
			tableName += "." + s.Text()
		}
		return "DELETE FROM " + tableName

	case sqlscanner.INSERT:
		if !scanUntil(sqlscanner.INTO) {
			break
		}
		if !scanToken(sqlscanner.IDENT) {
			break
		}
		tableName := s.Text()
		for scanToken(sqlscanner.PERIOD) && scanToken(sqlscanner.IDENT) {
			tableName += "." + s.Text()
		}
		return "INSERT INTO " + tableName

	case sqlscanner.SELECT:
		var level int
	scanLoop:
		for s.Scan() {
			switch tok := s.Token(); tok {
			case sqlscanner.LPAREN:
				level++
			case sqlscanner.RPAREN:
				level--
			case sqlscanner.FROM:
				if level != 0 {
					continue scanLoop
				}
				if !scanToken(sqlscanner.IDENT) {
					break scanLoop
				}
				tableName := s.Text()
				for scanToken(sqlscanner.PERIOD) && scanToken(sqlscanner.IDENT) {
					tableName += "." + s.Text()
				}
				return "SELECT FROM " + tableName
			}
		}

	case sqlscanner.TRUNCATE:
		if !scanUntil(sqlscanner.IDENT) {
			break
		}
		tableName := s.Text()
		for scanToken(sqlscanner.PERIOD) && scanToken(sqlscanner.IDENT) {
			tableName += "." + s.Text()
		}
		return "TRUNCATE " + tableName

	case sqlscanner.UPDATE:
		if !scanToken(sqlscanner.IDENT) {
			break
		}
		tableName := s.Text()
		for scanToken(sqlscanner.PERIOD) && scanToken(sqlscanner.IDENT) {
			tableName += "." + s.Text()
		}
		return "UPDATE " + tableName
	}

	// If all else fails, just return the first token of the query.
	fields := strings.Fields(query)
	if len(fields) == 0 {
		return ""
	}
	return strings.ToUpper(fields[0])
}
