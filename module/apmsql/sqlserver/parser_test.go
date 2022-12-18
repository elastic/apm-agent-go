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

package apmsqlserver_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.elastic.co/apm/module/apmsql/v2"
	apmsqlserver "go.elastic.co/apm/module/apmsql/v2/sqlserver"
)

func TestParseDSN(t *testing.T) {
	info := apmsqlserver.ParseDSN("sqlserver://user:hunter2@localhost:1433?database=dbname")
	assert.Equal(t, apmsql.DSNInfo{
		Address:  "localhost",
		Port:     1433,
		Database: "dbname",
		User:     "user",
	}, info)
}

func TestParseDSNAddr(t *testing.T) {
	test := func(dsn, addr string, port int) {
		parsed := apmsqlserver.ParseDSN(dsn)
		assert.Equal(t, addr, parsed.Address)
		assert.Equal(t, port, parsed.Port)
	}

	test("sqlserver://user:pass@localhost:9930", "localhost", 9930)
	test("sqlserver://user:pass@localhost", "localhost", 0)
	test("server=(local)", "localhost", 0)
	test("server=.", "localhost", 0)
}

func TestParseDSNError(t *testing.T) {
	info := apmsqlserver.ParseDSN("nonsense")
	assert.Equal(t, apmsql.DSNInfo{Address: "localhost"}, info)
}
