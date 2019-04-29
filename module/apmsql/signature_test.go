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

package apmsql_test

import (
	"encoding/json"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.elastic.co/apm/module/apmsql"
)

type test struct {
	Comment string `json:"comment,omitempty"`
	Input   string `json:"input"`
	Output  string `json:"output"`
}

func TestQuerySignature(t *testing.T) {
	var tests []test
	data, err := ioutil.ReadFile("testdata/signature_tests.json")
	require.NoError(t, err)
	err = json.Unmarshal(data, &tests)
	require.NoError(t, err)

	for _, test := range tests {
		msgFormat := "%s"
		args := []interface{}{test.Input}
		if test.Comment != "" {
			msgFormat += " (%s)"
			args = append(args, test.Comment)
		}
		out := apmsql.QuerySignature(test.Input)
		if !assert.Equal(t, test.Output, out, append([]interface{}{msgFormat}, args...)) {
			if test.Comment != "" {
				t.Logf("// %s", test.Comment)
			}
			t.Logf("%q => %q", test.Input, test.Output)
		}
	}
}

func BenchmarkQuerySignature(b *testing.B) {
	sql := "SELECT *,(SELECT COUNT(*) FROM table2 WHERE table2.field1 = table1.id) AS count FROM table1 WHERE table1.field1 = 'value'"
	for i := 0; i < b.N; i++ {
		signature := apmsql.QuerySignature(sql)
		if signature != "SELECT FROM table1" {
			panic("unexpected result: " + signature)
		}
		b.SetBytes(int64(len(sql)))
	}
}
