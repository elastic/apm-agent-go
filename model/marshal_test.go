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

package model_test

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.elastic.co/apm/model"
	"go.elastic.co/fastjson"
)

func TestMarshalTransaction(t *testing.T) {
	tx := fakeTransaction()

	var w fastjson.Writer
	tx.MarshalFastJSON(&w)

	decoded := mustUnmarshalJSON(w)
	expect := map[string]interface{}{
		"trace_id":  "0102030405060708090a0b0c0d0e0f10",
		"id":        "0102030405060708",
		"parent_id": "0001020304050607",
		"name":      "GET /foo/bar",
		"type":      "request",
		"timestamp": float64(123000000),
		"duration":  123.456,
		"result":    "418",
		"context": map[string]interface{}{
			"custom": map[string]interface{}{
				"bar": true,
				"baz": 3.45,
				"foo": "one",
				"qux": map[string]interface{}{"quux": float64(6)},
			},
			"service": map[string]interface{}{
				"framework": map[string]interface{}{
					"name":    "framework-name",
					"version": "framework-version",
				},
			},
			"request": map[string]interface{}{
				"url": map[string]interface{}{
					"full":     "https://testing.invalid/foo/bar?baz#qux",
					"protocol": "https",
					"hostname": "testing.invalid",
					"pathname": "/foo/bar",
					"search":   "baz",
					"hash":     "qux",
				},
				"method": "GET",
				"headers": map[string]interface{}{
					"User-Agent": "Mosaic/0.2 (Windows 3.1)",
					"Cookie":     "monster=yumyum; random=junk",
				},
				"body":         "ahoj",
				"http_version": "1.1",
				"cookies": map[string]interface{}{
					"monster": "yumyum",
					"random":  "junk",
				},
				"socket": map[string]interface{}{
					"encrypted":      true,
					"remote_address": "[::1]",
				},
			},
			"response": map[string]interface{}{
				"status_code": float64(418),
				"headers": map[string]interface{}{
					"Content-Type": "text/html",
				},
			},
			"user": map[string]interface{}{
				"username": "wanda",
			},
			"tags": map[string]interface{}{
				"tag": "urit",
			},
		},
		"span_count": map[string]interface{}{
			"started": float64(99),
			"dropped": float64(4),
		},
	}
	assert.Equal(t, expect, decoded)
}

func TestMarshalSpan(t *testing.T) {
	var w fastjson.Writer
	span := fakeSpan()
	span.Context = fakeDatabaseSpanContext()
	span.MarshalFastJSON(&w)

	decoded := mustUnmarshalJSON(w)
	assert.Equal(t, map[string]interface{}{
		"trace_id":       "000102030405060708090a0b0c0d0e0f",
		"id":             "0001020304050607",
		"parent_id":      "0001020304050607",
		"transaction_id": "0001020304050607",
		"name":           "SELECT FROM bar",
		"timestamp":      float64(123000000),
		"duration":       float64(3),
		"type":           "db.postgresql.query",
		"context": map[string]interface{}{
			"db": map[string]interface{}{
				"instance":  "wat",
				"statement": `SELECT foo FROM bar WHERE baz LIKE 'qu%x'`,
				"type":      "sql",
				"user":      "barb",
			},
		},
	}, decoded)

	w.Reset()
	span.Duration = 4
	span.Name = "GET testing.invalid:8000"
	span.Type = "ext.http"
	span.ParentID = model.SpanID{} // parent_id is optional
	span.Context = fakeHTTPSpanContext()
	span.MarshalFastJSON(&w)

	decoded = mustUnmarshalJSON(w)
	assert.Equal(t, map[string]interface{}{
		"trace_id":       "000102030405060708090a0b0c0d0e0f",
		"id":             "0001020304050607",
		"transaction_id": "0001020304050607",
		"name":           "GET testing.invalid:8000",
		"timestamp":      float64(123000000),
		"duration":       float64(4),
		"type":           "ext.http",
		"context": map[string]interface{}{
			"http": map[string]interface{}{
				"url": "http://testing.invalid:8000/path?query#fragment",
			},
		},
	}, decoded)
}

func TestMarshalMetrics(t *testing.T) {
	metrics := fakeMetrics()

	var w fastjson.Writer
	metrics.MarshalFastJSON(&w)

	decoded := mustUnmarshalJSON(w)
	expect := map[string]interface{}{
		"timestamp": float64(123000000),
		"tags": map[string]interface{}{
			"foo": "bar",
		},
		"samples": map[string]interface{}{
			"metric_one": map[string]interface{}{
				"value": float64(1024),
			},
			"metric_two": map[string]interface{}{
				"value": float64(-66.6),
			},
		},
	}
	assert.Equal(t, expect, decoded)
}

