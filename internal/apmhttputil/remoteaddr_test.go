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

package apmhttputil_test

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.elastic.co/apm/internal/apmhttputil"
)

func TestRemoteAddr(t *testing.T) {
	req := &http.Request{
		RemoteAddr: "[::1]:1234",
		Header:     make(http.Header),
	}
	req.Header.Set("X-Forwarded-For", "client.invalid, proxy.invalid")
	req.Header.Set("X-Real-IP", "127.1.2.3")
	assert.Equal(t, "::1", apmhttputil.RemoteAddr(req))
}

func TestDestinationAddr(t *testing.T) {
	test := func(u, expectAddr string, expectPort int) {
		t.Run(u, func(t *testing.T) {
			url, err := url.Parse(u)
			require.NoError(t, err)

			addr, port := apmhttputil.DestinationAddr(&http.Request{URL: url})
			assert.Equal(t, expectAddr, addr)
			assert.Equal(t, expectPort, port)
		})
	}
	test("http://127.0.0.1:80", "127.0.0.1", 80)
	test("http://127.0.0.1", "127.0.0.1", 80)
	test("https://127.0.0.1:443", "127.0.0.1", 443)
	test("https://127.0.0.1", "127.0.0.1", 443)
	test("https://[::1]", "::1", 443)
	test("https://[::1]:1234", "::1", 1234)
	test("gopher://gopher.invalid:70", "gopher.invalid", 70)
	test("gopher://gopher.invalid", "gopher.invalid", 0) // default unknown
}
