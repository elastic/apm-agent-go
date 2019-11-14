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
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.elastic.co/apm"
	"go.elastic.co/apm/apmtest"
	"go.elastic.co/apm/model"
	"go.elastic.co/apm/transport"
	"go.elastic.co/apm/transport/transporttest"
)

func TestTracerRequestTimeEnv(t *testing.T) {
	os.Setenv("ELASTIC_APM_API_REQUEST_TIME", "1s")
	defer os.Unsetenv("ELASTIC_APM_API_REQUEST_TIME")

	requestHandled := make(chan struct{}, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/intake/v2/events" {
			return
		}
		io.Copy(ioutil.Discard, req.Body)
		requestHandled <- struct{}{}
	}))
	defer server.Close()

	os.Setenv("ELASTIC_APM_SERVER_URLS", server.URL)
	defer os.Unsetenv("ELASTIC_APM_SERVER_URLS")

	httpTransport, err := transport.NewHTTPTransport()
	require.NoError(t, err)
	tracer, err := apm.NewTracerOptions(apm.TracerOptions{
		ServiceName: "tracer_testing",
		Transport:   httpTransport,
	})
	require.NoError(t, err)
	defer tracer.Close()

	clientStart := time.Now()
	tracer.StartTransaction("name", "type").End()
	<-requestHandled
	clientEnd := time.Now()

	assert.WithinDuration(t, clientStart.Add(time.Second), clientEnd, 200*time.Millisecond)
}

func TestTracerRequestTimeEnvInvalid(t *testing.T) {
	t.Run("invalid_duration", func(t *testing.T) {
		os.Setenv("ELASTIC_APM_API_REQUEST_TIME", "aeon")
		defer os.Unsetenv("ELASTIC_APM_API_REQUEST_TIME")
		_, err := apm.NewTracer("tracer_testing", "")
		assert.EqualError(t, err, "failed to parse ELASTIC_APM_API_REQUEST_TIME: invalid duration aeon")
	})
	t.Run("missing_suffix", func(t *testing.T) {
		os.Setenv("ELASTIC_APM_API_REQUEST_TIME", "1")
		defer os.Unsetenv("ELASTIC_APM_API_REQUEST_TIME")
		_, err := apm.NewTracer("tracer_testing", "")
		assert.EqualError(t, err, "failed to parse ELASTIC_APM_API_REQUEST_TIME: missing unit in duration 1 (allowed units: ms, s, m)")
	})
}

func TestTracerRequestSizeEnvInvalid(t *testing.T) {
	t.Run("too_small", func(t *testing.T) {
		os.Setenv("ELASTIC_APM_API_REQUEST_SIZE", "1B")
		defer os.Unsetenv("ELASTIC_APM_API_REQUEST_SIZE")
		_, err := apm.NewTracer("tracer_testing", "")
		assert.EqualError(t, err, "ELASTIC_APM_API_REQUEST_SIZE must be at least 1KB and less than 5MB, got 1B")
	})
	t.Run("too_large", func(t *testing.T) {
		os.Setenv("ELASTIC_APM_API_REQUEST_SIZE", "500GB")
		defer os.Unsetenv("ELASTIC_APM_API_REQUEST_SIZE")
		_, err := apm.NewTracer("tracer_testing", "")
		assert.EqualError(t, err, "ELASTIC_APM_API_REQUEST_SIZE must be at least 1KB and less than 5MB, got 500GB")
	})
}

func TestTracerBufferSizeEnvInvalid(t *testing.T) {
	t.Run("too_small", func(t *testing.T) {
		os.Setenv("ELASTIC_APM_API_BUFFER_SIZE", "1B")
		defer os.Unsetenv("ELASTIC_APM_API_BUFFER_SIZE")
		_, err := apm.NewTracer("tracer_testing", "")
		assert.EqualError(t, err, "ELASTIC_APM_API_BUFFER_SIZE must be at least 10KB and less than 100MB, got 1B")
	})
	t.Run("too_large", func(t *testing.T) {
		os.Setenv("ELASTIC_APM_API_BUFFER_SIZE", "500GB")
		defer os.Unsetenv("ELASTIC_APM_API_BUFFER_SIZE")
		_, err := apm.NewTracer("tracer_testing", "")
		assert.EqualError(t, err, "ELASTIC_APM_API_BUFFER_SIZE must be at least 10KB and less than 100MB, got 500GB")
	})
}

