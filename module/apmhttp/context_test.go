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

package apmhttp_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.elastic.co/apm/module/apmhttp"
)

func TestStatusCode(t *testing.T) {
	for i := 100; i < 200; i++ {
		assert.Equal(t, "HTTP 1xx", apmhttp.StatusCodeResult(i))
	}
	for i := 200; i < 300; i++ {
		assert.Equal(t, "HTTP 2xx", apmhttp.StatusCodeResult(i))
	}
	for i := 300; i < 400; i++ {
		assert.Equal(t, "HTTP 3xx", apmhttp.StatusCodeResult(i))
	}
	for i := 400; i < 500; i++ {
		assert.Equal(t, "HTTP 4xx", apmhttp.StatusCodeResult(i))
	}
	for i := 500; i < 600; i++ {
		assert.Equal(t, "HTTP 5xx", apmhttp.StatusCodeResult(i))
	}
	assert.Equal(t, "HTTP 0", apmhttp.StatusCodeResult(0))
	assert.Equal(t, "HTTP 600", apmhttp.StatusCodeResult(600))
}
