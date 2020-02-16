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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte("hello"))
		w.(http.Flusher).Flush()
		time.Sleep(500 * time.Millisecond)
		w.Write([]byte("world"))
	}))
	defer server.Close()

	var elapsed time.Duration
	client := &http.Client{Transport: apmelasticsearch.WrapRoundTripper(nil)}
	_, spans, errs := apmtest.WithTransaction(func(ctx context.Context) {
		before := time.Now()
		resp, err := ctxhttp.Get(ctx, client, server.URL+"/twitter/_search")
		require.NoError(t, err)
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
		elapsed = time.Since(before)
	})
	assert.Empty(t, errs)
	require.Len(t, spans, 1)

	assert.InEpsilon(t,
		elapsed,
		spans[0].Duration*float64(time.Millisecond),
		0.1, // 10% error
	)
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
	req, _ := http.NewRequest("POST", server.URL+"/twitter/_update_by_query", strings.NewReader("Request.Body"))

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
	testStatementGetBody(t, "_search")
	testStatementGetBody(t, "_msearch")
	testStatementGetBody(t, "_search/template")
	testStatementGetBody(t, "_msearch/template")
	testStatementGetBody(t, "_rollup_search")
}

func testStatementGetBody(t *testing.T, path string) {
	t.Run(path, func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {}))
		defer server.Close()

		client := &http.Client{Transport: apmelasticsearch.WrapRoundTripper(http.DefaultTransport)}
		req, _ := http.NewRequest("GET", server.URL+"/twitter/"+path, strings.NewReader("Request.Body"))
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
	})
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
		rc := readCloser{Reader: errorReader{errors.New("Read failed")}}
		runTest(t, func() (io.ReadCloser, error) { return &rc, nil })
		assert.True(t, rc.closed)
	})
	t.Run("Close", func(t *testing.T) {
		rc := readCloser{}
		runTest(t, func() (io.ReadCloser, error) { return &rc, nil })
		assert.True(t, rc.closed)
	})
}

func TestStatementBodyReadError(t *testing.T) {
	// Use a custom RoundTripper to check that the request body is passed
	// on unharmed, including the error, when the instrumentation receives
	// an error reading from it to capture the search body.
	const contentLength = 5
	readErr := errors.New("Read failed")
	var roundTripper roundTripperFunc = func(req *http.Request) (*http.Response, error) {
		defer req.Body.Close()
		data, err := ioutil.ReadAll(req.Body)
		require.EqualError(t, err, readErr.Error())
		assert.Equal(t, strings.Repeat("!", contentLength), string(data))
		return nil, err
	}
	client := &http.Client{Transport: apmelasticsearch.WrapRoundTripper(roundTripper)}

	rc := readCloser{Reader: errorReader{readErr}}
	req, _ := http.NewRequest("GET", "http://testing.invalid/twitter/_search", &rc)
	req.ContentLength = contentLength
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

func TestDestination(t *testing.T) {
	var rt roundTripperFunc = func(req *http.Request) (*http.Response, error) {
		return httptest.NewRecorder().Result(), nil
	}
	client := &http.Client{Transport: apmelasticsearch.WrapRoundTripper(rt)}

	test := func(url, destinationAddr string, destinationPort int) {
		req, err := http.NewRequest("GET", url, nil)
		require.NoError(t, err)
		_, spans, _ := apmtest.WithTransaction(func(ctx context.Context) {
			resp, err := client.Do(req.WithContext(ctx))
			assert.NoError(t, err)
			resp.Body.Close()
		})
		require.Len(t, spans, 1)
		assert.Equal(t, &model.DestinationSpanContext{
			Address: destinationAddr,
			Port:    destinationPort,
			Service: &model.DestinationServiceSpanContext{
				Type:     "db",
				Name:     "elasticsearch",
				Resource: "elasticsearch",
			},
		}, spans[0].Context.Destination)
	}
	test("http://host:9200/_search", "host", 9200)
	test("http://host:80/_search", "host", 80)
	test("http://127.0.0.1:9200/_search", "127.0.0.1", 9200)
	test("http://[2001:db8::1]:9200/_search", "2001:db8::1", 9200)
	test("http://[2001:db8::1]:80/_search", "2001:db8::1", 80)
}

type readCloser struct {
	io.Reader
	closed bool
}

func (r *readCloser) Read(p []byte) (int, error) {
	if r.Reader != nil {
		return r.Reader.Read(p)
	}
	return len(p), nil
}

func (r *readCloser) Close() error {
	r.closed = true
	return errors.New("Close failed")
}

type errorReader struct {
	err error
}

func (e errorReader) Read(p []byte) (int, error) {
	if e.err != nil {
		copy(p, bytes.Repeat([]byte("!"), len(p)))
		return len(p), e.err
	}
	return len(p), nil
}

type readerFunc func(p []byte) (int, error)

func (f readerFunc) Read(p []byte) (int, error) {
	return f(p)
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (r roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return r(req)
}
