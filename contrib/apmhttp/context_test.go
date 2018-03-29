package apmhttp_test

import (
	"crypto/tls"
	"net/http"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/apm-agent-go/contrib/apmhttp"
	"github.com/elastic/apm-agent-go/model"
)

func TestRequestRemoteAddress(t *testing.T) {
	req := &http.Request{
		RemoteAddr: "[::1]:1234",
		Header:     make(http.Header),
	}
	assert.Equal(t, "::1", apmhttp.RequestRemoteAddress(req))

	req.Header.Set("X-Forwarded-For", "client.invalid")
	assert.Equal(t, "client.invalid", apmhttp.RequestRemoteAddress(req))

	req.Header.Set("X-Forwarded-For", "client.invalid, proxy.invalid")
	assert.Equal(t, "client.invalid", apmhttp.RequestRemoteAddress(req))

	req.Header.Set("X-Real-IP", "127.1.2.3")
	assert.Equal(t, "127.1.2.3", apmhttp.RequestRemoteAddress(req))

	// "for" is missing from Forwarded, so fall back to the next thing
	req.Header.Set("Forwarded", `by=127.0.0.1`)
	assert.Equal(t, "127.1.2.3", apmhttp.RequestRemoteAddress(req))

	req.Header.Set("Forwarded", `for=_secret`)
	assert.Equal(t, "_secret", apmhttp.RequestRemoteAddress(req))

	req.Header.Set("Forwarded", `by=127.0.0.1; for="[2001:db8:cafe::17]:4711"; proto=HTTPS`)
	assert.Equal(t, "2001:db8:cafe::17", apmhttp.RequestRemoteAddress(req))
}

func TestRequestURLClient(t *testing.T) {
	req := mustNewRequest("https://user:pass@host.invalid:9443/path?query&querier=foo#fragment")
	assert.Equal(t, model.URL{
		// Username and password removed
		Full:     "https://host.invalid:9443/path?query&querier=foo#fragment",
		Protocol: "https",
		Hostname: "host.invalid",
		Port:     "9443",
		Path:     "/path",
		Search:   "query&querier=foo",
		Hash:     "fragment",
	}, apmhttp.RequestURL(req))
}

func TestRequestURLServer(t *testing.T) {
	req := mustNewRequest("/path?query&querier=foo")
	req.Host = "host.invalid:8080"

	assert.Equal(t, model.URL{
		Full:     "http://host.invalid:8080/path?query&querier=foo",
		Protocol: "http",
		Hostname: "host.invalid",
		Port:     "8080",
		Path:     "/path",
		Search:   "query&querier=foo",
	}, apmhttp.RequestURL(req))
}

func TestRequestURLServerTLS(t *testing.T) {
	req := mustNewRequest("/path?query&querier=foo")
	req.Host = "host.invalid:8080"
	req.TLS = &tls.ConnectionState{}
	assert.Equal(t, "https", apmhttp.RequestURL(req).Protocol)
}

func TestRequestURLHeaders(t *testing.T) {
	type test struct {
		name   string
		full   string
		header http.Header
	}

	tests := []test{{
		name: "Forwarded",
		full: "https://forwarded.invalid:443/",
		header: http.Header{
			"Forwarded": []string{
				"by=127.0.0.1; for=127.1.1.1; Host=\"forwarded.invalid:443\"; proto=HTTPS",
				"host=whatever", // only first value considered
			},
		},
	}, {
		name:   "Forwarded-Multi",
		full:   "http://first.invalid/",
		header: http.Header{"Forwarded": []string{"host=first.invalid, host=second.invalid"}},
	}, {
		name:   "Forwarded-Malformed-Fields-Ignored",
		full:   "http://first.invalid/",
		header: http.Header{"Forwarded": []string{"what; nonsense=\"; host=first.invalid"}},
	}, {
		name:   "Forwarded-Trailing-Separators",
		full:   "http://first.invalid/",
		header: http.Header{"Forwarded": []string{"host=first.invalid;,"}},
	}, {
		name:   "Forwarded-Empty-Host",
		full:   "http://host.invalid/", // falls back to the next option
		header: http.Header{"Forwarded": []string{"host="}},
	}, {
		name:   "X-Forwarded-Host",
		full:   "http://x-forwarded-host.invalid/",
		header: http.Header{"X-Forwarded-Host": []string{"x-forwarded-host.invalid"}},
	}, {
		name:   "X-Forwarded-Proto",
		full:   "https://host.invalid/",
		header: http.Header{"X-Forwarded-Proto": []string{"https"}},
	}, {
		name:   "X-Forwarded-Protocol",
		full:   "https://host.invalid/",
		header: http.Header{"X-Forwarded-Protocol": []string{"https"}},
	}, {
		name:   "X-Url-Scheme",
		full:   "https://host.invalid/",
		header: http.Header{"X-Url-Scheme": []string{"https"}},
	}, {
		name:   "Front-End-Https",
		full:   "https://host.invalid/",
		header: http.Header{"Front-End-Https": []string{"on"}},
	}, {
		name:   "X-Forwarded-Ssl",
		full:   "https://host.invalid/",
		header: http.Header{"X-Forwarded-Ssl": []string{"on"}},
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := mustNewRequest("/")
			req.Host = "host.invalid"
			req.Header = test.header
			out := apmhttp.RequestURL(req)
			req.Header = nil
			assert.Equal(t, test.full, out.Full)
		})
	}
}

func TestStatusCode(t *testing.T) {
	for i := 100; i < 600; i++ {
		assert.Equal(t, strconv.Itoa(i), apmhttp.StatusCodeString(i))
	}
}

func mustNewRequest(url string) *http.Request {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		panic(err)
	}
	return req
}
