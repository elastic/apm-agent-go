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
	"net"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.elastic.co/apm/transport"
)

func TestInitDefault(t *testing.T) {
	var h recordingHandler
	server := httptest.NewServer(&h)
	defer server.Close()

	defer patchEnv("ELASTIC_APM_SERVER_URL", server.URL)()

	tr, err := transport.InitDefault()
	assert.NoError(t, err)
	assert.NotNil(t, tr)
	assert.Exactly(t, tr, transport.Default)

	err = tr.SendStream(context.Background(), strings.NewReader("request-body"))
	assert.NoError(t, err)
	assert.Len(t, h.requests, 1)
}

func TestInitDefaultDiscard(t *testing.T) {
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

	defer patchEnv("ELASTIC_APM_SERVER_URL", "")()

	tr, err := transport.InitDefault()
	assert.NoError(t, err)
	assert.NotNil(t, tr)
	assert.Exactly(t, tr, transport.Default)

	err = tr.SendStream(context.Background(), strings.NewReader("request-body"))
	assert.NoError(t, err)
	assert.Len(t, h.requests, 1)
}

func TestInitDefaultError(t *testing.T) {
	defer patchEnv("ELASTIC_APM_SERVER_URL", ":")()

	tr, initErr := transport.InitDefault()
	assert.Error(t, initErr)
	assert.NotNil(t, tr)
	assert.Exactly(t, tr, transport.Default)

	sendErr := tr.SendStream(context.Background(), strings.NewReader("request-body"))
	assert.Exactly(t, initErr, sendErr)
}
