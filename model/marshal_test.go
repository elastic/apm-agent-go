package model_test

import (
	"encoding/json"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/apm-agent-go/model"
)

func TestMarshalJSON(t *testing.T) {
	tx := fakeTransaction()
	out, err := json.Marshal(tx)
	if err != nil {
		t.Fatalf("encoding/json.Marshal failed: %v", err)
	}

	var in map[string]interface{}
	if err := json.Unmarshal(out, &in); err != nil {
		t.Fatalf("unmarshalling result failed: %v", err)
	}

	// NOTE(axw) best practice would be to do a round-trip,
	// marshal/unmarshal into the same struct type, but we
	// do not implement unmarshalling in this package.

	expect := map[string]interface{}{
		"id":        "d51ae41d-93da-4984-bba3-ae15e9b2247f",
		"name":      "GET /foo/bar",
		"type":      "request",
		"timestamp": "1970-01-01T00:02:03Z",
		"duration":  123.456,
		"result":    "418",
		"context": map[string]interface{}{
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
					"user-agent": "Mosaic/0.2 (Windows 3.1)",
					"cookie":     "monster=yumyum; random=junk",
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
					"content-type": "text/html",
				},
			},
			"user": map[string]interface{}{
				"username": "wanda",
			},
			"custom": map[string]interface{}{
				"custom.foo": map[string]interface{}{
					"bar": "baz",
					"qux": float64(123),
				},
			},
			"tags": map[string]interface{}{
				"tag": "urit",
			},
		},
		"span_count": map[string]interface{}{
			"dropped": map[string]interface{}{
				"total": float64(4),
			},
		},
		"spans": []interface{}{
			map[string]interface{}{
				"name":     "SELECT FROM bar",
				"start":    float64(2),
				"duration": float64(3),
				"type":     "db.postgresql.query",
				"context": map[string]interface{}{
					"db": map[string]interface{}{
						"instance":  "wat",
						"statement": `SELECT foo FROM bar WHERE baz LIKE 'qu%x'`,
						"type":      "sql",
						"user":      "barb",
					},
				},
			},
		},
	}

	assert.Equal(t, expect, in)
}

func TestErrorMarshalJSON(t *testing.T) {
	var e model.Error
	out, err := json.Marshal(&e)
	assert.NoError(t, err)
	assert.Equal(t, `{"timestamp":"0001-01-01T00:00:00Z"}`, string(out))

	e.TransactionID = "xyz"
	out, err = json.Marshal(&e)
	assert.NoError(t, err)
	assert.Equal(t, `{"timestamp":"0001-01-01T00:00:00Z","transaction":{"id":"xyz"}}`, string(out))
}

func fakeTransaction() *model.Transaction {
	return &model.Transaction{
		ID:        "d51ae41d-93da-4984-bba3-ae15e9b2247f",
		Name:      "GET /foo/bar",
		Type:      "request",
		Timestamp: time.Unix(123, 0),
		Duration:  123456 * time.Microsecond,
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
				Headers: &model.RequestHeaders{
					UserAgent: "Mosaic/0.2 (Windows 3.1)",
					Cookie:    "monster=yumyum; random=junk",
				},
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
				Headers: &model.ResponseHeaders{
					ContentType: "text/html",
				},
			},
			User: &model.User{
				Username: "wanda",
			},
			Custom: map[string]interface{}{
				"custom.foo": map[string]interface{}{
					"bar": "baz",
					"qux": 123,
				},
			},
			Tags: map[string]string{
				"tag": "urit",
			},
		},
		SpanCount: &model.SpanCount{
			Dropped: &model.SpanCountDropped{
				Total: 4,
			},
		},
		Spans: []*model.Span{{
			Name:     "SELECT FROM bar",
			Start:    2 * time.Millisecond,
			Duration: 3 * time.Millisecond,
			Type:     "db.postgresql.query",
			Context: &model.SpanContext{
				Database: &model.DatabaseSpanContext{
					Instance:  "wat",
					Statement: `SELECT foo FROM bar WHERE baz LIKE 'qu%x'`,
					Type:      "sql",
					User:      "barb",
				},
			},
		}},
	}
}

func fakeService() *model.Service {
	return &model.Service{
		Name:        "fake-service",
		Version:     "1.0.0-rc1",
		Environment: "dev",
		Agent: model.Agent{
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

func fakeTransactionsPayload(n int) model.TransactionsPayload {
	transactions := make([]*model.Transaction, n)
	tx := fakeTransaction()
	for i := range transactions {
		transactions[i] = tx
	}
	return model.TransactionsPayload{
		Service:      fakeService(),
		Process:      fakeProcess(),
		System:       fakeSystem(),
		Transactions: transactions,
	}
}

func mustParseURL(s string) *url.URL {
	u, err := url.Parse(s)
	if err != nil {
		panic(err)
	}
	return u
}