func TestTracerMetricsBufferSizeEnvInvalid(t *testing.T) {
	t.Run("too_small", func(t *testing.T) {
		os.Setenv("ELASTIC_APM_METRICS_BUFFER_SIZE", "1B")
		defer os.Unsetenv("ELASTIC_APM_METRICS_BUFFER_SIZE")
		_, err := apm.NewTracer("tracer_testing", "")
		assert.EqualError(t, err, "ELASTIC_APM_METRICS_BUFFER_SIZE must be at least 10KB and less than 100MB, got 1B")
	})
	t.Run("too_large", func(t *testing.T) {
		os.Setenv("ELASTIC_APM_METRICS_BUFFER_SIZE", "500GB")
		defer os.Unsetenv("ELASTIC_APM_METRICS_BUFFER_SIZE")
		_, err := apm.NewTracer("tracer_testing", "")
		assert.EqualError(t, err, "ELASTIC_APM_METRICS_BUFFER_SIZE must be at least 10KB and less than 100MB, got 500GB")
	})
}

func TestTracerTransactionRateEnv(t *testing.T) {
	t.Run("0.5", func(t *testing.T) {
		testTracerTransactionRateEnv(t, "0.5", 0.5)
	})
	t.Run("0.75", func(t *testing.T) {
		testTracerTransactionRateEnv(t, "0.75", 0.75)
	})
	t.Run("1.0", func(t *testing.T) {
		testTracerTransactionRateEnv(t, "1.0", 1.0)
	})
}

func TestTracerTransactionRateEnvInvalid(t *testing.T) {
	os.Setenv("ELASTIC_APM_TRANSACTION_SAMPLE_RATE", "2.0")
	defer os.Unsetenv("ELASTIC_APM_TRANSACTION_SAMPLE_RATE")

	_, err := apm.NewTracer("tracer_testing", "")
	assert.EqualError(t, err, "invalid value for ELASTIC_APM_TRANSACTION_SAMPLE_RATE: 2.0 (out of range [0,1.0])")
}

func testTracerTransactionRateEnv(t *testing.T, envValue string, ratio float64) {
	os.Setenv("ELASTIC_APM_TRANSACTION_SAMPLE_RATE", envValue)
	defer os.Unsetenv("ELASTIC_APM_TRANSACTION_SAMPLE_RATE")

	tracer := apmtest.NewDiscardTracer()
	defer tracer.Close()

	const N = 10000
	var sampled int
	for i := 0; i < N; i++ {
		tx := tracer.StartTransaction("name", "type")
		if tx.Sampled() {
			sampled++
		}
		tx.End()
	}
	assert.InDelta(t, N*ratio, sampled, N*0.02) // allow 2% error
}

func TestTracerSanitizeFieldNamesEnv(t *testing.T) {
	testTracerSanitizeFieldNamesEnv(t, "secRet", "[REDACTED]")
	testTracerSanitizeFieldNamesEnv(t, "nada", "top")
}

func testTracerSanitizeFieldNamesEnv(t *testing.T, envValue, expect string) {
	os.Setenv("ELASTIC_APM_SANITIZE_FIELD_NAMES", envValue)
	defer os.Unsetenv("ELASTIC_APM_SANITIZE_FIELD_NAMES")

	req, _ := http.NewRequest("GET", "http://server.testing/", nil)
	req.AddCookie(&http.Cookie{Name: "secret", Value: "top"})

	tx, _, _ := apmtest.WithTransaction(func(ctx context.Context) {
		tx := apm.TransactionFromContext(ctx)
		tx.Context.SetHTTPRequest(req)
	})
	assert.Equal(t, tx.Context.Request.Cookies, model.Cookies{
		{Name: "secret", Value: expect},
	})
}

func TestTracerServiceNameEnvSanitizationSpecified(t *testing.T) {
	_, _, service, _ := getSubprocessMetadata(t, "ELASTIC_APM_SERVICE_NAME=foo!bar")
	assert.Equal(t, "foo_bar", service.Name)
}

func TestTracerServiceNameEnvSanitizationExecutableName(t *testing.T) {
	_, _, service, _ := getSubprocessMetadata(t)
	assert.Equal(t, "apm_test", service.Name) // .test -> _test
}

func TestTracerGlobalLabelsUnspecified(t *testing.T) {
	_, _, _, labels := getSubprocessMetadata(t)
	assert.Equal(t, model.StringMap{}, labels)
}

