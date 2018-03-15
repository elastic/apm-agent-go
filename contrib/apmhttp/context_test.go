package apmhttp_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/apm-agent-go/contrib/apmhttp"
)

// TODO(axw) test the rest

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
}
