package apmhttp_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/apm-agent-go/module/apmhttp"
)

func TestStatusCode(t *testing.T) {
	for i := 100; i < 200; i++ {
		assert.Equal(t, "HTTP 1xx", apmhttp.StatusCodeResult(i))
	}
	for i := 200; i < 300; i++ {
		assert.Equal(t, "HTTP 2xx", apmhttp.StatusCodeResult(i))
	}
	for i := 300; i < 400; i++ {
		assert.Equal(t, "HTTP 3xx", apmhttp.StatusCodeResult(i))
	}
	for i := 400; i < 500; i++ {
		assert.Equal(t, "HTTP 4xx", apmhttp.StatusCodeResult(i))
	}
	for i := 500; i < 600; i++ {
		assert.Equal(t, "HTTP 5xx", apmhttp.StatusCodeResult(i))
	}
	assert.Equal(t, "HTTP 0", apmhttp.StatusCodeResult(0))
	assert.Equal(t, "HTTP 600", apmhttp.StatusCodeResult(600))
}
