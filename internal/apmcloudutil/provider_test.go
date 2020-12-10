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

package apmcloudutil

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.elastic.co/apm/model"
)

func TestAutoProviderAllFail(t *testing.T) {
	var out model.Cloud
	var logger testLogger
	client := &http.Client{Transport: newTargetedRoundTripper("", "testing.invalid")}
	assert.False(t, Auto.getCloudMetadata(context.Background(), client, &logger, &out))
	assert.Zero(t, logger)
}

func TestNone(t *testing.T) {
	type wrappedRoundTripper struct {
		http.RoundTripper
	}
	var out model.Cloud
	var logger testLogger
	client := &http.Client{Transport: &wrappedRoundTripper{nil /*panics if called*/}}
	assert.False(t, None.getCloudMetadata(context.Background(), client, &logger, &out))
	assert.Zero(t, logger)
}

// newTargetedRoundTripper returns a net/http.RoundTripper which wraps net/http.DefaultTransport,
// rewriting requests for host to be sent to target, and causing all other requests to fail.
func newTargetedRoundTripper(host, target string) http.RoundTripper {
	return &targetedRoundTripper{
		Transport: http.DefaultTransport.(*http.Transport),
		host:      host,
		target:    target,
	}
}

type targetedRoundTripper struct {
	*http.Transport
	host   string
	target string
}

func (rt *targetedRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Host != rt.host {
		port, err := strconv.Atoi(req.URL.Port())
		if err != nil {
			port = 80
		}
		return nil, &net.OpError{
			Op:   "dial",
			Net:  "tcp",
			Addr: &net.TCPAddr{IP: net.ParseIP(req.URL.Hostname()), Port: port},
			Err:  errors.New("connect: no route to host"),
		}
	}
	req.URL.Host = rt.target
	return rt.Transport.RoundTrip(req)
}

type testLogger struct {
	Logger // panic on unexpected method calls
	buf    bytes.Buffer
}

func (tl *testLogger) Warningf(format string, args ...interface{}) {
	fmt.Fprintf(&tl.buf, "[warning] "+format, args...)
}
