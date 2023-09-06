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
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.elastic.co/apm/module/apmelasticsearch/v2"
	"go.elastic.co/apm/module/apmhttp/v2"
	"go.elastic.co/apm/v2"
	"go.elastic.co/apm/v2/apmtest"
	"go.elastic.co/apm/v2/model"
	"go.elastic.co/apm/v2/transport/transporttest"
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
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/twitter/_search", nil)
		require.NoError(t, err)
		resp, err := client.Do(req)
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
		res := httptest.NewRecorder().Result()
		res.Request = req
		return res, nil
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

func TestServiceTarget(t *testing.T) {
	baseURL := "https://testing.invalid:9200"
	var foundHandlingCluster string
	var requestError error
	var requestPaths []string
	var rt roundTripperFunc = func(req *http.Request) (*http.Response, error) {
		requestPaths = append(requestPaths, req.URL.Path)
		if requestError != nil {
			return nil, requestError
		}
		rec := httptest.NewRecorder()
		if foundHandlingCluster != "" {
			rec.Header().Add("x-found-handling-cluster", foundHandlingCluster)
		}
		res := rec.Result()
		res.Request = req
		return res, nil
	}
	client := &http.Client{Transport: apmelasticsearch.WrapRoundTripper(rt)}

	var overrideHost string
	doRequest := func() *model.ServiceTargetSpanContext {
		req, err := http.NewRequest("GET", baseURL+"/_search", nil)
		require.NoError(t, err)
		if overrideHost != "" {
			req.Host = overrideHost
		}
		_, spans, _ := apmtest.WithTransaction(func(ctx context.Context) {
			resp, err := client.Do(req.WithContext(ctx))
			if requestError != nil {
				expectedError := &url.Error{Op: "Get", URL: req.URL.String(), Err: requestError}
				assert.EqualError(t, err, expectedError.Error())
			} else {
				assert.NoError(t, err)
			}
			if err == nil {
				resp.Body.Close()
			}
		})
		require.Len(t, spans, 1)
		return spans[0].Context.Service.Target
	}

	// No X-Found-Handling-Cluster: no cluster name recorded.
	assert.Equal(t, &model.ServiceTargetSpanContext{Type: "elasticsearch"}, doRequest())
	assert.Equal(t, []string{"/_search"}, requestPaths)
	requestPaths = nil

	// X-Found-Handling-Cluster specified.
	foundHandlingCluster = "found_handling_cluster"
	assert.Equal(t, &model.ServiceTargetSpanContext{
		Type: "elasticsearch",
		Name: foundHandlingCluster,
	}, doRequest())
	assert.Equal(t, []string{"/_search"}, requestPaths)
	requestPaths = nil

	// Search request returned an error: no cluster name recorded.
	requestError = errors.New("boom")
	assert.Equal(t, &model.ServiceTargetSpanContext{Type: "elasticsearch"}, doRequest())
	assert.Equal(t, []string{"/_search"}, requestPaths)
	requestPaths = nil
}

func TestTraceHeaders(t *testing.T) {
	headers := make(map[string]string)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		for k, vs := range req.Header {
			headers[k] = strings.Join(vs, " ")
		}
	}))
	defer server.Close()
	client := &http.Client{Transport: apmelasticsearch.WrapRoundTripper(http.DefaultTransport)}

	req, err := http.NewRequest("GET", server.URL, nil)
	require.NoError(t, err)

	_, _, _ = apmtest.WithTransaction(func(ctx context.Context) {
		_, err := client.Do(req.WithContext(ctx))
		assert.NoError(t, err)
	})

	assert.Contains(t, headers, apmhttp.ElasticTraceparentHeader)
	assert.Contains(t, headers, apmhttp.W3CTraceparentHeader)
	assert.Contains(t, headers, apmhttp.TracestateHeader)
}

func TestClientSpanDropped(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte(req.Header.Get("Traceparent")))
	}))
	defer server.Close()

	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	tracer.SetMaxSpans(1)
	tx := tracer.StartTransaction("name", "type")
	ctx := apm.ContextWithTransaction(context.Background(), tx)

	var responseBodies []string
	for i := 0; i < 2; i++ {
		body, err := doGET(ctx, server.URL)
		require.NoError(t, err)
		responseBodies = append(responseBodies, body)
	}

	tx.End()
	tracer.Flush(nil)
	payloads := transport.Payloads()
	require.Len(t, payloads.Spans, 1)
	transaction := payloads.Transactions[0]
	span := payloads.Spans[0] // for first request

	clientTraceContext, err := apmhttp.ParseTraceparentHeader(string(responseBodies[0]))
	require.NoError(t, err)
	assert.Equal(t, span.TraceID, model.TraceID(clientTraceContext.Trace))
	assert.Equal(t, span.ID, model.SpanID(clientTraceContext.Span))

	clientTraceContext, err = apmhttp.ParseTraceparentHeader(string(responseBodies[1]))
	require.NoError(t, err)
	assert.Equal(t, transaction.TraceID, model.TraceID(clientTraceContext.Trace))
	assert.Equal(t, transaction.ID, model.SpanID(clientTraceContext.Span))
}

func TestClientTransactionUnsampled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte(req.Header.Get("Traceparent")))
	}))
	defer server.Close()

	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()
	tracer.SetSampler(apm.NewRatioSampler(0)) // sample nothing

	tx := tracer.StartTransaction("name", "type")
	ctx := apm.ContextWithTransaction(context.Background(), tx)
	body, err := doGET(ctx, server.URL)
	require.NoError(t, err)

	tx.End()
	tracer.Flush(nil)

	payloads := transport.Payloads()
	require.Len(t, payloads.Transactions, 1)
	require.Len(t, payloads.Spans, 0)
	transaction := payloads.Transactions[0]

	clientTraceContext, err := apmhttp.ParseTraceparentHeader(string(body))
	require.NoError(t, err)
	assert.Equal(t, transaction.TraceID, model.TraceID(clientTraceContext.Trace))
	assert.Equal(t, transaction.ID, model.SpanID(clientTraceContext.Span))
}

func doGET(ctx context.Context, url string) (string, error) {
	client := &http.Client{Transport: apmelasticsearch.WrapRoundTripper(http.DefaultTransport)}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	return string(body), nil
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
