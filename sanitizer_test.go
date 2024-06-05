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
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.elastic.co/apm/v2"
	"go.elastic.co/apm/v2/apmtest"
	"go.elastic.co/apm/v2/model"
	"go.elastic.co/apm/v2/transport/transporttest"
)

func TestSanitizeRequestResponse(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://server.testing/", nil)
	req.SetBasicAuth("foo", "bar")
	req.Header.Set("X-Custom-Preauth-Header", "fubar")

	for _, c := range []*http.Cookie{
		{Name: "secret", Value: "top"},
		{Name: "Custom-Credit-Card-Number", Value: "top"},
		{Name: "sessionid", Value: "123"},
		{Name: "user_id", Value: "456"},
	} {
		req.AddCookie(c)
	}

	tx, _, errors := apmtest.WithTransaction(func(ctx context.Context) {
		e := apm.CaptureError(ctx, errors.New("boom!"))
		defer e.Send()

		tx := apm.TransactionFromContext(ctx)
		tx.Context.SetHTTPRequest(req)
		e.Context.SetHTTPRequest(req)

		h := make(http.Header)
		h.Add("Set-Cookie", (&http.Cookie{Name: "foo", Value: "bar"}).String())
		h.Add("Set-Cookie", (&http.Cookie{Name: "baz", Value: "qux"}).String())
		h.Set("X-Custom-Authly-Header", "bazquux")
		tx.Context.SetHTTPResponseHeaders(h)
		tx.Context.SetHTTPStatusCode(http.StatusTeapot)
		e.Context.SetHTTPResponseHeaders(h)
		e.Context.SetHTTPStatusCode(http.StatusTeapot)
	})

	checkContext := func(context *model.Context) {
		assert.Equal(t, context.Request.Cookies, model.Cookies{
			{Name: "Custom-Credit-Card-Number", Value: "[REDACTED]"},
			{Name: "secret", Value: "[REDACTED]"},
			{Name: "sessionid", Value: "[REDACTED]"},
			{Name: "user_id", Value: "456"},
		})
		assert.Equal(t, model.Headers{
			{
				Key:    "Authorization",
				Values: []string{"[REDACTED]"},
			},
			{
				Key:    "X-Custom-Preauth-Header",
				Values: []string{"[REDACTED]"},
			},
		}, context.Request.Headers)

		// NOTE: the response includes multiple Set-Cookie headers,
		// but we only report a single "[REDACTED]" value.
		assert.Equal(t, model.Headers{
			{
				Key:    "Set-Cookie",
				Values: []string{"[REDACTED]"},
			},
			{
				Key:    "X-Custom-Authly-Header",
				Values: []string{"[REDACTED]"},
			},
		}, context.Response.Headers)
	}
	checkContext(tx.Context)
	for _, e := range errors {
		checkContext(e.Context)
	}
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

type testSanitizeRaw struct {
	Name  string               `json:"name"`
	Infos testSanitizeRawInfos `json:"infos"`
}

type testSanitizeRawInfos struct {
	Id       string                  `json:"id"`
	Balance  interface{}             `json:"balance"` // interface{} to allow unmarshalling the sanitized body as number or string
	CardInfo testSanitizeRawCardInfo `json:"cardInfo"`
}

type testSanitizeRawCardInfo struct {
	CardNumber string `json:"cardNumber"`
}

func TestSanitizeRaw(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	tracer.SetSanitizedFieldNames("id", "balance", "cardNumber")
	tracer.SetCaptureBody(apm.CaptureBodyAll)

	body := testSanitizeRaw{
		Name: "Helias",
		Infos: testSanitizeRawInfos{
			Id:      "12345678912",
			Balance: 123.45,
			CardInfo: testSanitizeRawCardInfo{
				CardNumber: "321321321321321",
			},
		},
	}

	bodyBytes, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "http://server.testing/", bytes.NewBufferString(string(bodyBytes)))
	req.Header.Set("Content-Type", "application/json")
	req.ParseForm()

	tx := tracer.StartTransaction("name", "type")

	bodyCapturer := tracer.CaptureHTTPRequestBody(req)
	tx.Context.SetHTTPRequestBody(bodyCapturer)
	bodyCapturer.Discard()

	tx.Context.SetHTTPRequest(req)
	tx.End()
	tracer.Flush(nil)
	payloads := transport.Payloads()
	require.Len(t, payloads.Transactions, 1)

	body.Infos.Id = "[REDACTED]"
	body.Infos.CardInfo.CardNumber = "[REDACTED]"
	body.Infos.Balance = "[REDACTED]"

	var sanitizedBody testSanitizeRaw
	err := json.Unmarshal([]byte(payloads.Transactions[0].Context.Request.Body.Raw), &sanitizedBody)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, body.Infos.Id, sanitizedBody.Infos.Id)
	assert.Equal(t, body.Infos.CardInfo.CardNumber, sanitizedBody.Infos.CardInfo.CardNumber)
	assert.Equal(t, body.Infos.Balance, sanitizedBody.Infos.Balance)
}
