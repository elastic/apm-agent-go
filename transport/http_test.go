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
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.elastic.co/apm/v2/apmconfig"
	"go.elastic.co/apm/v2/transport"
)

func init() {
	// Don't let the environment influence tests.
	os.Unsetenv("ELASTIC_APM_SERVER_TIMEOUT")
	os.Unsetenv("ELASTIC_APM_SERVER_URLS")
	os.Unsetenv("ELASTIC_APM_SERVER_URL")
	os.Unsetenv("ELASTIC_APM_SECRET_TOKEN")
	os.Unsetenv("ELASTIC_APM_SERVER_CERT")
	os.Unsetenv("ELASTIC_APM_VERIFY_SERVER_CERT")
}

func TestNewHTTPTransportDefaultURL(t *testing.T) {
	var h recordingHandler
	server := httptest.NewUnstartedServer(&h)
	defer server.Close()

	lis, err := net.Listen("tcp", "localhost:8200")
	if err != nil {
		t.Skipf("cannot listen on default server address: %s", err)
	}
	server.Listener.Close()
	server.Listener = lis
	server.Start()

	transport, err := transport.NewHTTPTransport(transport.HTTPTransportOptions{})
	assert.NoError(t, err)
	err = transport.SendStream(context.Background(), strings.NewReader(""))
	assert.NoError(t, err)
	assert.Len(t, h.requests, 1)
}

func TestHTTPTransportUserAgent(t *testing.T) {
	var h recordingHandler
	server := httptest.NewServer(&h)
	defer server.Close()
	defer patchEnv("ELASTIC_APM_SERVER_URL", server.URL)()

	transport, err := transport.NewHTTPTransport(transport.HTTPTransportOptions{})
	assert.NoError(t, err)
	err = transport.SendStream(context.Background(), strings.NewReader(""))
	assert.NoError(t, err)
	assert.Len(t, h.requests, 1)

	transport.SetUserAgent("foo")
	err = transport.SendStream(context.Background(), strings.NewReader(""))
	assert.NoError(t, err)
	assert.Len(t, h.requests, 2)

	assert.Regexp(t, "apm-agent-go/.*", h.requests[0].UserAgent())
	assert.Equal(t, "foo", h.requests[1].UserAgent())
}

func TestHTTPTransportSecretToken(t *testing.T) {
	var h recordingHandler
	server := httptest.NewServer(&h)
	defer server.Close()
	defer patchEnv("ELASTIC_APM_SERVER_URL", server.URL)()

	transport, err := transport.NewHTTPTransport(transport.HTTPTransportOptions{})
	transport.SetSecretToken("hunter2")
	assert.NoError(t, err)
	transport.SendStream(context.Background(), strings.NewReader(""))

	assert.Len(t, h.requests, 1)
	assertAuthorization(t, h.requests[0], "Bearer hunter2")
}

func TestHTTPTransportEnvSecretToken(t *testing.T) {
	var h recordingHandler
	server := httptest.NewServer(&h)
	defer server.Close()
	defer patchEnv("ELASTIC_APM_SERVER_URL", server.URL)()
	defer patchEnv("ELASTIC_APM_SECRET_TOKEN", "hunter2")()

	transport, err := transport.NewHTTPTransport(transport.HTTPTransportOptions{})
	assert.NoError(t, err)
	transport.SendStream(context.Background(), strings.NewReader(""))

	assert.Len(t, h.requests, 1)
	assertAuthorization(t, h.requests[0], "Bearer hunter2")
}

func TestHTTPTransportAPIKey(t *testing.T) {
	var h recordingHandler
	server := httptest.NewServer(&h)
	defer server.Close()
	defer patchEnv("ELASTIC_APM_SERVER_URL", server.URL)()

	transport, err := transport.NewHTTPTransport(transport.HTTPTransportOptions{})
	transport.SetAPIKey("hunter2")
	assert.NoError(t, err)
	transport.SendStream(context.Background(), strings.NewReader(""))

	assert.Len(t, h.requests, 1)
	assertAuthorization(t, h.requests[0], "ApiKey hunter2")
}

