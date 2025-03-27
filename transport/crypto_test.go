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
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.elastic.co/apm/v2/transport"
)

func TestHTTPTransportEnvVerifyServerCert(t *testing.T) {
	var h recordingHandler
	server := httptest.NewTLSServer(&h)
	defer server.Close()
	defer patchEnv("ELASTIC_APM_SERVER_URL", server.URL)()
	defer patchEnv("ELASTIC_APM_VERIFY_SERVER_CERT", "false")()

	transport, err := transport.NewHTTPTransport(transport.HTTPTransportOptions{})
	assert.NoError(t, err)

	assert.NotNil(t, transport.Client)
	assert.IsType(t, &http.Transport{}, transport.Client.Transport)
	httpTransport := transport.Client.Transport.(*http.Transport)
	assert.NotNil(t, httpTransport.TLSClientConfig)
	assert.True(t, httpTransport.TLSClientConfig.InsecureSkipVerify)

	err = transport.SendStream(context.Background(), strings.NewReader(""))
	assert.NoError(t, err)
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

	transport, err := transport.NewHTTPTransport(transport.HTTPTransportOptions{})
	require.NoError(t, err)
	err = transport.SetServerURL(mustParseURL(server1.URL), mustParseURL(server2.URL))
	require.NoError(t, err)

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

func TestHTTPTransportServerCert(t *testing.T) {
	var h recordingHandler
	server := httptest.NewUnstartedServer(&h)
	server.Config.ErrorLog = log.New(ioutil.Discard, "", 0)
	server.StartTLS()
	defer server.Close()
	defer patchEnv("ELASTIC_APM_SERVER_URL", server.URL)()

	p := strings.NewReader("")

	newTransport := func() *transport.HTTPTransport {
		transport, err := transport.NewHTTPTransport(transport.HTTPTransportOptions{})
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
	// server certificate. We avoid using server.Client here, as
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

	_, err = transport.NewHTTPTransport(transport.HTTPTransportOptions{})
	assert.EqualError(t, err, fmt.Sprintf("failed to load certificate from %s: missing or invalid certificate", f.Name()))
}

func TestHTTPTransportCACert(t *testing.T) {
	var h recordingHandler
	server := httptest.NewUnstartedServer(&h)
	server.Config.ErrorLog = log.New(ioutil.Discard, "", 0)
	server.StartTLS()
	defer server.Close()
	defer patchEnv("ELASTIC_APM_SERVER_URL", server.URL)()

	p := strings.NewReader("")

	// SendStream should fail, because we haven't told the client about
	// the server certificate, nor disabled certificate verification.
	trans, err := transport.NewHTTPTransport(transport.HTTPTransportOptions{})
	assert.NoError(t, err)
	assert.NotNil(t, trans)
	err = trans.SendStream(context.Background(), p)
	assert.Error(t, err)

	// Set the env var to a file that doesn't exist, should get an error
	defer patchEnv("ELASTIC_APM_SERVER_CA_CERT_FILE", "./testdata/file_that_doesnt_exist.pem")()
	trans, err = transport.NewHTTPTransport(transport.HTTPTransportOptions{})
	assert.Error(t, err)
	assert.Nil(t, trans)

	// Set the env var to a file that has no cert, should get an error
	f, err := ioutil.TempFile("", "apm-test-1")
	require.NoError(t, err)
	defer os.Remove(f.Name())
	defer f.Close()
	defer patchEnv("ELASTIC_APM_SERVER_CA_CERT_FILE", f.Name())()
	trans, err = transport.NewHTTPTransport(transport.HTTPTransportOptions{})
	assert.Error(t, err)
	assert.Nil(t, trans)

	// Set a certificate that doesn't match, SendStream should still fail
	defer patchEnv("ELASTIC_APM_SERVER_CA_CERT_FILE", "./testdata/cert.pem")()
	trans, err = transport.NewHTTPTransport(transport.HTTPTransportOptions{})
	assert.NoError(t, err)
	assert.NotNil(t, trans)
	err = trans.SendStream(context.Background(), p)
	assert.Error(t, err)

	f, err = ioutil.TempFile("", "apm-test-2")
	require.NoError(t, err)
	defer os.Remove(f.Name())
	defer f.Close()
	defer patchEnv("ELASTIC_APM_SERVER_CA_CERT_FILE", f.Name())()

	err = pem.Encode(f, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: server.TLS.Certificates[0].Certificate[0],
	})
	require.NoError(t, err)

	trans, err = transport.NewHTTPTransport(transport.HTTPTransportOptions{})
	assert.NoError(t, err)
	assert.NotNil(t, trans)
	err = trans.SendStream(context.Background(), p)
	assert.NoError(t, err)
}
