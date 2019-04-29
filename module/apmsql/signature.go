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

package apmsql

import (
	"strings"

	"go.elastic.co/apm/internal/sqlscanner"
)

// QuerySignature returns the "signature" for a query:
// a high level description of the operation.
//
// For DDL statements (CREATE, DROP, ALTER, etc.), we we only
// report the first keyword, on the grounds that these statements
// are not expected to be common within the hot code paths of
// an application. For SELECT, INSERT, and UPDATE, and DELETE,
// we attempt to extract the first table name. If we are unable
// to identify the table name, we simply omit it.
func QuerySignature(query string) string {
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
	case sqlscanner.CALL:
		if !scanUntil(sqlscanner.IDENT) {
			break
		}
		return "CALL " + s.Text()

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

	case sqlscanner.INSERT, sqlscanner.REPLACE:
		action := s.Text()
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
		return action + " INTO " + tableName

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

	case sqlscanner.UPDATE:
		// Scan for the table name. Some dialects allow
		// option keywords before the table name.
		var havePeriod, haveFirstPeriod bool
		if !scanToken(sqlscanner.IDENT) {
			return "UPDATE"
		}
		tableName := s.Text()
		for s.Scan() {
			switch tok := s.Token(); tok {
			case sqlscanner.IDENT:
				if havePeriod {
					tableName += s.Text()
					havePeriod = false
				}
				if !haveFirstPeriod {
					tableName = s.Text()
				} else {
					// Two adjacent identifiers found
					// after the first period. Ignore
					// the secondary ones, in case they
					// are unknown keywords.
				}
			case sqlscanner.PERIOD:
				haveFirstPeriod = true
				havePeriod = true
				tableName += "."
			default:
				return "UPDATE " + tableName
			}
		}
	}

	// If all else fails, just return the first token of the query.
	fields := strings.Fields(query)
	if len(fields) == 0 {
		return ""
	}
	return strings.ToUpper(fields[0])
}
