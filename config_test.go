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

package apm_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.elastic.co/apm"
	"go.elastic.co/apm/apmconfig"
	"go.elastic.co/apm/apmtest"
	"go.elastic.co/apm/transport"
)

func TestTracerCentralConfigUpdate(t *testing.T) {
	run := func(configKey, configValue string, isRemote func(*apm.Tracer) bool) {
		t.Run(configKey, func(t *testing.T) {
			response, _ := json.Marshal(map[string]string{configKey: configValue})
			testTracerCentralConfigUpdate(t, string(response), isRemote)
		})
	}
	run("transaction_sample_rate", "0", func(tracer *apm.Tracer) bool {
		return !tracer.StartTransaction("name", "type").Sampled()
	})
	run("transaction_max_spans", "0", func(tracer *apm.Tracer) bool {
		return tracer.StartTransaction("name", "type").StartSpan("name", "type", nil).Dropped()
	})
	run("capture_body", "all", func(tracer *apm.Tracer) bool {
		req, _ := http.NewRequest("POST", "/", strings.NewReader("..."))
		capturer := tracer.CaptureHTTPRequestBody(req)
		return capturer != nil
	})
}

func testTracerCentralConfigUpdate(t *testing.T, serverResponse string, isRemote func(*apm.Tracer) bool) {
	// This test server will respond initially with config that
	// disables sampling, and subsequently responses will indicate
	// lack of agent config, causing the agent to revert to local
	// config.
	responded := make(chan struct{})
	var responses int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		assert.Equal(t, "/config/v1/agents", req.URL.Path)
		w.Header().Set("Cache-Control", "max-age=1")
		if responses == 0 {
			w.Header().Set("Etag", `"foo"`)
			w.Write([]byte(serverResponse))
		} else {
			w.Header().Set("Etag", `"bar"`)
			w.Write([]byte(`{}`))
		}
		responses++
		select {
		case responded <- struct{}{}:
		case <-req.Context().Done():
		}
	}))
	defer server.Close()
	serverURL, _ := url.Parse(server.URL)

	httpTransport, err := transport.NewHTTPTransport()
	require.NoError(t, err)
	httpTransport.SetServerURL(serverURL)

	tracer, err := apm.NewTracerOptions(apm.TracerOptions{Transport: httpTransport})
	require.NoError(t, err)
	defer tracer.Close()

	// This test can be run in parallel with others after creating the tracer,
	// but not before, because we depend on NewTracerOptions picking up default
	// configuration.
	t.Parallel()

	tracer.SetLogger(apmtest.NewTestLogger(t))
	assert.False(t, isRemote(tracer))

	timeout := time.After(10 * time.Second)
	for {
		// There's a time window between the server responding
		// and the agent updating the config, so we spin until
		// it's updated.
		remote := isRemote(tracer)
		if !remote {
			break
		}
		select {
		case <-time.After(10 * time.Millisecond):
		case <-timeout:
			t.Fatal("timed out waiting for config update")
		}
	}
	// We wait for 2 responses so that we know we've unblocked the
	// 2nd response, and that the 2nd response has been fully consumed.
	for i := 0; i < 2; i++ {
		select {
		case <-responded:
		case <-timeout:
			t.Fatal("timed out waiting for config update")
		}
	}
	for {
		remote := isRemote(tracer)
		if !remote {
			break
		}
		select {
		case <-time.After(10 * time.Millisecond):
		case <-timeout:
			t.Fatal("timed out waiting for config to revert")
		}
	}
}

func TestTracerCentralConfigUpdateDisabled(t *testing.T) {
	responded := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		select {
		case responded <- struct{}{}:
		case <-req.Context().Done():
		}
	}))
	defer server.Close()

	os.Setenv("ELASTIC_APM_SERVER_URLS", server.URL)
	defer os.Unsetenv("ELASTIC_APM_SERVER_URLS")

	os.Setenv("ELASTIC_APM_CENTRAL_CONFIG", "false")
	defer os.Unsetenv("ELASTIC_APM_CENTRAL_CONFIG")

	httpTransport, err := transport.NewHTTPTransport()
	require.NoError(t, err)
	tracer, err := apm.NewTracerOptions(apm.TracerOptions{Transport: httpTransport})
	require.NoError(t, err)
	defer tracer.Close()
	tracer.SetLogger(apmtest.NewTestLogger(t))

	select {
	case <-responded:
		t.Fatal("unexpected config watcher response")
	case <-time.After(2 * time.Second):
	}
}

func TestTracerSetConfigWatcher(t *testing.T) {
	watcherClosed := make(chan struct{})
	watcherFunc := apmtest.WatchConfigFunc(func(ctx context.Context, params apmconfig.WatchParams) <-chan apmconfig.Change {
		changes := make(chan apmconfig.Change)
		go func() {
			<-ctx.Done()
			close(watcherClosed)
		}()
		return changes
	})

	tracer, err := apm.NewTracer("", "")
	require.NoError(t, err)
	defer tracer.Close()

	tracer.SetLogger(apmtest.NewTestLogger(t))
	tracer.SetConfigWatcher(watcherFunc)
	tracer.SetConfigWatcher(nil)
	select {
	case <-watcherClosed:
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for watcher context to be cancelled")
	}
}

func TestTracerConfigWatcherPrecedence(t *testing.T) {
	watcherFunc := apmtest.WatchConfigFunc(func(ctx context.Context, params apmconfig.WatchParams) <-chan apmconfig.Change {
		changes := make(chan apmconfig.Change)
		go func() {
			select {
			case changes <- apmconfig.Change{
				Attrs: map[string]string{"transaction_sample_rate": "0"},
			}:
			case <-ctx.Done():
			}
		}()
		return changes
	})
	tracer, err := apm.NewTracer("", "")
	require.NoError(t, err)
	defer tracer.Close()

	tracer.SetLogger(apmtest.NewTestLogger(t))
	tracer.SetConfigWatcher(watcherFunc)
	timeout := time.After(10 * time.Second)
	for {
		sampled := tracer.StartTransaction("name", "type").Sampled()
		if !sampled {
			// Updated
			break
		}
		select {
		case <-time.After(10 * time.Millisecond):
		case <-timeout:
			t.Fatal("timed out waiting for config update")
		}
	}

	// Setting a sampler locally will have no effect while there is remote
	// configuration in place.
	tracer.SetSampler(apm.NewRatioSampler(1))
	sampled := tracer.StartTransaction("name", "type").Sampled()
	assert.False(t, sampled)

	// Disable remote config, which also reverts to local config.
	tracer.SetConfigWatcher(nil)
	for {
		sampled := tracer.StartTransaction("name", "type").Sampled()
		if sampled {
			// Reverted
			break
		}
		select {
		case <-time.After(10 * time.Millisecond):
		case <-timeout:
			t.Fatal("timed out waiting for config to revert to local")
		}
	}
}