func TestMarshalError(t *testing.T) {
	var e model.Error
	time, err := time.Parse("2006-01-02T15:04:05.999Z", "1970-01-01T00:02:03Z")
	assert.NoError(t, err)
	e.Timestamp = model.Time(time)

	// The primary error ID is required, all other IDs are optional
	var w fastjson.Writer
	e.MarshalFastJSON(&w)
	assert.Equal(t, `{"id":"00000000000000000000000000000000","timestamp":123000000}`, string(w.Bytes()))

	e.ID = model.TraceID{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}
	e.TransactionID = model.SpanID{1, 2, 3, 4, 5, 6, 7, 8}
	e.TraceID = model.TraceID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	e.ParentID = model.SpanID{1, 2, 3, 4, 5, 6, 7, 8}
	w.Reset()
	e.MarshalFastJSON(&w)
	assert.Equal(t,
		`{"id":"000102030405060708090a0b0c0d0e0f","timestamp":123000000,"parent_id":"0102030405060708","trace_id":"0102030405060708090a0b0c0d0e0f10","transaction_id":"0102030405060708"}`,
		string(w.Bytes()),
	)
}

func TestMarshalErrorTransactionUnsampled(t *testing.T) {
	var e model.Error
	time, err := time.Parse("2006-01-02T15:04:05.999Z", "1970-01-01T00:02:03Z")
	assert.NoError(t, err)
	e.Timestamp = model.Time(time)
	e.Transaction.Sampled = new(bool)
	e.Transaction.Type = "foo"

	var w fastjson.Writer
	e.MarshalFastJSON(&w)
	assert.Equal(t, `{"id":"00000000000000000000000000000000","timestamp":123000000,"transaction":{"sampled":false,"type":"foo"}}`, string(w.Bytes()))
}

func TestMarshalCookies(t *testing.T) {
	cookies := model.Cookies{
		{Name: "foo", Value: "!"}, // eclipsed
		{Name: "baz", Value: "qux"},
		{Name: "foo", Value: "bar"},
	}
	var w fastjson.Writer
	cookies.MarshalFastJSON(&w)
	assert.Equal(t, `{"foo":"bar","baz":"qux"}`, string(w.Bytes()))
}

func TestMarshalRequestBody(t *testing.T) {
	body := model.RequestBody{
		Raw: "rawr",
	}
	var w fastjson.Writer
	body.MarshalFastJSON(&w)
	assert.Equal(t, `"rawr"`, string(w.Bytes()))

	body.Form = url.Values{
		"first":    []string{"jackie"},
		"last":     []string{"brown"},
		"keywords": []string{"rum", "punch"},
	}
	w.Reset()
	body.MarshalFastJSON(&w)

	decoded := mustUnmarshalJSON(w)
	expect := map[string]interface{}{
		"first":    "jackie",
		"last":     "brown",
		"keywords": []interface{}{"rum", "punch"},
	}
	assert.Equal(t, expect, decoded)
}

func TestMarshalLog(t *testing.T) {
	log := model.Log{
		Message:      "foo",
		Level:        "bar",
		LoggerName:   "baz",
		ParamMessage: "%s",
	}
	var w fastjson.Writer
	log.MarshalFastJSON(&w)

	assert.Equal(t, `{"message":"foo","level":"bar","logger_name":"baz","param_message":"%s"}`, string(w.Bytes()))

	log = model.Log{
		Message:    "foo",
		LoggerName: "bar",
	}
	w.Reset()
	log.MarshalFastJSON(&w)
	assert.Equal(t, `{"message":"foo","logger_name":"bar"}`, string(w.Bytes()))
}

func TestMarshalException(t *testing.T) {
	x := model.Exception{
		Message: "foo",
		Type:    "bar",
		Module:  "baz",
		Attributes: map[string]interface{}{
			"qux": map[string]interface{}{
				"quux": "corge",
			},
		},
		Handled: true,
	}
	var w fastjson.Writer
	x.MarshalFastJSON(&w)

	assert.Equal(t,
		`{"handled":true,"message":"foo","attributes":{"qux":{"quux":"corge"}},"module":"baz","type":"bar"}`,
		string(w.Bytes()),
	)
}

func TestMarshalExceptionCode(t *testing.T) {
	code := model.ExceptionCode{
		String: "boom",
		Number: 123,
	}
	var w fastjson.Writer
	code.MarshalFastJSON(&w)
	assert.Equal(t, `"boom"`, string(w.Bytes()))

	w.Reset()
	code.String = ""
	code.MarshalFastJSON(&w)
	assert.Equal(t, `123`, string(w.Bytes()))
}

func TestMarshalUser(t *testing.T) {
	user := model.User{
		Email:    "foo@example.com",
		ID:       "123",
		Username: "bar",
	}
	var w fastjson.Writer
	user.MarshalFastJSON(&w)
	assert.Equal(t, `{"email":"foo@example.com","id":"123","username":"bar"}`, string(w.Bytes()))
}

