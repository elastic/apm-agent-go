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

//go:build go1.10
// +build go1.10

package apmstrings

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConcat(t *testing.T) {
	type args struct {
		elements []string
	}
	tests := []struct {
		expect   string
		elements []string
	}{
		{elements: []string{"a", "b", "c"}, expect: "abc"},
		{elements: []string{"Calls to ", "redis"}, expect: "Calls to redis"},
	}
	for i, test := range tests {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			assert.Equal(t, test.expect, Concat(test.elements...), i)
		})
	}
}

var benchRes string

func BenchmarkConcat(b *testing.B) {
	elements := map[bool][]string{
		true:  {"Calls to ", "redis"},
		false: {"Calls to ", "verylongservicenamewegothere"},
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		benchRes = Concat(elements[i%2 == 0]...)
	}
}
