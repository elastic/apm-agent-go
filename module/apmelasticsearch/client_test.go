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

package apmelasticsearch_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context/ctxhttp"

	"go.elastic.co/apm/apmtest"
	"go.elastic.co/apm/model"
	"go.elastic.co/apm/module/apmelasticsearch"
)

func TestWrapRoundTripper(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {}))
	defer server.Close()

	client := &http.Client{Transport: apmelasticsearch.WrapRoundTripper(http.DefaultTransport)}
	req1, _ := http.NewRequest("GET", server.URL+"/twitter/_search?q=user:kimchy", nil)
	req2, _ := http.NewRequest("GET", server.URL+"/twitter/_search", strings.NewReader(`query":{term":{"user":"kimchy"}}`))
	req2.SetBasicAuth("Aladdin", "open sesame")
	req2.GetBody = nil

	_, spans, errs := apmtest.WithTransaction(func(ctx context.Context) {
		resp1, err := client.Do(req1.WithContext(ctx))
		require.NoError(t, err)
		resp1.Body.Close()

		resp2, err := client.Do(req2.WithContext(ctx))
		require.NoError(t, err)
		resp2.Body.Close()
	})
	assert.Empty(t, errs)
	require.Len(t, spans, 2)

	assert.Equal(t, "q=user:kimchy", spans[0].Context.HTTP.URL.RawQuery)
	assert.Equal(t, &model.DatabaseSpanContext{
		Type:      "elasticsearch",
		Statement: "user:kimchy",
	}, spans[0].Context.Database)

	assert.Equal(t, &model.DatabaseSpanContext{
		Type:      "elasticsearch",
		Statement: `query":{term":{"user":"kimchy"}}`,
		User:      "Aladdin",
	}, spans[1].Context.Database)
}

func TestSpanDuration(t *testing.T) {
	const delay = 500 * time.Millisecond
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte("hello"))
		w.(http.Flusher).Flush()
		time.Sleep(delay)
		w.Write([]byte("world"))
	}))
	defer server.Close()

	client := &http.Client{Transport: apmelasticsearch.WrapRoundTripper(nil)}
	_, spans, errs := apmtest.WithTransaction(func(ctx context.Context) {
		resp, err := ctxhttp.Get(ctx, client, server.URL+"/twitter/_search")
		require.NoError(t, err)
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
	})
	assert.Empty(t, errs)
	require.Len(t, spans, 1)

	assert.InDelta(t, delay/time.Millisecond, spans[0].Duration, 100)
}

func TestStatementTruncation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {}))
	defer server.Close()

	bodyContent := strings.Repeat("*", 10000) + "!"
	client := &http.Client{Transport: apmelasticsearch.WrapRoundTripper(http.DefaultTransport)}
	req, _ := http.NewRequest("GET", server.URL+"/twitter/_search", strings.NewReader(bodyContent))

	_, spans, errs := apmtest.WithTransaction(func(ctx context.Context) {
		resp, err := client.Do(req.WithContext(ctx))
		require.NoError(t, err)
		resp.Body.Close()
	})
	assert.Empty(t, errs)
	require.Len(t, spans, 1)
	assert.Equal(t, bodyContent[:10000], spans[0].Context.Database.Statement)
}

func TestStatementNonSearch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {}))
	defer server.Close()

	client := &http.Client{Transport: apmelasticsearch.WrapRoundTripper(http.DefaultTransport)}
	req, _ := http.NewRequest("GET", server.URL+"/twitter/_msearch", strings.NewReader("Request.Body"))

	_, spans, errs := apmtest.WithTransaction(func(ctx context.Context) {
		resp, err := client.Do(req.WithContext(ctx))
		require.NoError(t, err)
		resp.Body.Close()
	})
	assert.Empty(t, errs)
	require.Len(t, spans, 1)

	// Only _search requests get a statement.
	assert.Equal(t, "", spans[0].Context.Database.Statement)
}

func TestStatementGetBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {}))
	defer server.Close()

	client := &http.Client{Transport: apmelasticsearch.WrapRoundTripper(http.DefaultTransport)}
	req, _ := http.NewRequest("GET", server.URL+"/twitter/_search", strings.NewReader("Request.Body"))
	req.GetBody = func() (io.ReadCloser, error) {
		return ioutil.NopCloser(strings.NewReader("Request.GetBody")), nil
	}

	_, spans, errs := apmtest.WithTransaction(func(ctx context.Context) {
		resp, err := client.Do(req.WithContext(ctx))
		require.NoError(t, err)
		resp.Body.Close()
	})
	assert.Empty(t, errs)
	require.Len(t, spans, 1)

	assert.Equal(t, &model.DatabaseSpanContext{
		Type:      "elasticsearch",
		Statement: "Request.GetB", // limited to Content-Length
	}, spans[0].Context.Database)
}

func TestStatementGetBodyErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {}))
	defer server.Close()
	client := &http.Client{Transport: apmelasticsearch.WrapRoundTripper(http.DefaultTransport)}

	runTest := func(t *testing.T, getBody func() (io.ReadCloser, error)) {
		req, _ := http.NewRequest("GET", server.URL+"/twitter/_search", strings.NewReader("Request.Body"))
		req.GetBody = getBody
		_, spans, errs := apmtest.WithTransaction(func(ctx context.Context) {
			resp, err := client.Do(req.WithContext(ctx))
			require.NoError(t, err)
			resp.Body.Close()
		})
		assert.Empty(t, errs)
		require.Len(t, spans, 1)

		assert.Equal(t, &model.DatabaseSpanContext{
			Type:      "elasticsearch",
			Statement: "", // GetBody/reader returned an error
		}, spans[0].Context.Database)
	}

	t.Run("GetBody", func(t *testing.T) {
		runTest(t, func() (io.ReadCloser, error) { return nil, errors.New("nope") })
	})
	t.Run("Read", func(t *testing.T) {
		rc := errorReadCloser{readError: errors.New("Read failed")}
		runTest(t, func() (io.ReadCloser, error) { return &rc, nil })
		assert.True(t, rc.closed)
	})
	t.Run("Close", func(t *testing.T) {
		rc := errorReadCloser{}
		runTest(t, func() (io.ReadCloser, error) { return &rc, nil })
		assert.True(t, rc.closed)
	})
}

func TestStatementBodyReadError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {}))
	defer server.Close()
	client := &http.Client{Transport: apmelasticsearch.WrapRoundTripper(http.DefaultTransport)}

	rc := errorReadCloser{readError: errors.New("Read failed")}
	req, _ := http.NewRequest("GET", server.URL+"/twitter/_search", &rc)
	_, spans, errs := apmtest.WithTransaction(func(ctx context.Context) {
		_, err := client.Do(req.WithContext(ctx))
		require.Error(t, err)
		assert.Regexp(t, "Get .*: Read failed", err.Error())
	})
	assert.Empty(t, errs)
	assert.True(t, rc.closed)
	require.Len(t, spans, 1)

	assert.Equal(t, &model.DatabaseSpanContext{
		Type:      "elasticsearch",
		Statement: "", // req.Body.Read returned an error
	}, spans[0].Context.Database)
}

func TestStatementBodyGzipContentEncoding(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {}))
	defer server.Close()
	client := &http.Client{Transport: apmelasticsearch.WrapRoundTripper(http.DefaultTransport)}

	var body bytes.Buffer
	gzipWriter := gzip.NewWriter(&body)
	gzipWriter.Write([]byte("decoded"))
	assert.NoError(t, gzipWriter.Close())

	req, _ := http.NewRequest("GET", server.URL+"/twitter/_search", &body)
	req.Header.Set("Content-Encoding", "gzip")

	_, spans, errs := apmtest.WithTransaction(func(ctx context.Context) {
		resp, err := client.Do(req.WithContext(ctx))
		assert.NoError(t, err)
		resp.Body.Close()
	})
	assert.Empty(t, errs)
	require.Len(t, spans, 1)

	assert.Equal(t, &model.DatabaseSpanContext{
		Type:      "elasticsearch",
		Statement: "decoded",
	}, spans[0].Context.Database)
}

type errorReadCloser struct {
	readError error
	closed    bool
}

func (r *errorReadCloser) Read(p []byte) (int, error) {
	if r.readError != nil {
		copy(p, bytes.Repeat([]byte("!"), len(p)))
		return len(p), r.readError
	}
	return len(p), nil
}

func (r *errorReadCloser) Close() error {
	r.closed = true
	return errors.New("Close failed")
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (r roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return r(req)
}