func TestMarshalStacktraceFrame(t *testing.T) {
	f := model.StacktraceFrame{
		File:         "file.go",
		Line:         123,
		AbsolutePath: "fabulous",
		Function:     "wonderment",
	}
	var w fastjson.Writer
	f.MarshalFastJSON(&w)

	assert.Equal(t,
		`{"filename":"file.go","lineno":123,"abs_path":"fabulous","function":"wonderment"}`,
		string(w.Bytes()),
	)

	f = model.StacktraceFrame{
		File:         "file.go",
		Line:         123,
		LibraryFrame: true,
		ContextLine:  "0",
		PreContext:   []string{"-2", "-1"},
		PostContext:  []string{"+1", "+2"},
		Vars: map[string]interface{}{
			"foo": []string{"bar", "baz"},
		},
	}
	w.Reset()
	f.MarshalFastJSON(&w)
	assert.Equal(t,
		`{"filename":"file.go","lineno":123,"context_line":"0","library_frame":true,"post_context":["+1","+2"],"pre_context":["-2","-1"],"vars":{"foo":["bar","baz"]}}`,
		string(w.Bytes()),
	)
}

func TestMarshalResponse(t *testing.T) {
	finished := true
	headersSent := true
	response := model.Response{
		Finished: &finished,
		Headers: model.Headers{{
			Key:    "Content-Type",
			Values: []string{"text/plain"},
		}},
		HeadersSent: &headersSent,
		StatusCode:  200,
	}
	var w fastjson.Writer
	response.MarshalFastJSON(&w)
	assert.Equal(t,
		`{"finished":true,"headers":{"Content-Type":"text/plain"},"headers_sent":true,"status_code":200}`,
		string(w.Bytes()),
	)
}

func TestMarshalURL(t *testing.T) {
	in := model.URL{
		Path:     "/",
		Search:   "abc=def",
		Hash:     strings.Repeat("x", 1000), // exceed "full" URL length
		Hostname: "testing.invalid",
		Port:     "999",
		Protocol: "http",
	}

	var w fastjson.Writer
	in.MarshalFastJSON(&w)

	var out model.URL
	err := json.Unmarshal(w.Bytes(), &out)
	require.NoError(t, err)

	// The full URL should have been truncated to avoid a validation error.
	assert.Equal(t, "http://testing.invalid:999/?abc=def#"+strings.Repeat("x", 988), out.Full)
	out.Full = ""

	assert.Equal(t, in, out)
}

func TestMarshalURLPathLeadingSlashMissing(t *testing.T) {
	in := model.URL{
		Path:     "foo",
		Search:   "abc=def",
		Hostname: "testing.invalid",
		Port:     "999",
		Protocol: "http",
	}

	var w fastjson.Writer
	in.MarshalFastJSON(&w)

	var out model.URL
	err := json.Unmarshal(w.Bytes(), &out)
	require.NoError(t, err)

	assert.Equal(t, "http://testing.invalid:999/foo?abc=def", out.Full)
	out.Full = ""

	// Leading slash should have been added during marshalling. We do it
	// here rather than when building the model to avoid allocation.
	in.Path = "/foo"

	assert.Equal(t, in, out)
}

func TestMarshalHTTPSpanContextURLPathLeadingSlashMissing(t *testing.T) {
	httpSpanContext := model.HTTPSpanContext{
		URL: mustParseURL("http://testing.invalid:8000/path?query#fragment"),
	}
	httpSpanContext.URL.Path = "path"

	var w fastjson.Writer
	httpSpanContext.MarshalFastJSON(&w)

	var out model.HTTPSpanContext
	err := json.Unmarshal(w.Bytes(), &out)
	require.NoError(t, err)

	// Leading slash should have been added during marshalling.
	httpSpanContext.URL.Path = "/path"
	assert.Equal(t, httpSpanContext.URL, out.URL)
}

func TestTransactionUnmarshalJSON(t *testing.T) {
	tx := fakeTransaction()
	var w fastjson.Writer
	tx.MarshalFastJSON(&w)

	var out model.Transaction
	err := json.Unmarshal(w.Bytes(), &out)
	require.NoError(t, err)
	assert.Equal(t, tx, out)
}

