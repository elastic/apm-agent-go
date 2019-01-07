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
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.elastic.co/apm/transport"
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

	transport, err := transport.NewHTTPTransport()
	assert.NoError(t, err)
	err = transport.SendStream(context.Background(), strings.NewReader(""))
	assert.NoError(t, err)
	assert.Len(t, h.requests, 1)
}

func TestHTTPTransportUserAgent(t *testing.T) {
	var h recordingHandler
	server := httptest.NewServer(&h)
	defer server.Close()
	defer patchEnv("ELASTIC_APM_SERVER_URLS", server.URL)()

	transport, err := transport.NewHTTPTransport()
	assert.NoError(t, err)
	err = transport.SendStream(context.Background(), strings.NewReader(""))
	assert.NoError(t, err)
	assert.Len(t, h.requests, 1)

	transport.SetUserAgent("foo")
	err = transport.SendStream(context.Background(), strings.NewReader(""))
	assert.NoError(t, err)
	assert.Len(t, h.requests, 2)

	assert.Regexp(t, "Go-http-client/.*", h.requests[0].UserAgent())
	assert.Equal(t, "foo", h.requests[1].UserAgent())
}

func TestHTTPTransportSecretToken(t *testing.T) {
	var h recordingHandler
	server := httptest.NewServer(&h)
	defer server.Close()
	defer patchEnv("ELASTIC_APM_SERVER_URLS", server.URL)()

	transport, err := transport.NewHTTPTransport()
	transport.SetSecretToken("hunter2")
	assert.NoError(t, err)
	transport.SendStream(context.Background(), strings.NewReader(""))

	assert.Len(t, h.requests, 1)
	assertAuthorization(t, h.requests[0], "hunter2")
}

func TestHTTPTransportEnvSecretToken(t *testing.T) {
	var h recordingHandler
	server := httptest.NewServer(&h)
	defer server.Close()
	defer patchEnv("ELASTIC_APM_SERVER_URLS", server.URL)()
	defer patchEnv("ELASTIC_APM_SECRET_TOKEN", "hunter2")()

	transport, err := transport.NewHTTPTransport()
	assert.NoError(t, err)
	transport.SendStream(context.Background(), strings.NewReader(""))

	assert.Len(t, h.requests, 1)
	assertAuthorization(t, h.requests[0], "hunter2")
}

func TestHTTPTransportNoSecretToken(t *testing.T) {
	var h recordingHandler
	transport, server := newHTTPTransport(t, &h)
	defer server.Close()

	transport.SendStream(context.Background(), strings.NewReader(""))

	assert.Len(t, h.requests, 1)
	assertAuthorization(t, h.requests[0], "")
}

func TestHTTPTransportTLS(t *testing.T) {
	var h recordingHandler
	server := httptest.NewUnstartedServer(&h)
	server.Config.ErrorLog = log.New(ioutil.Discard, "", 0)
	server.StartTLS()
	defer server.Close()
	defer patchEnv("ELASTIC_APM_SERVER_URLS", server.URL)()

	transport, err := transport.NewHTTPTransport()
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

func TestHTTPTransportEnvVerifyServerCert(t *testing.T) {
	var h recordingHandler
	server := httptest.NewTLSServer(&h)
	defer server.Close()
	defer patchEnv("ELASTIC_APM_SERVER_URLS", server.URL)()
	defer patchEnv("ELASTIC_APM_VERIFY_SERVER_CERT", "false")()

	transport, err := transport.NewHTTPTransport()
	assert.NoError(t, err)

	assert.NotNil(t, transport.Client)
	assert.IsType(t, &http.Transport{}, transport.Client.Transport)
	httpTransport := transport.Client.Transport.(*http.Transport)
	assert.NotNil(t, httpTransport.TLSClientConfig)
	assert.True(t, httpTransport.TLSClientConfig.InsecureSkipVerify)

	err = transport.SendStream(context.Background(), strings.NewReader(""))
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
	defer patchEnv("ELASTIC_APM_SERVER_URLS", server.URL)()

	transport, err := transport.NewHTTPTransport()
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
	defer patchEnv("ELASTIC_APM_SERVER_URLS", server.URL)()
	defer patchEnv("ELASTIC_APM_SERVER_TIMEOUT", "50ms")()

	before := time.Now()
	transport, err := transport.NewHTTPTransport()
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

func TestHTTPTransportServerFailover(t *testing.T) {
	defer patchEnv("ELASTIC_APM_VERIFY_SERVER_CERT", "false")()

	var hosts []string
	errorHandler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		hosts = append(hosts, req.Host)
		http.Error(w, "error-message", http.StatusInternalServerError)
	})
	server1 := httptest.NewServer(errorHandler)
	defer server1.Close()
	server2 := httptest.NewTLSServer(errorHandler)
	defer server2.Close()

	transport, err := transport.NewHTTPTransport()
	require.NoError(t, err)
	transport.SetServerURL(mustParseURL(server1.URL), mustParseURL(server2.URL))

	for i := 0; i < 4; i++ {
		err := transport.SendStream(context.Background(), strings.NewReader(""))
		assert.EqualError(t, err, "request failed with 500 Internal Server Error: error-message")
	}
	assert.Len(t, hosts, 4)

	// Each time SendStream returns an error, the transport should switch
	// to the next URL in the list. The list is shuffled so we only compare
	// the output values to each other, rather than to the original input.
	assert.NotEqual(t, hosts[0], hosts[1])
	assert.Equal(t, hosts[0], hosts[2])
	assert.Equal(t, hosts[1], hosts[3])
}

