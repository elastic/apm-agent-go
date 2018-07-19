package sqlscanner

import "fmt"

// Token represents a SQL token.
type Token int

func (t Token) String() string {
	s := tokenStrings[t]
	if s == "" {
		s = fmt.Sprintf("Token(%d)", t)
	}
	return s
}

// The list of SQL tokens.
const (
	OTHER Token = iota // anything we don't specifically care about
	eof
	COMMENT

	IDENT  // includes unhandled keywords
	NUMBER // 123, 123.45, 123e+45
	STRING // 'foo'

	PERIOD // .
	COMMA  // ,
	LPAREN // (
	RPAREN // )

	CALL
	DELETE
	FROM
	INSERT
	INTO
	OR
	REPLACE
	SELECT
	SET
	TABLE
	TRUNCATE // Cassandra/CQL-specific
	UPDATE
)

var tokenStrings = [...]string{
	OTHER:   "OTHER",
	eof:     "EOF",
	COMMENT: "COMMENT",

	IDENT:  "IDENT",
	NUMBER: "NUMBER",
	STRING: "STRING",

	PERIOD: ".",
	COMMA:  ",",
	LPAREN: "(",
	RPAREN: ")",

	CALL:     "CALL",
	DELETE:   "DELETE",
	FROM:     "FROM",
	INSERT:   "INSERT",
	INTO:     "INTO",
	OR:       "OR",
	REPLACE:  "REPLACE",
	SELECT:   "SELECT",
	SET:      "SET",
	TABLE:    "TABLE",
	TRUNCATE: "TRUNCATE",
	UPDATE:   "UPDATE",
}

// keywords contains keyword tokens, indexed
// at the position of keyword length.
var keywords = [...][]Token{
	2: []Token{OR},
	3: []Token{SET},
	4: []Token{CALL, FROM, INTO},
	5: []Token{TABLE},
	6: []Token{DELETE, INSERT, SELECT, UPDATE},
	7: []Token{REPLACE},
	8: []Token{TRUNCATE},
}
