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

package apmpq_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.elastic.co/apm/module/apmsql"
	apmpq "go.elastic.co/apm/module/apmsql/pq"
)

func patchEnv(k, v string) func() {
	old, reset := os.LookupEnv(k)
	os.Setenv(k, v)
	if reset {
		return func() { os.Setenv(k, old) }
	} else {
		return func() { os.Unsetenv(k) }
	}
}

func TestParseDSNURL(t *testing.T) {
	for _, k := range []string{"PGDATABASE", "PGUSER", "PGHOST", "PGPORT"} {
		unpatch := patchEnv(k, "")
		defer unpatch()
	}

	test := func(url, addr string, port int) {
		info := apmpq.ParseDSN(url)
		assert.Equal(t, apmsql.DSNInfo{
			Address:  addr,
			Port:     port,
			Database: "dbinst",
			User:     "user",
		}, info)
	}
	test("postgresql://user:pass@localhost/dbinst", "localhost", 5432)
	test("postgresql://user:pass@localhost:5432/dbinst", "localhost", 5432)
	test("postgresql://user:pass@localhost:5433/dbinst", "localhost", 5433)
	test("postgresql://user:pass@127.0.0.1/dbinst", "127.0.0.1", 5432)
	test("postgresql://user:pass@[::1]:1234/dbinst", "::1", 1234)
	test("postgresql://user:pass@[::1]/dbinst", "::1", 5432)
	test("postgresql://user:pass@::1/dbinst", "::1", 5432)
}

func TestParseDSNConnectionString(t *testing.T) {
	for _, k := range []string{"PGDATABASE", "PGUSER", "PGHOST", "PGPORT"} {
		unpatch := patchEnv(k, "")
		defer unpatch()
	}
	info := apmpq.ParseDSN("dbname=foo\\ bar user='baz'")
	assert.Equal(t, apmsql.DSNInfo{
		Address:  "localhost",
		Port:     5432,
		Database: "foo bar",
		User:     "baz",
	}, info)
}

func TestParseDSNEnv(t *testing.T) {
	for _, kv := range [][]string{
		{"PGDATABASE", "dbinst"}, {"PGUSER", "bob"}, {"PGHOST", "postgres"}, {"PGPORT", "2345"},
	} {
		unpatch := patchEnv(kv[0], kv[1])
		defer unpatch()
	}

	info := apmpq.ParseDSN("postgres://")
	assert.Equal(t, apmsql.DSNInfo{
		Address:  "postgres",
		Port:     2345,
		Database: "dbinst",
		User:     "bob",
	}, info)
}