func TestTracerGlobalLabelsSpecified(t *testing.T) {
	_, _, _, labels := getSubprocessMetadata(t, "ELASTIC_APM_GLOBAL_LABELS=a=b,c = d")
	assert.Equal(t, model.StringMap{{Key: "a", Value: "b"}, {Key: "c", Value: "d"}}, labels)
}

func TestTracerGlobalLabelsIgnoreInvalid(t *testing.T) {
	_, _, _, labels := getSubprocessMetadata(t, "ELASTIC_APM_GLOBAL_LABELS=a,=,b==c,d=")
	assert.Equal(t, model.StringMap{{Key: "b", Value: "=c"}, {Key: "d", Value: ""}}, labels)
}

func TestTracerCaptureBodyEnv(t *testing.T) {
	t.Run("all", func(t *testing.T) { testTracerCaptureBodyEnv(t, "all", true) })
	t.Run("transactions", func(t *testing.T) { testTracerCaptureBodyEnv(t, "transactions", true) })
}

func TestTracerCaptureBodyEnvOff(t *testing.T) {
	t.Run("unset", func(t *testing.T) { testTracerCaptureBodyEnv(t, "", false) })
	t.Run("off", func(t *testing.T) { testTracerCaptureBodyEnv(t, "off", false) })
}

func TestTracerCaptureBodyEnvInvalid(t *testing.T) {
	os.Setenv("ELASTIC_APM_CAPTURE_BODY", "invalid")
	defer os.Unsetenv("ELASTIC_APM_CAPTURE_BODY")
	_, err := apm.NewTracer("", "")
	assert.EqualError(t, err, `invalid ELASTIC_APM_CAPTURE_BODY value "invalid"`)
}

func testTracerCaptureBodyEnv(t *testing.T, envValue string, expectBody bool) {
	os.Setenv("ELASTIC_APM_CAPTURE_BODY", envValue)
	defer os.Unsetenv("ELASTIC_APM_CAPTURE_BODY")

	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	req, _ := http.NewRequest("GET", "/", strings.NewReader("foo_bar"))
	body := tracer.CaptureHTTPRequestBody(req)
	tx := tracer.StartTransaction("name", "type")
	tx.Context.SetHTTPRequest(req)
	tx.Context.SetHTTPRequestBody(body)
	tx.End()
	tracer.Flush(nil)

	out := transport.Payloads().Transactions[0]
	if expectBody {
		require.NotNil(t, out.Context.Request.Body)
		assert.Equal(t, "foo_bar", out.Context.Request.Body.Raw)
	} else {
		assert.Nil(t, out.Context.Request.Body)
	}
}

func TestTracerSpanFramesMinDurationEnv(t *testing.T) {
	os.Setenv("ELASTIC_APM_SPAN_FRAMES_MIN_DURATION", "10ms")
	defer os.Unsetenv("ELASTIC_APM_SPAN_FRAMES_MIN_DURATION")

	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	tx := tracer.StartTransaction("name", "type")
	s := tx.StartSpan("name", "type", nil)
	s.Duration = 9 * time.Millisecond
	s.End()
	s = tx.StartSpan("name", "type", nil)
	s.Duration = 10 * time.Millisecond
	s.End()
	tx.End()
	tracer.Flush(nil)

	spans := transport.Payloads().Spans
	assert.Len(t, spans, 2)

	// Span 0 took only 9ms, so we don't set its stacktrace.
	assert.Nil(t, spans[0].Stacktrace)

	// Span 1 took the required 10ms, so we set its stacktrace.
	assert.NotNil(t, spans[1].Stacktrace)
}

func TestTracerSpanFramesMinDurationEnvInvalid(t *testing.T) {
	os.Setenv("ELASTIC_APM_SPAN_FRAMES_MIN_DURATION", "aeon")
	defer os.Unsetenv("ELASTIC_APM_SPAN_FRAMES_MIN_DURATION")

	_, err := apm.NewTracer("tracer_testing", "")
	assert.EqualError(t, err, "failed to parse ELASTIC_APM_SPAN_FRAMES_MIN_DURATION: invalid duration aeon")
}

func TestTracerStackTraceLimitEnv(t *testing.T) {
	os.Setenv("ELASTIC_APM_STACK_TRACE_LIMIT", "0")
	defer os.Unsetenv("ELASTIC_APM_STACK_TRACE_LIMIT")

	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	sendSpan := func() {
		tx := tracer.StartTransaction("name", "type")
		s := tx.StartSpan("name", "type", nil)
		s.Duration = time.Second
		s.End()
		tx.End()
	}

	sendSpan()
	tracer.SetStackTraceLimit(2)
	sendSpan()

	tracer.Flush(nil)
	spans := transport.Payloads().Spans
	require.Len(t, spans, 2)
	assert.Nil(t, spans[0].Stacktrace)
	assert.Len(t, spans[1].Stacktrace, 2)
}

