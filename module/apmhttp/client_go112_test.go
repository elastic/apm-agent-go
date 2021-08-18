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

//go:build go1.12
// +build go1.12

package apmhttp_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.elastic.co/apm/module/apmhttp"
)

func TestClientCloseIdleConnections(t *testing.T) {
	var closed bool
	transport := &idleConnectionsCloser{
		RoundTripper: http.DefaultTransport,
		closeIdleConnections: func() {
			closed = true
		},
	}
	client := &http.Client{
		Transport: apmhttp.WrapRoundTripper(transport),
	}
	client.CloseIdleConnections()
	assert.True(t, closed)
}

type idleConnectionsCloser struct {
	http.RoundTripper
	closeIdleConnections func()
}

func (c *idleConnectionsCloser) CloseIdleConnections() {
	c.closeIdleConnections()
}
