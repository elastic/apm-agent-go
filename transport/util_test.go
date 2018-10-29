package transport_test

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

type nopHandler struct{}

func (nopHandler) ServeHTTP(http.ResponseWriter, *http.Request) {}

type recordingHandler struct {
	mu       sync.Mutex
	requests []*http.Request
}

func (h *recordingHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	h.mu.Lock()
	defer h.mu.Unlock()

	var buf bytes.Buffer
	_, err := io.Copy(&buf, req.Body)
	if err != nil {
		panic(err)
	}
	req.Body = ioutil.NopCloser(&buf)
	h.requests = append(h.requests, req)
}

func assertAuthorization(t *testing.T, req *http.Request, token string) {
	values, ok := req.Header["Authorization"]
	if !ok {
		if token == "" {
			return
		}
		t.Errorf("missing Authorization header")
		return
	}
	var expect []string
	if token != "" {
		expect = []string{"Bearer " + token}
	}
	assert.Equal(t, expect, values)
}

func patchEnv(key, value string) func() {
	old, had := os.LookupEnv(key)
	if err := os.Setenv(key, value); err != nil {
		panic(err)
	}
	return func() {
		var err error
		if !had {
			err = os.Unsetenv(key)
		} else {
			err = os.Setenv(key, old)
		}
		if err != nil {
			panic(err)
		}
	}
}