func TestTracerStackTraceLimitEnvInvalid(t *testing.T) {
	os.Setenv("ELASTIC_APM_STACK_TRACE_LIMIT", "sky")
	defer os.Unsetenv("ELASTIC_APM_STACK_TRACE_LIMIT")

	_, err := apm.NewTracer("tracer_testing", "")
	assert.EqualError(t, err, "failed to parse ELASTIC_APM_STACK_TRACE_LIMIT: strconv.Atoi: parsing \"sky\": invalid syntax")
}

func TestTracerActiveEnv(t *testing.T) {
	os.Setenv("ELASTIC_APM_ACTIVE", "false")
	defer os.Unsetenv("ELASTIC_APM_ACTIVE")

	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()
	assert.False(t, tracer.Active())

	tx := tracer.StartTransaction("name", "type")
	tx.End()

	tracer.Flush(nil)
	assert.Zero(t, transport.Payloads())
}

func TestTracerActiveEnvInvalid(t *testing.T) {
	os.Setenv("ELASTIC_APM_ACTIVE", "yep")
	defer os.Unsetenv("ELASTIC_APM_ACTIVE")

	_, err := apm.NewTracer("tracer_testing", "")
	assert.EqualError(t, err, "failed to parse ELASTIC_APM_ACTIVE: strconv.ParseBool: parsing \"yep\": invalid syntax")
}

func TestTracerEnvironmentEnv(t *testing.T) {
	os.Setenv("ELASTIC_APM_ENVIRONMENT", "friendly")
	defer os.Unsetenv("ELASTIC_APM_ENVIRONMENT")

	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	tracer.StartTransaction("name", "type").End()
	tracer.Flush(nil)

	_, _, service, _ := transport.Metadata()
	assert.Equal(t, "friendly", service.Environment)
}

func TestTracerCaptureHeadersEnv(t *testing.T) {
	os.Setenv("ELASTIC_APM_CAPTURE_HEADERS", "false")
	defer os.Unsetenv("ELASTIC_APM_CAPTURE_HEADERS")

	tx, _, _ := apmtest.WithTransaction(func(ctx context.Context) {
		req, err := http.NewRequest("GET", "http://testing.invalid", nil)
		require.NoError(t, err)
		req.Header.Set("foo", "bar")
		respHeaders := make(http.Header)
		respHeaders.Set("baz", "qux")

		tx := apm.TransactionFromContext(ctx)
		tx.Context.SetHTTPRequest(req)
		tx.Context.SetHTTPResponseHeaders(respHeaders)
		tx.Context.SetHTTPStatusCode(202)
	})

	require.NotNil(t, tx.Context.Request)
	require.NotNil(t, tx.Context.Response)
	assert.Nil(t, tx.Context.Request.Headers)
	assert.Nil(t, tx.Context.Response.Headers)
}

func TestServiceNodeNameEnvSpecified(t *testing.T) {
	_, _, service, _ := getSubprocessMetadata(t, "ELASTIC_APM_SERVICE_NODE_NAME=foo_bar")
	assert.Equal(t, "foo_bar", service.Node.ConfiguredName)
}

func TestUseElasticTraceparentHeader(t *testing.T) {
	t.Run("default", func(t *testing.T) { testUseElasticTraceparentHeader(t, "", true) })
	t.Run("false", func(t *testing.T) { testUseElasticTraceparentHeader(t, "false", false) })
	t.Run("true", func(t *testing.T) { testUseElasticTraceparentHeader(t, "true", true) })
}

func testUseElasticTraceparentHeader(t *testing.T, envValue string, expectPropagate bool) {
	os.Setenv("ELASTIC_APM_USE_ELASTIC_TRACEPARENT_HEADER", envValue)
	defer os.Unsetenv("ELASTIC_APM_USE_ELASTIC_TRACEPARENT_HEADER")

	tracer := apmtest.NewDiscardTracer()
	defer tracer.Close()

	tx := tracer.StartTransaction("name", "type")
	propagate := tx.ShouldPropagateLegacyHeader()
	tx.Discard()
	assert.Equal(t, expectPropagate, propagate)
}
