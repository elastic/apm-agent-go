package apm_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.elastic.co/apm/model"
	"go.elastic.co/apm/module/apmhttp"
	"go.elastic.co/apm/transport/transporttest"
)

func TestSanitizeRequestResponse(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	mux := http.NewServeMux()
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		http.SetCookie(w, &http.Cookie{
			Name:  "foo",
			Value: "bar",
		})
		http.SetCookie(w, &http.Cookie{
			Name:  "baz",
			Value: "qux",
		})
		w.WriteHeader(http.StatusTeapot)
	}))
	h := apmhttp.Wrap(mux, apmhttp.WithTracer(tracer))

	w := httptest.NewRecorder()
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
	h.ServeHTTP(w, req)
	tracer.Flush(nil)

	payloads := transport.Payloads()
	tx := payloads.Transactions[0]

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

	mux := http.NewServeMux()
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))
	h := apmhttp.Wrap(mux, apmhttp.WithTracer(tracer))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "http://server.testing/", nil)
	req.AddCookie(&http.Cookie{Name: "secret", Value: "top"})
	h.ServeHTTP(w, req)
	tracer.Flush(nil)

	payloads := transport.Payloads()
	tx := payloads.Transactions[0]

	assert.Equal(t, tx.Context.Request.Cookies, model.Cookies{
		{Name: "secret", Value: expect},
	})
}
