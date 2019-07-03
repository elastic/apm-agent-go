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

package apmstrings_test

import (
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"

	"go.elastic.co/apm/internal/apmstrings"
)

func TestTruncate(t *testing.T) {
	const limit = 2
	test := func(name, in, expect string) {
		t.Run(name, func(t *testing.T) {
			out, n := apmstrings.Truncate(in, limit)
			assert.Equal(t, expect, out)
			assert.Equal(t, utf8.RuneCountInString(out), n)
		})
	}
	test("empty", "", "")
	test("limit_ascii", "xx", "xx")
	test("limit_multibyte", "世界", "世界")
	test("truncate_ascii", "xxx", "xx")
	test("truncate_multibyte", "世界世", "世界")
}
