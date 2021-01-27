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
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.elastic.co/apm"
	"go.elastic.co/apm/apmconfig"
	"go.elastic.co/apm/apmtest"
	"go.elastic.co/apm/internal/apmlog"
	"go.elastic.co/apm/transport"
	"go.elastic.co/apm/transport/transporttest"
)

func TestTracerCentralConfigUpdate(t *testing.T) {
	run := func(configKey, configValue string, isRemote func(*apmtest.RecordingTracer) bool) {
		t.Run(configKey, func(t *testing.T) {
			response, _ := json.Marshal(map[string]string{configKey: configValue})
			testTracerCentralConfigUpdate(t, apmtest.NewTestLogger(t), string(response), isRemote)
		})
	}
	run("transaction_sample_rate", "0", func(tracer *apmtest.RecordingTracer) bool {
		return !tracer.StartTransaction("name", "type").Sampled()
	})
	run("transaction_max_spans", "0", func(tracer *apmtest.RecordingTracer) bool {
		return tracer.StartTransaction("name", "type").StartSpan("name", "type", nil).Dropped()
	})
	run("capture_body", "all", func(tracer *apmtest.RecordingTracer) bool {
		req, _ := http.NewRequest("POST", "/", strings.NewReader("..."))
		capturer := tracer.CaptureHTTPRequestBody(req)
		return capturer != nil
	})
	run("recording", "false", func(tracer *apmtest.RecordingTracer) bool {
		return !tracer.Recording()
	})
	run("span_frames_min_duration", "1ms", func(tracer *apmtest.RecordingTracer) bool {
		tracer.ResetPayloads()

		tx := tracer.StartTransaction("name", "type")
		span := tx.StartSpan("name", "type", nil)
		span.Duration = 1 * time.Millisecond
		span.End()
		tx.End()

		tracer.Flush(nil)
		payloads := tracer.Payloads()
		assert.Len(t, payloads.Spans, 1)
		return len(payloads.Spans[0].Stacktrace) > 0
	})
	run("stack_trace_limit", "1", func(tracer *apmtest.RecordingTracer) bool {
		tracer.ResetPayloads()
		tracer.NewError(errors.New("boom")).Send()
		tracer.Flush(nil)
		payloads := tracer.Payloads()
		assert.Len(t, payloads.Errors, 1)
		return len(payloads.Errors[0].Exception.Stacktrace) == 1
	})
	run("sanitize_field_names", "secret", func(tracer *apmtest.RecordingTracer) bool {
		tracer.ResetPayloads()
		tracer.SetSanitizedFieldNames("not_secret")
		req, _ := http.NewRequest("GET", "http://server.testing/", nil)
		req.AddCookie(&http.Cookie{Name: "secret", Value: "top"})
		tx := tracer.StartTransaction("name", "type")
		tx.Context.SetHTTPRequest(req)
		tx.End()
		tracer.Flush(nil)
		payloads := tracer.Payloads()
		assert.Len(t, payloads.Transactions, 1)
		assert.Len(t, payloads.Transactions[0].Context.Request.Cookies, 1)
		return payloads.Transactions[0].Context.Request.Cookies[0].Value == "[REDACTED]"
	})
	t.Run("log_level", func(t *testing.T) {
		tempdir, err := ioutil.TempDir("", "apmtest_log_level")
		require.NoError(t, err)
		defer os.RemoveAll(tempdir)

		logfile := filepath.Join(tempdir, "apm.log")
		os.Setenv(apmlog.EnvLogFile, logfile)
		os.Setenv(apmlog.EnvLogLevel, "off")
		defer os.Unsetenv(apmlog.EnvLogFile)
		defer os.Unsetenv(apmlog.EnvLogLevel)
		apmlog.InitDefaultLogger()

		response, _ := json.Marshal(map[string]string{"log_level": "debug"})
		testTracerCentralConfigUpdate(t, apmlog.DefaultLogger, string(response), func(tracer *apmtest.RecordingTracer) bool {
			require.NoError(t, os.Truncate(logfile, 0))
			tracer.StartTransaction("name", "type").End()
			tracer.Flush(nil)
			log, err := ioutil.ReadFile(logfile)
			require.NoError(t, err)
			return len(log) > 0
		})
	})
	run("transaction_ignore_urls", "*", func(tracer *apmtest.RecordingTracer) bool {
		u, err := url.Parse("http://testing.invalid/")
		require.NoError(t, err)
		return tracer.IgnoredTransactionURL(u)
	})
}

func testTracerCentralConfigUpdate(t *testing.T, logger apm.Logger, serverResponse string, isRemote func(*apmtest.RecordingTracer) bool) {
	type response struct {
		etag string
		body string
	}
	responses := make(chan response)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		assert.Equal(t, "/config/v1/agents", req.URL.Path)
		w.Header().Set("Cache-Control", "max-age=1")
		select {
		case response := <-responses:
			w.Header().Set("Etag", strconv.Quote(response.etag))
			w.Write([]byte(response.body))
		case <-req.Context().Done():
		}
	}))
	defer server.Close()
	serverURL, _ := url.Parse(server.URL)

	httpTransport, err := transport.NewHTTPTransport()
	require.NoError(t, err)
	httpTransport.SetServerURL(serverURL)

	tracer := &apmtest.RecordingTracer{}
	var testTransport struct {
		apmconfig.Watcher
		*transporttest.RecorderTransport
	}
	testTransport.Watcher = httpTransport
	testTransport.RecorderTransport = &tracer.RecorderTransport

	tracer.Tracer, err = apm.NewTracerOptions(apm.TracerOptions{Transport: &testTransport})
	require.NoError(t, err)
	defer tracer.Tracer.Close()

	// This test can be run in parallel with others after creating the tracer,
	// but not before, because we depend on NewTracerOptions picking up default
	// configuration.
	t.Parallel()

	tracer.SetLogger(logger)
	assert.False(t, isRemote(tracer))

	timeout := time.After(10 * time.Second)

	// We each response payload twice, which causes us to block until
	// the first one is fully consumed.
	for i := 0; i < 2; i++ {
		select {
		case responses <- response{etag: "foo", body: serverResponse}:
		case <-timeout:
			t.Fatal("timed out waiting for config update")
		}
	}
	assert.True(t, isRemote(tracer))

	for i := 0; i < 2; i++ {
		select {
		case responses <- response{etag: "bar", body: "{}"}:
		case <-timeout:
			t.Fatal("timed out waiting for config update")
		}
	}
	assert.False(t, isRemote(tracer))
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

	os.Setenv("ELASTIC_APM_SERVER_URL", server.URL)
	defer os.Unsetenv("ELASTIC_APM_SERVER_URL")

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
