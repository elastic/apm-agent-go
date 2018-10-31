package apm_test

import (
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.elastic.co/apm"
	"go.elastic.co/apm/model"
	"go.elastic.co/apm/module/apmhttp"
	"go.elastic.co/apm/transport"
	"go.elastic.co/apm/transport/transporttest"
)

func TestTracerRequestTimeEnv(t *testing.T) {
	os.Setenv("ELASTIC_APM_API_REQUEST_TIME", "1s")
	defer os.Unsetenv("ELASTIC_APM_API_REQUEST_TIME")

	requestHandled := make(chan struct{}, 1)
	var serverStart, serverEnd time.Time
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		serverStart = time.Now()
		io.Copy(ioutil.Discard, req.Body)
		serverEnd = time.Now()
		requestHandled <- struct{}{}
	}))
	defer server.Close()

	os.Setenv("ELASTIC_APM_SERVER_URLS", server.URL)
	defer os.Unsetenv("ELASTIC_APM_SERVER_URLS")

	tracer, err := apm.NewTracer("tracer_testing", "")
	require.NoError(t, err)
	defer tracer.Close()
	httpTransport, err := transport.NewHTTPTransport()
	require.NoError(t, err)
	tracer.Transport = httpTransport

	clientStart := time.Now()
	for i := 0; i < 1000; i++ {
		tracer.StartTransaction("name", "type").End()
		// Yield to the tracer for more predictable timing.
		runtime.Gosched()
	}
	<-requestHandled
	clientEnd := time.Now()
	assert.WithinDuration(t, clientStart.Add(time.Second), clientEnd, 100*time.Millisecond)
	assert.WithinDuration(t, clientStart, serverStart, 100*time.Millisecond)
	assert.WithinDuration(t, clientEnd, serverEnd, 100*time.Millisecond)
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
	assert.EqualError(t, err, "invalid ELASTIC_APM_TRANSACTION_SAMPLE_RATE value 2.0: out of range [0,1.0]")
}

func testTracerTransactionRateEnv(t *testing.T, envValue string, ratio float64) {
	os.Setenv("ELASTIC_APM_TRANSACTION_SAMPLE_RATE", envValue)
	defer os.Unsetenv("ELASTIC_APM_TRANSACTION_SAMPLE_RATE")

	tracer, err := apm.NewTracer("tracer_testing", "")
	require.NoError(t, err)
	defer tracer.Close()
	tracer.Transport = transporttest.Discard

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

	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "http://server.testing/", nil)
	req.AddCookie(&http.Cookie{Name: "secret", Value: "top"})
	h := apmhttp.Wrap(http.NotFoundHandler(), apmhttp.WithTracer(tracer))
	h.ServeHTTP(w, req)
	tracer.Flush(nil)

	tx := transport.Payloads().Transactions[0]
	assert.Equal(t, tx.Context.Request.Cookies, model.Cookies{
		{Name: "secret", Value: expect},
	})
}

func TestTracerServiceNameEnvSanitizationSpecified(t *testing.T) {
	testTracerServiceNameSanitization(
		t, "foo_bar", "ELASTIC_APM_SERVICE_NAME=foo!bar",
	)
}

func TestTracerServiceNameEnvSanitizationExecutableName(t *testing.T) {
	testTracerServiceNameSanitization(
		t, "apm_test", // .test -> _test
	)
}

func testTracerServiceNameSanitization(t *testing.T, sanitizedServiceName string, env ...string) {
	if os.Getenv("_INSIDE_TEST") != "1" {
		cmd := exec.Command(os.Args[0], "-test.run=^"+t.Name()+"$")
		cmd.Env = append(os.Environ(), "_INSIDE_TEST=1")
		cmd.Env = append(cmd.Env, env...)
		output, err := cmd.CombinedOutput()
		if !assert.NoError(t, err) {
			t.Logf("output:\n%s", output)
		}
		return
	}

	var transport transporttest.RecorderTransport
	tracer, _ := apm.NewTracer("", "")
	tracer.Transport = &transport
	defer tracer.Close()

	tx := tracer.StartTransaction("name", "type")
	tx.End()
	tracer.Flush(nil)

	_, _, service := transport.Metadata()
	assert.Equal(t, sanitizedServiceName, service.Name)
}

func TestTracerCaptureBodyEnv(t *testing.T) {
	test := func(t *testing.T, envValue string) {
		testTracerCaptureBodyEnv(t, envValue, true)
	}
	t.Run("all", func(t *testing.T) { test(t, "all") })
	t.Run("transactions", func(t *testing.T) { test(t, "transactions") })
}

func TestTracerCaptureBodyEnvOff(t *testing.T) {
	test := func(t *testing.T, envValue string) {
		testTracerCaptureBodyEnv(t, envValue, false)
	}
	t.Run("unset", func(t *testing.T) { test(t, "") })
	t.Run("off", func(t *testing.T) { test(t, "off") })
	t.Run("invalid", func(t *testing.T) { test(t, "invalid") })
}

func testTracerCaptureBodyEnv(t *testing.T, envValue string, expectBody bool) {
	if os.Getenv("_INSIDE_TEST") != "1" {
		cmd := exec.Command(os.Args[0], "-test.run=^"+t.Name()+"$")
		cmd.Env = append(os.Environ(), "_INSIDE_TEST=1")
		cmd.Env = append(cmd.Env, "ELASTIC_APM_CAPTURE_BODY="+envValue)
		if expectBody {
			cmd.Env = append(cmd.Env, "_EXPECT_BODY=1")
		}
		err := cmd.Run()
		assert.NoError(t, err)
		return
	}

	var transport transporttest.RecorderTransport
	tracer := apm.DefaultTracer
	tracer.Transport = &transport

	req, _ := http.NewRequest("GET", "/", strings.NewReader("foo_bar"))
	body := tracer.CaptureHTTPRequestBody(req)
	tx := tracer.StartTransaction("name", "type")
	tx.Context.SetHTTPRequest(req)
	tx.Context.SetHTTPRequestBody(body)
	tx.End()
	tracer.Flush(nil)

	out := transport.Payloads().Transactions[0]
	if os.Getenv("_EXPECT_BODY") == "1" {
		assert.NotNil(t, out.Context.Request.Body)
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

func TestTracerActive(t *testing.T) {
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

func TestTracerActiveInvalid(t *testing.T) {
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

	_, _, service := transport.Metadata()
	assert.Equal(t, "friendly", service.Environment)
}
