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
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.elastic.co/apm/module/apmhttp"
)

func TestServerRequestIgnorer(t *testing.T) {
	r1 := &http.Request{URL: &url.URL{Path: "/foo"}}
	r2 := &http.Request{URL: &url.URL{Path: "/foo", RawQuery: "bar=baz"}}
	r3 := &http.Request{URL: &url.URL{Scheme: "http", Host: "testing.invalid", Path: "/foo", RawQuery: "bar=baz"}}

	testServerRequestIgnorer(t, "", r1, false)
	testServerRequestIgnorer(t, "", r2, false)
	testServerRequestIgnorer(t, "", r3, false)
	testServerRequestIgnorer(t, ",", r1, false) // equivalent to empty

	testServerRequestIgnorer(t, "*/foo*", r1, true)
	testServerRequestIgnorer(t, "*/foo*", r2, true)
	testServerRequestIgnorer(t, "*/foo*", r3, true)
	testServerRequestIgnorer(t, "*/FOO*", r3, true) // case insensitive by default

	testServerRequestIgnorer(t, "*/foo?bar=baz", r1, false)
	testServerRequestIgnorer(t, "*/foo?bar=baz", r2, true)
	testServerRequestIgnorer(t, "*/foo?bar=baz", r3, true)

	testServerRequestIgnorer(t, "http://*", r1, false)
	testServerRequestIgnorer(t, "http://*", r2, false)
	testServerRequestIgnorer(t, "http://*", r3, true)
}

func testServerRequestIgnorer(t *testing.T, ignoreURLs string, r *http.Request, expect bool) {
	testName := fmt.Sprintf("%s_%s", ignoreURLs, r.URL.String())
	t.Run(testName, func(t *testing.T) {
		if os.Getenv("_INSIDE_TEST") != "1" {
			cmd := exec.Command(os.Args[0], "-test.run=^"+regexp.QuoteMeta(t.Name())+"$")
			cmd.Env = append(os.Environ(), "_INSIDE_TEST=1")
			cmd.Env = append(cmd.Env, "ELASTIC_APM_TRANSACTION_IGNORE_URLS="+ignoreURLs)
			assert.NoError(t, cmd.Run())
			return
		}
		defaultIgnorer := apmhttp.DefaultServerRequestIgnorer()
		assert.Equal(t, expect, defaultIgnorer(r))

		tracer := newTracer()
		defer tracer.Close()

		dynamicIgnorer := apmhttp.NewDynamicServerRequestIgnorer(tracer)
		assert.Equal(t, expect, dynamicIgnorer(r))
	})
}

func TestFallbackDeprecatedRequestIgnorer(t *testing.T) {
	if os.Getenv("_INSIDE_TEST") != "1" {
		cmd := exec.Command(os.Args[0], "-test.run=^"+regexp.QuoteMeta(t.Name())+"$")
		cmd.Env = append(os.Environ(), "_INSIDE_TEST=1")
		cmd.Env = append(cmd.Env, "ELASTIC_APM_IGNORE_URLS=*/foo*")
		assert.NoError(t, cmd.Run())
		return
	}
	req := &http.Request{URL: &url.URL{Path: "/foo"}}
	ignorer := apmhttp.DefaultServerRequestIgnorer()
	assert.Equal(t, true, ignorer(req))
}