func TestHTTPTransportV2NotFound(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()

	transport, err := transport.NewHTTPTransport()
	require.NoError(t, err)
	transport.SetServerURL(mustParseURL(server.URL))

	err = transport.SendStream(context.Background(), strings.NewReader(""))
	assert.EqualError(t, err, fmt.Sprintf("request failed with 404 Not Found: %s/intake/v2/events not found (requires APM Server 6.5.0 or newer)", server.URL))
}

func TestHTTPTransportServerCert(t *testing.T) {
	var h recordingHandler
	server := httptest.NewUnstartedServer(&h)
	server.Config.ErrorLog = log.New(ioutil.Discard, "", 0)
	server.StartTLS()
	defer server.Close()
	defer patchEnv("ELASTIC_APM_SERVER_URLS", server.URL)()

	p := strings.NewReader("")

	newTransport := func() *transport.HTTPTransport {
		transport, err := transport.NewHTTPTransport()
		require.NoError(t, err)
		return transport
	}

	// SendStream should fail, because we haven't told the client about
	// the server certificate, nor disabled certificate verification.
	transport := newTransport()
	err := transport.SendStream(context.Background(), p)
	assert.Error(t, err)

	// Set a certificate that doesn't match, SendStream should still fail.
	defer patchEnv("ELASTIC_APM_SERVER_CERT", "./testdata/cert.pem")()
	transport = newTransport()
	err = transport.SendStream(context.Background(), p)
	assert.Error(t, err)

	f, err := ioutil.TempFile("", "apm-test")
	require.NoError(t, err)
	defer os.Remove(f.Name())
	defer f.Close()
	defer patchEnv("ELASTIC_APM_SERVER_CERT", f.Name())()

	// Reconfigure the transport so that it knows about the
	// CA certificate. We avoid using server.Client here, as
	// it is not available in older versions of Go.
	err = pem.Encode(f, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: server.TLS.Certificates[0].Certificate[0],
	})
	require.NoError(t, err)

	transport = newTransport()
	err = transport.SendStream(context.Background(), p)
	assert.NoError(t, err)
}

func TestHTTPTransportServerCertInvalid(t *testing.T) {
	f, err := ioutil.TempFile("", "apm-test")
	require.NoError(t, err)
	defer os.Remove(f.Name())
	defer f.Close()
	defer patchEnv("ELASTIC_APM_SERVER_CERT", f.Name())()

	fmt.Fprintln(f, `
-----BEGIN GARBAGE-----
garbage
-----END GARBAGE-----
`[1:])

	_, err = transport.NewHTTPTransport()
	assert.EqualError(t, err, fmt.Sprintf("failed to load certificate from %s: missing or invalid certificate", f.Name()))
}

func newHTTPTransport(t *testing.T, handler http.Handler) (*transport.HTTPTransport, *httptest.Server) {
	server := httptest.NewServer(handler)
	transport, err := transport.NewHTTPTransport()
	if !assert.NoError(t, err) {
		server.Close()
		t.FailNow()
	}
	transport.SetServerURL(mustParseURL(server.URL))
	return transport, server
}

func mustParseURL(s string) *url.URL {
	url, err := url.Parse(s)
	if err != nil {
		panic(err)
	}
	return url
}
