package elasticapm_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/apm-agent-go/model"
	"github.com/elastic/apm-agent-go/module/apmhttp"
	"github.com/elastic/apm-agent-go/transport/transporttest"
)

func TestSanitizeRequest(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	mux := http.NewServeMux()
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))
	h := apmhttp.Wrap(mux, apmhttp.WithTracer(tracer))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "http://server.testing/", nil)
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
	require.Len(t, payloads, 1)
	transactions := payloads[0].Transactions()
	require.Len(t, transactions, 1)

	tx := transactions[0]
	assert.Equal(t, tx.Context.Request.Cookies, model.Cookies{
		{Name: "Custom-Credit-Card-Number", Value: "[REDACTED]"},
		{Name: "secret", Value: "[REDACTED]"},
		{Name: "sessionid", Value: "[REDACTED]"},
		{Name: "user_id", Value: "456"},
	})
	assert.Equal(t,
		"secret=[REDACTED];Custom-Credit-Card-Number=[REDACTED];sessionid=[REDACTED];user_id=456",
		tx.Context.Request.Headers.Cookie,
	)
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
	require.Len(t, payloads, 1)
	transactions := payloads[0].Transactions()
	require.Len(t, transactions, 1)

	tx := transactions[0]
	assert.Equal(t, tx.Context.Request.Cookies, model.Cookies{
		{Name: "secret", Value: expect},
	})
	assert.Equal(t, "secret="+expect, tx.Context.Request.Headers.Cookie)
}