func TestHTTPTransportEnvAPIKey(t *testing.T) {
	var h recordingHandler
	server := httptest.NewServer(&h)
	defer server.Close()
	defer patchEnv("ELASTIC_APM_SERVER_URL", server.URL)()
	defer patchEnv("ELASTIC_APM_API_KEY", "api_key_wins")()
	defer patchEnv("ELASTIC_APM_SECRET_TOKEN", "secret_token_loses")()

	transport, err := transport.NewHTTPTransport(transport.HTTPTransportOptions{})
	assert.NoError(t, err)
	transport.SendStream(context.Background(), strings.NewReader(""))

	assert.Len(t, h.requests, 1)
	assertAuthorization(t, h.requests[0], "ApiKey api_key_wins")
}

func TestHTTPTransportNoAuthorization(t *testing.T) {
	var h recordingHandler
	transport, server := newHTTPTransport(t, &h)
	defer server.Close()

	transport.SendStream(context.Background(), strings.NewReader(""))

	assert.Len(t, h.requests, 1)
	assertAuthorization(t, h.requests[0])
}

func TestHTTPTransportTLS(t *testing.T) {
	var h recordingHandler
	server := httptest.NewUnstartedServer(&h)
	server.Config.ErrorLog = log.New(ioutil.Discard, "", 0)
	server.StartTLS()
	defer server.Close()
	defer patchEnv("ELASTIC_APM_SERVER_URL", server.URL)()

	transport, err := transport.NewHTTPTransport(transport.HTTPTransportOptions{})
	assert.NoError(t, err)

	p := strings.NewReader("")

	// Send should fail, because we haven't told the client
	// about the CA certificate, nor configured it to disable
	// certificate verification.
	err = transport.SendStream(context.Background(), p)
	assert.Error(t, err)

	// Reconfigure the transport so that it knows about the
	// CA certificate. We avoid using server.Client here, as
	// it is not available in older versions of Go.
	certificate, err := x509.ParseCertificate(server.TLS.Certificates[0].Certificate[0])
	assert.NoError(t, err)
	certpool := x509.NewCertPool()
	certpool.AddCert(certificate)
	transport.Client.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs: certpool,
		},
	}
	err = transport.SendStream(context.Background(), p)
	assert.NoError(t, err)
}

func TestHTTPError(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		http.Error(w, "error-message", http.StatusInternalServerError)
	})
	tr, server := newHTTPTransport(t, h)
	defer server.Close()

	err := tr.SendStream(context.Background(), strings.NewReader(""))
	assert.EqualError(t, err, "request failed with 500 Internal Server Error: error-message")
}

func TestHTTPTransportContent(t *testing.T) {
	var h recordingHandler
	server := httptest.NewServer(&h)
	defer server.Close()
	defer patchEnv("ELASTIC_APM_SERVER_URL", server.URL)()

	transport, err := transport.NewHTTPTransport(transport.HTTPTransportOptions{})
	assert.NoError(t, err)
	transport.SendStream(context.Background(), strings.NewReader("request-body"))

	require.Len(t, h.requests, 1)
	assert.Equal(t, "deflate", h.requests[0].Header.Get("Content-Encoding"))
	assert.Equal(t, "application/x-ndjson", h.requests[0].Header.Get("Content-Type"))
}

