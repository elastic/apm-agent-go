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

func assertAuthorization(t *testing.T, req *http.Request, expect ...string) {
	values, ok := req.Header["Authorization"]
	if ok && len(expect) == 0 {
		t.Errorf("unexpected Authorization header")
		return
	}
	if !ok && len(expect) != 0 {
		t.Errorf("missing Authorization header")
		return
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
