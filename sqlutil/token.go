// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package sqlutil // import "go.elastic.co/apm/v2/sqlutil"

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

	AS
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

	PERIOD: "PERIOD",
	COMMA:  "COMMA",
	LPAREN: "LPAREN",
	RPAREN: "RPAREN",

	AS:       "AS",
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
	2: []Token{AS, OR},
	3: []Token{SET},
	4: []Token{CALL, FROM, INTO},
	5: []Token{TABLE},
	6: []Token{DELETE, INSERT, SELECT, UPDATE},
	7: []Token{REPLACE},
	8: []Token{TRUNCATE},
}