func TestHTTPTransportServerTimeout(t *testing.T) {
	done := make(chan struct{})
	blockingHandler := func(w http.ResponseWriter, req *http.Request) { <-done }
	server := httptest.NewServer(http.HandlerFunc(blockingHandler))
	defer server.Close()
	defer close(done)
	defer patchEnv("ELASTIC_APM_SERVER_URL", server.URL)()
	defer patchEnv("ELASTIC_APM_SERVER_TIMEOUT", "50ms")()

	before := time.Now()
	transport, err := transport.NewHTTPTransport(transport.HTTPTransportOptions{})
	assert.NoError(t, err)
	err = transport.SendStream(context.Background(), strings.NewReader(""))
	taken := time.Since(before)
	assert.Error(t, err)
	err = errors.Cause(err)
	assert.Implements(t, new(net.Error), err)
	assert.True(t, err.(net.Error).Timeout())
	assert.Condition(t, func() bool {
		return taken >= 50*time.Millisecond
	})
}

func TestHTTPTransportV2NotFound(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()

	transport, err := transport.NewHTTPTransport(transport.HTTPTransportOptions{})
	require.NoError(t, err)
	transport.SetServerURL(mustParseURL(server.URL))

	err = transport.SendStream(context.Background(), strings.NewReader(""))
	assert.EqualError(t, err, fmt.Sprintf("request failed with 404 Not Found: %s/intake/v2/events not found (requires APM Server 6.5.0 or newer)", server.URL))
}

func TestHTTPTransportWatchConfig(t *testing.T) {
	type response struct {
		code         int
		cacheControl string
		etag         string
		body         string
	}
	responses := make(chan response, 1)

	var responseEtag string
	transport, server := newHTTPTransport(t, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		var response response
		var ok bool
		select {
		case response, ok = <-responses:
			if !ok {
				w.WriteHeader(http.StatusTeapot)
				return
			}
		case <-time.After(10 * time.Millisecond):
			// This is necessary in case the previous config change
			// wasn't consumed before a new request was made. This
			// will return to the request loop.
			w.Header().Set("Cache-Control", "max-age=0")
			w.WriteHeader(http.StatusNotModified)
			return
		case <-req.Context().Done():
			return
		}
		ifNoneMatch := req.Header.Get("If-None-Match")
		if ifNoneMatch == "" {
			assert.Equal(t, "", responseEtag)
		} else {
			assert.Equal(t, responseEtag, ifNoneMatch)
		}
		if response.cacheControl != "" {
			w.Header().Set("Cache-Control", response.cacheControl)
		}
		if response.etag != "" {
			w.Header().Set("Etag", response.etag)
			responseEtag = response.etag
		}
		w.WriteHeader(response.code)
		w.Write([]byte(response.body))
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var watchParams apmconfig.WatchParams
	watchParams.Service.Name = "name"
	watchParams.Service.Environment = "env"
	changes := transport.WatchConfig(ctx, watchParams)
	require.NotNil(t, changes)

	responses <- response{code: 200, cacheControl: "max-age=0", etag: `"empty"`}
	assert.Equal(t, apmconfig.Change{Attrs: map[string]string{}}, <-changes)

	responses <- response{code: 200, cacheControl: "max-age=0", etag: `"foobar"`, body: `{"foo": "bar"}`}
	assert.Equal(t, apmconfig.Change{Attrs: map[string]string{"foo": "bar"}}, <-changes)

	responses <- response{code: 200, cacheControl: "max-age=0", etag: `"empty"`}
	assert.Equal(t, apmconfig.Change{Attrs: map[string]string{}}, <-changes)

	responses <- response{code: 304, cacheControl: "max-age=0"}
	// No change.

	responses <- response{code: 200, cacheControl: "max-age=0", etag: `"foobaz"`, body: `{"foo": "baz"}`}
	assert.Equal(t, apmconfig.Change{Attrs: map[string]string{"foo": "baz"}}, <-changes)

	responses <- response{code: 200, cacheControl: "max-age=0", etag: `"foobar"`, body: `{"foo": "bar"}`}
	assert.Equal(t, apmconfig.Change{Attrs: map[string]string{"foo": "bar"}}, <-changes)

	responses <- response{code: 403, cacheControl: "max-age=0"}
	// 403s are not reported.

	close(responses)
	if change := <-changes; assert.Error(t, change.Err) {
		assert.Equal(t, "request failed with 418 I'm a teapot", change.Err.Error())
	}
}

func TestHTTPTransportWatchConfigQueryParams(t *testing.T) {
	test := func(t *testing.T, serviceName, serviceEnvironment, expectedQuery string) {
		query, err := url.ParseQuery(expectedQuery)
		require.NoError(t, err)
		transport, server := newHTTPTransport(t, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			assert.Equal(t, query, req.URL.Query())
			w.WriteHeader(500)
		}))
		defer server.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var watchParams apmconfig.WatchParams
		watchParams.Service.Name = serviceName
		watchParams.Service.Environment = serviceEnvironment
		<-transport.WatchConfig(ctx, watchParams)
	}
	t.Run("name_only", func(t *testing.T) { test(t, "opbeans", "", "service.name=opbeans") })
	t.Run("name_and_env", func(t *testing.T) { test(t, "opbeans", "dev", "service.name=opbeans&service.environment=dev") })
	t.Run("name_empty", func(t *testing.T) { test(t, "", "dev", "service.name=&service.environment=dev") })
	t.Run("both_empty", func(t *testing.T) { test(t, "", "", "service.name=") })
}

func TestHTTPTransportWatchConfigContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	transport, server := newHTTPTransport(t, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		cancel() // cancel client-side request context
		<-req.Context().Done()
	}))
	defer server.Close()

	var watchParams apmconfig.WatchParams
	watchParams.Service.Name = "name"
	watchParams.Service.Environment = "env"
	changes := transport.WatchConfig(ctx, watchParams)
	require.NotNil(t, changes)

	_, ok := <-changes
	require.False(t, ok)
}

func TestNewHTTPTransportTrailingSlash(t *testing.T) {
	var h recordingHandler
	mux := http.NewServeMux()
	mux.Handle("/intake/v2/events", &h)
	transport, server := newHTTPTransport(t, mux)
	defer server.Close()

	transport.SetServerURL(mustParseURL(server.URL + "/"))

	err := transport.SendStream(context.Background(), strings.NewReader(""))
	assert.NoError(t, err)
	require.Len(t, h.requests, 1)
	assert.Equal(t, "POST", h.requests[0].Method)
	assert.Equal(t, "/intake/v2/events", h.requests[0].URL.Path)
}

func TestHTTPTransportSendProfile(t *testing.T) {
	metadata := "metadata"
	profile1 := "profile1"
	profile2 := "profile2"

	type part struct {
		formName string
		fileName string
		header   textproto.MIMEHeader
		content  string
	}

	var parts []part
	transport, server := newHTTPTransport(t, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		r, err := req.MultipartReader()
		if err != nil {
			panic(err)
		}
		for {
			p, err := r.NextPart()
			if err == io.EOF {
				break
			}
			content, err := ioutil.ReadAll(p)
			if err == io.EOF {
				panic(err)
			}
			parts = append(parts, part{
				formName: p.FormName(),
				fileName: p.FileName(),
				header:   p.Header,
				content:  string(content),
			})
		}
	}))
	defer server.Close()

	err := transport.SendProfile(
		context.Background(),
		strings.NewReader(metadata),
		strings.NewReader(profile1),
		strings.NewReader(profile2),
	)
	require.NoError(t, err)

	makeHeader := func(kv ...string) textproto.MIMEHeader {
		h := make(textproto.MIMEHeader)
		for i := 0; i < len(kv); i += 2 {
			h.Set(kv[i], kv[i+1])
		}
		return h
	}

	assert.Equal(t,
		[]part{{
			formName: "metadata",
			header: makeHeader(
				"Content-Disposition", `form-data; name="metadata"`,
				"Content-Type", "application/json",
			),
			content: "metadata",
		}, {
			formName: "profile",
			header: makeHeader(
				"Content-Disposition", `form-data; name="profile"`,
				"Content-Type", `application/x-protobuf; messageType="perftools.profiles.Profile"`,
			),
			content: "profile1",
		}, {
			formName: "profile",
			header: makeHeader(
				"Content-Disposition", `form-data; name="profile"`,
				"Content-Type", `application/x-protobuf; messageType="perftools.profiles.Profile"`,
			),
			content: "profile2",
		}},
		parts,
	)
}