func fakeTransaction() model.Transaction {
	return model.Transaction{
		TraceID:   model.TraceID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		ID:        model.SpanID{1, 2, 3, 4, 5, 6, 7, 8},
		ParentID:  model.SpanID{0, 1, 2, 3, 4, 5, 6, 7},
		Name:      "GET /foo/bar",
		Type:      "request",
		Timestamp: model.Time(time.Unix(123, 0).UTC()),
		Duration:  123.456,
		Result:    "418",
		Context: &model.Context{
			Request: &model.Request{
				URL: model.URL{
					Full:     "https://testing.invalid/foo/bar?baz#qux",
					Hostname: "testing.invalid",
					Protocol: "https",
					Path:     "/foo/bar",
					Search:   "baz",
					Hash:     "qux",
				},
				Method: "GET",
				Headers: model.Headers{{
					Key: "Cookie", Values: []string{"monster=yumyum; random=junk"},
				}, {
					Key: "User-Agent", Values: []string{"Mosaic/0.2 (Windows 3.1)"},
				}},
				Body: &model.RequestBody{
					Raw: "ahoj",
				},
				HTTPVersion: "1.1",
				Cookies: []*http.Cookie{
					{Name: "monster", Value: "yumyum"},
					{Name: "random", Value: "junk"},
				},
				Socket: &model.RequestSocket{
					Encrypted:     true,
					RemoteAddress: "[::1]",
				},
			},
			Response: &model.Response{
				StatusCode: 418,
				Headers: model.Headers{{
					Key: "Content-Type", Values: []string{"text/html"},
				}},
			},
			Custom: model.IfaceMap{
				{Key: "bar", Value: true},
				{Key: "baz", Value: 3.45},
				{Key: "foo", Value: "one"},
				{Key: "qux", Value: map[string]interface{}{"quux": float64(6)}},
			},
			User: &model.User{
				Username: "wanda",
			},
			Tags: model.IfaceMap{{
				Key: "tag", Value: "urit",
			}},
			Service: &model.Service{
				Framework: &model.Framework{
					Name:    "framework-name",
					Version: "framework-version",
				},
			},
		},
		SpanCount: model.SpanCount{
			Started: 99,
			Dropped: 4,
		},
	}
}

func fakeSpan() model.Span {
	return model.Span{
		Name:          "SELECT FROM bar",
		ID:            model.SpanID{0, 1, 2, 3, 4, 5, 6, 7},
		ParentID:      model.SpanID{0, 1, 2, 3, 4, 5, 6, 7},
		TransactionID: model.SpanID{0, 1, 2, 3, 4, 5, 6, 7},
		TraceID:       model.TraceID{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15},
		Timestamp:     model.Time(time.Unix(123, 0).UTC()),
		Duration:      3,
		Type:          "db.postgresql.query",
		Context:       fakeDatabaseSpanContext(),
	}
}

func fakeDatabaseSpanContext() *model.SpanContext {
	return &model.SpanContext{
		Database: &model.DatabaseSpanContext{
			Instance:  "wat",
			Statement: `SELECT foo FROM bar WHERE baz LIKE 'qu%x'`,
			Type:      "sql",
			User:      "barb",
		},
	}
}

func fakeHTTPSpanContext() *model.SpanContext {
	return &model.SpanContext{
		HTTP: &model.HTTPSpanContext{
			URL: mustParseURL("http://testing.invalid:8000/path?query#fragment"),
		},
	}
}

func fakeMetrics() *model.Metrics {
	return &model.Metrics{
		Timestamp: model.Time(time.Unix(123, 0).UTC()),
		Labels:    model.StringMap{{Key: "foo", Value: "bar"}},
		Samples: map[string]model.Metric{
			"metric_one": {Value: 1024},
			"metric_two": {Value: -66.6},
		},
	}
}

func fakeService() *model.Service {
	return &model.Service{
		Name:        "fake-service",
		Version:     "1.0.0-rc1",
		Environment: "dev",
		Agent: &model.Agent{
			Name:    "go",
			Version: "0.1.0",
		},
		Framework: &model.Framework{
			Name:    "gin",
			Version: "1.0",
		},
		Language: &model.Language{
			Name:    "go",
			Version: "1.10",
		},
		Runtime: &model.Runtime{
			Name:    "go",
			Version: "gc 1.10",
		},
	}
}

func fakeSystem() *model.System {
	return &model.System{
		Architecture: "x86_64",
		Hostname:     "host.example",
		Platform:     "linux",
	}
}

func fakeProcess() *model.Process {
	ppid := 1
	return &model.Process{
		Pid:   1234,
		Ppid:  &ppid,
		Title: "my-fake-service",
		Argv:  []string{"my-fake-service", "-f", "config.yml"},
	}
}

func mustParseURL(s string) *url.URL {
	u, err := url.Parse(s)
	if err != nil {
		panic(err)
	}
	return u
}

func newUint64(v uint64) *uint64 {
	return &v
}

func newFloat64(v float64) *float64 {
	return &v
}

func mustUnmarshalJSON(w fastjson.Writer) interface{} {
	var out interface{}
	err := json.Unmarshal(w.Bytes(), &out)
	if err != nil {
		panic(err)
	}
	return out
}
