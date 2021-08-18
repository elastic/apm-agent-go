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

//go:build go1.9
// +build go1.9

package apmhttp_test

import (
	"crypto/tls"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/http2"

	"go.elastic.co/apm/model"
	"go.elastic.co/apm/module/apmhttp"
	"go.elastic.co/apm/transport/transporttest"
)

func TestHandlerHTTP2(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	mux := http.NewServeMux()
	mux.Handle("/foo", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		w.Write([]byte("bar"))
	}))
	srv := httptest.NewUnstartedServer(apmhttp.Wrap(mux, apmhttp.WithTracer(tracer)))
	err := http2.ConfigureServer(srv.Config, nil)
	require.NoError(t, err)
	srv.TLS = srv.Config.TLSConfig
	srv.StartTLS()
	defer srv.Close()
	srvAddr := srv.Listener.Addr().(*net.TCPAddr)

	client := &http.Client{Transport: &http2.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}}
	require.NoError(t, err)

	req, _ := http.NewRequest("GET", srv.URL+"/foo", nil)
	resp, err := client.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	tracer.Flush(nil)

	payloads := transport.Payloads()
	transaction := payloads.Transactions[0]

	assert.Equal(t, &model.Context{
		Request: &model.Request{
			Socket: &model.RequestSocket{
				Encrypted: true,
				// 127.0.0.1 or ::1.
				RemoteAddress: srvAddr.IP.String(),
			},
			URL: model.URL{
				Full:     srv.URL + "/foo",
				Protocol: "https",
				Hostname: srvAddr.IP.String(),
				Port:     strconv.Itoa(srvAddr.Port),
				Path:     "/foo",
			},
			Method: "GET",
			Headers: model.Headers{{
				Key:    "Accept-Encoding",
				Values: []string{"gzip"},
			}, {
				Key:    "User-Agent",
				Values: []string{"Go-http-client/2.0"},
			}},
			HTTPVersion: "2.0",
		},
		Response: &model.Response{
			StatusCode: 418,
		},
	}, transaction.Context)
}