func TestHTTPTransportOptionsValidation(t *testing.T) {
	validURL, err := url.Parse("http://localhost:8200")
	require.NoError(t, err)

	t.Run("valid", func(t *testing.T) {
		transport, err := transport.NewHTTPTransport(transport.HTTPTransportOptions{
			ServerURLs:    []*url.URL{validURL},
			ServerTimeout: 30 * time.Second,
		})
		assert.NoError(t, err)
		assert.NotNil(t, transport)
	})
	t.Run("invalid_timeout", func(t *testing.T) {
		transport, err := transport.NewHTTPTransport(transport.HTTPTransportOptions{
			ServerTimeout: -1,
		})
		assert.EqualError(t, err, "apm transport options: ServerTimeout must be greater or equal to 0")
		assert.Nil(t, transport)
	})
}

func TestHTTPTransportOptionsEmptyURL(t *testing.T) {
	var h recordingHandler
	server := httptest.NewUnstartedServer(&h)
	defer server.Close()

	lis, err := net.Listen("tcp", "localhost:8200")
	if err != nil {
		t.Skipf("cannot listen on default server address: %s", err)
	}
	server.Listener.Close()
	server.Listener = lis
	server.Start()

	transport, err := transport.NewHTTPTransport(transport.HTTPTransportOptions{})
	require.NoError(t, err)
	require.NotNil(t, transport)

	err = transport.SendStream(context.Background(), strings.NewReader(""))
	assert.NoError(t, err)
	assert.Len(t, h.requests, 1)
}

func TestHTTPTransportOptionsDefaults(t *testing.T) {
	validURL, err := url.Parse("http://localhost:8200")
	require.NoError(t, err)
	transport, err := transport.NewHTTPTransport(transport.HTTPTransportOptions{
		ServerURLs: []*url.URL{validURL},
	})
	assert.NoError(t, err)
	assert.Equal(t, transport.Client.Timeout, 30*time.Second)
}

func TestSetServerURL(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		validURL, err := url.Parse("http://localhost:8200")
		require.NoError(t, err)
		transport, err := transport.NewHTTPTransport(transport.HTTPTransportOptions{
			ServerURLs: []*url.URL{validURL},
		})
		anotherURL, err := url.Parse("http://somethingelse:8200")
		require.NoError(t, err)

		err = transport.SetServerURL(anotherURL)
		require.NoError(t, err)
	})
	t.Run("invalid", func(t *testing.T) {
		validURL, err := url.Parse("http://localhost:8200")
		require.NoError(t, err)
		transport, err := transport.NewHTTPTransport(transport.HTTPTransportOptions{
			ServerURLs: []*url.URL{validURL},
		})

		err = transport.SetServerURL()
		require.EqualError(t, err, "SetServerURL expects at least one URL")
	})
}

