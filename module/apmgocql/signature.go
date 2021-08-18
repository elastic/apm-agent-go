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

//go:build go1.9
// +build go1.9

package apmgocql // import "go.elastic.co/apm/module/apmgocql"

import (
	"strings"

	"go.elastic.co/apm/internal/sqlscanner"
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
