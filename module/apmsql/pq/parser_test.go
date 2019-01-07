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

	apmpq "go.elastic.co/apm/module/apmsql/pq"
)

func TestParseDSNURL(t *testing.T) {
	info := apmpq.ParseDSN("postgresql://user:pass@localhost/dbinst")
	assert.Equal(t, "dbinst", info.Database)
	assert.Equal(t, "user", info.User)
}

func TestParseDSNConnectionString(t *testing.T) {
	info := apmpq.ParseDSN("dbname=foo\\ bar user='baz'")
	assert.Equal(t, "foo bar", info.Database)
	assert.Equal(t, "baz", info.User)
}

func TestParseDSNEnv(t *testing.T) {
	os.Setenv("PGDATABASE", "dbinst")
	os.Setenv("PGUSER", "bob")
	defer os.Unsetenv("PGDATABASE")
	defer os.Unsetenv("PGUSER")

	info := apmpq.ParseDSN("postgres://")
	assert.Equal(t, "dbinst", info.Database)
	assert.Equal(t, "bob", info.User)
}
