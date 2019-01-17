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

package apm_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.elastic.co/apm"
	"go.elastic.co/apm/apmtest"
	"go.elastic.co/apm/model"
	"go.elastic.co/apm/transport/transporttest"
)

func TestSanitizeRequestResponse(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://server.testing/", nil)
	req.SetBasicAuth("foo", "bar")
	for _, c := range []*http.Cookie{
		{Name: "secret", Value: "top"},
		{Name: "Custom-Credit-Card-Number", Value: "top"},
		{Name: "sessionid", Value: "123"},
		{Name: "user_id", Value: "456"},
	} {
		req.AddCookie(c)
	}

	tx, _, _ := apmtest.WithTransaction(func(ctx context.Context) {
		tx := apm.TransactionFromContext(ctx)
		tx.Context.SetHTTPRequest(req)

		h := make(http.Header)
		h.Add("Set-Cookie", (&http.Cookie{Name: "foo", Value: "bar"}).String())
		h.Add("Set-Cookie", (&http.Cookie{Name: "baz", Value: "qux"}).String())
		tx.Context.SetHTTPResponseHeaders(h)
		tx.Context.SetHTTPStatusCode(http.StatusTeapot)
	})

	assert.Equal(t, tx.Context.Request.Cookies, model.Cookies{
		{Name: "Custom-Credit-Card-Number", Value: "[REDACTED]"},
		{Name: "secret", Value: "[REDACTED]"},
		{Name: "sessionid", Value: "[REDACTED]"},
		{Name: "user_id", Value: "456"},
	})
	assert.Equal(t, model.Headers{{
		Key:    "Authorization",
		Values: []string{"[REDACTED]"},
	}}, tx.Context.Request.Headers)

	// NOTE: the response includes multiple Set-Cookie headers,
	// but we only report a single "[REDACTED]" value.
	assert.Equal(t, model.Headers{{
		Key:    "Set-Cookie",
		Values: []string{"[REDACTED]"},
	}}, tx.Context.Response.Headers)
}

func TestSetSanitizedFieldNamesNone(t *testing.T) {
	testSetSanitizedFieldNames(t, "top")
}

func TestSetSanitizedFieldNamesCaseSensitivity(t *testing.T) {
	// patterns are matched case-insensitively by default
	testSetSanitizedFieldNames(t, "[REDACTED]", "Secret")

	// patterns can be made case-sensitive by clearing the "i" flag.
	testSetSanitizedFieldNames(t, "top", "(?-i:Secret)")
}

func testSetSanitizedFieldNames(t *testing.T, expect string, sanitized ...string) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()
	tracer.SetSanitizedFieldNames(sanitized...)

	req, _ := http.NewRequest("GET", "http://server.testing/", nil)
	req.AddCookie(&http.Cookie{Name: "secret", Value: "top"})

	tx := tracer.StartTransaction("name", "type")
	tx.Context.SetHTTPRequest(req)
	tx.End()
	tracer.Flush(nil)
	payloads := transport.Payloads()
	require.Len(t, payloads.Transactions, 1)

	assert.Equal(t, payloads.Transactions[0].Context.Request.Cookies, model.Cookies{
		{Name: "secret", Value: expect},
	})
}
