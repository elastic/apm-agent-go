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

package sqlscanner

import (
	"encoding/json"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type test struct {
	Name    string `json:"name"`
	Comment string `json:"comment,omitempty"`
	Input   string `json:"input"`
	Tokens  []struct {
		Kind string `json:"kind"`
		Text string `json:"text"`
	} `json:"tokens,omitempty"`
}

func TestScanner(t *testing.T) {
	var tests []test
	data, err := ioutil.ReadFile("testdata/tests.json")
	require.NoError(t, err)
	err = json.Unmarshal(data, &tests)
	require.NoError(t, err)

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			msgFormat := "%s"
			args := []interface{}{test.Input}
			if test.Comment != "" {
				msgFormat += " (%s)"
				args = append(args, test.Comment)
			}

			s := NewScanner(test.Input)
			for _, tok := range test.Tokens {
				if !assert.True(t, s.Scan()) {
					return
				}
				assert.Equal(t, tok.Kind, s.Token().String())
				assert.Equal(t, tok.Text, s.Text())
			}
			assert.False(t, s.Scan())
		})
	}
}
