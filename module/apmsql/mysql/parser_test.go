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

package apmmysql_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.elastic.co/apm/module/apmsql"
	apmmysql "go.elastic.co/apm/module/apmsql/mysql"
)

func TestParseDSN(t *testing.T) {
	info := apmmysql.ParseDSN("user:pass@/dbname")
	assert.Equal(t, "dbname", info.Database)
	assert.Equal(t, "user", info.User)
}

func TestParseDSNError(t *testing.T) {
	info := apmmysql.ParseDSN("nonsense")
	assert.Equal(t, apmsql.DSNInfo{}, info)
}
