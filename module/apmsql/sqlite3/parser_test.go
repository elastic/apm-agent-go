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

package apmsqlite3_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.elastic.co/apm/module/apmsql/v2"
	apmsqlite3 "go.elastic.co/apm/module/apmsql/v2/sqlite3"
	_ "go.elastic.co/apm/v2/apmtest" // disable default tracer
)

func TestParseDSN(t *testing.T) {
	assert.Equal(t, apmsql.DSNInfo{Database: "test.db"}, apmsqlite3.ParseDSN("test.db"))
	assert.Equal(t, apmsql.DSNInfo{Database: ":memory:"}, apmsqlite3.ParseDSN(":memory:"))
	assert.Equal(t, apmsql.DSNInfo{Database: "file:test.db"}, apmsqlite3.ParseDSN("file:test.db?cache=shared&mode=memory"))
}