func TestMajorServerVersion(t *testing.T) {
	newTransport := func(t *testing.T, u string) *transport.HTTPTransport {
		validURL, err := url.Parse(u)
		require.NoError(t, err)
		transport, err := transport.NewHTTPTransport(transport.HTTPTransportOptions{
			ServerURLs: []*url.URL{validURL},
		})
		require.NoError(t, err)
		return transport
	}

	t.Run("failure", func(t *testing.T) {
		var count uint32
		srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
			switch atomic.LoadUint32(&count) {
			case 0:
				rw.WriteHeader(200)
				rw.Write([]byte(`invalid json`))
			case 1:
				rw.WriteHeader(200)
				rw.Write([]byte(`{"version":"7.17.0"}`))
			default:
				http.Error(rw, `{"ok":false,"message":"The instance rejected the connection."}`, 502)
			}
			atomic.AddUint32(&count, 1)
		}))
		defer srv.Close()

		transport := newTransport(t, srv.URL)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		version := transport.MajorServerVersion(ctx, true)
		assert.Zero(t, version)

		version = transport.MajorServerVersion(ctx, true)
		assert.Equal(t, uint32(7), version)

		// Verifies that the cache has been invalidated when the server returns
		// an error.
		transport.SendStream(ctx, strings.NewReader("{}"))
		version = transport.MajorServerVersion(ctx, false)
		assert.Zero(t, version)
	})
	t.Run("failure_timeout", func(t *testing.T) {
		var count uint32
		wait := make(chan struct{})
		srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
			c := atomic.LoadUint32(&count)
			atomic.AddUint32(&count, 1)
			if c == 0 {
				<-wait
				return
			}
			rw.WriteHeader(200)
			rw.Write([]byte(`{"version":"7.16.3"}`))
		}))
		defer srv.Close()

		transport := newTransport(t, srv.URL)
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		version := transport.MajorServerVersion(ctx, true)
		close(wait)
		assert.Zero(t, version)
		assert.Error(t, ctx.Err())

		version = transport.MajorServerVersion(context.Background(), true)
		assert.Equal(t, uint32(2), count, "count == 1 means that the first request context was cancelled before the http test server received it")
		assert.Equal(t, uint32(7), version)
	})
	t.Run("success", func(t *testing.T) {
		var count uint32
		srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/", r.URL.Path)
			rw.WriteHeader(200)
			if atomic.LoadUint32(&count) > 0 {
				rw.Write([]byte(`{"version":"8.1.0"}`))
			} else {
				rw.Write([]byte(`{"version":"8.0.0"}`))
			}
			atomic.AddUint32(&count, 1)
		}))
		defer srv.Close()

		transport := newTransport(t, srv.URL)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		// Run GetVersion a few times and ensure that the same version is
		// returned on subsequent calls
		for i := 0; i < 5; i++ {
			version := transport.MajorServerVersion(ctx, true)
			assert.Equal(t, uint32(8), version, fmt.Sprintf("iteration %d", i))
		}
	})
	t.Run("concurrent", func(t *testing.T) {
		var count uint32
		srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			rw.WriteHeader(200)
			if atomic.LoadUint32(&count) > 0 {
				rw.Write([]byte(`{"version":"8.1.0"}`))
			} else {
				rw.Write([]byte(`{"version":"8.0.0"}`))
			}
			atomic.AddUint32(&count, 1)
		}))
		defer srv.Close()

		transport := newTransport(t, srv.URL)
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		// Run GetVersion a few times and ensure that the same version is
		// returned on subsequent calls
		var wg sync.WaitGroup
		iterations := 5
		wg.Add(iterations)
		for i := 0; i < iterations; i++ {
			go func(i int) {
				version := transport.MajorServerVersion(ctx, true)
				assert.Equal(t, uint32(8), version, fmt.Sprintf("iteration %d", i))
				wg.Done()
			}(i)
		}
		wg.Wait()
	})
}

func newHTTPTransport(t *testing.T, handler http.Handler) (*transport.HTTPTransport, *httptest.Server) {
	server := httptest.NewServer(handler)
	transport, err := transport.NewHTTPTransport(transport.HTTPTransportOptions{
		ServerURLs: []*url.URL{mustParseURL(server.URL)},
	})
	if !assert.NoError(t, err) {
		server.Close()
		t.FailNow()
	}
	return transport, server
}

func mustParseURL(s string) *url.URL {
	url, err := url.Parse(s)
	if err != nil {
		panic(err)
	}
	return url
}
