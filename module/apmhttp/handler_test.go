package apmhttp_test

import (
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/http2"

	"github.com/elastic/apm-agent-go"
	"github.com/elastic/apm-agent-go/model"
	"github.com/elastic/apm-agent-go/module/apmhttp"
	"github.com/elastic/apm-agent-go/transport/transporttest"
)

func TestHandler(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	mux := http.NewServeMux()
	mux.Handle("/foo", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		w.Write([]byte("bar"))
	}))

	h := apmhttp.Wrap(mux, apmhttp.WithTracer(tracer))
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "http://server.testing/foo", nil)
	req.Header.Set("User-Agent", "apmhttp_test")
	req.RemoteAddr = "client.testing:1234"
	h.ServeHTTP(w, req)
	tracer.Flush(nil)

	payloads := transport.Payloads()
	transactions := payloads[0].Transactions()
	transaction := transactions[0]
	assert.Equal(t, "GET /foo", transaction.Name)
	assert.Equal(t, "request", transaction.Type)
	assert.Equal(t, "HTTP 4xx", transaction.Result)

	true_ := true
	assert.Equal(t, &model.Context{
		Request: &model.Request{
			Socket: &model.RequestSocket{
				RemoteAddress: "client.testing",
			},
			URL: model.URL{
				Full:     "http://server.testing/foo",
				Protocol: "http",
				Hostname: "server.testing",
				Path:     "/foo",
			},
			Method: "GET",
			Headers: &model.RequestHeaders{
				UserAgent: "apmhttp_test",
			},
			HTTPVersion: "1.1",
		},
		Response: &model.Response{
			StatusCode: 418,
			Finished:   &true_,
		},
	}, transaction.Context)
}

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
	req.Header.Set("X-Real-IP", "client.testing")
	resp, err := client.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	tracer.Flush(nil)

	payloads := transport.Payloads()
	transaction := payloads[0].Transactions()[0]

	true_ := true
	assert.Equal(t, &model.Context{
		Request: &model.Request{
			Socket: &model.RequestSocket{
				Encrypted:     true,
				RemoteAddress: "client.testing",
			},
			URL: model.URL{
				Full:     srv.URL + "/foo",
				Protocol: "https",
				Hostname: srvAddr.IP.String(),
				Port:     strconv.Itoa(srvAddr.Port),
				Path:     "/foo",
			},
			Method: "GET",
			Headers: &model.RequestHeaders{
				UserAgent: "Go-http-client/2.0",
			},
			HTTPVersion: "2.0",
		},
		Response: &model.Response{
			StatusCode: 418,
			Finished:   &true_,
		},
	}, transaction.Context)
}

func TestHandlerCaptureBodyRaw(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	tracer.SetCaptureBody(elasticapm.CaptureBodyTransactions)
	h := apmhttp.Wrap(
		http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}),
		apmhttp.WithTracer(tracer),
	)
	tx := testPostTransaction(h, tracer, transport, strings.NewReader("foo"))
	assert.Equal(t, &model.RequestBody{Raw: "foo"}, tx.Context.Request.Body)
}

func TestHandlerCaptureBodyForm(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	tracer.SetCaptureBody(elasticapm.CaptureBodyTransactions)
	h := apmhttp.Wrap(
		http.HandlerFunc(func(_ http.ResponseWriter, req *http.Request) {
			if err := req.ParseForm(); err != nil {
				panic(err)
			}
		}),
		apmhttp.WithTracer(tracer),
	)
	tx := testPostTransaction(h, tracer, transport, strings.NewReader("foo=bar&foo=baz"))
	assert.Equal(t, &model.RequestBody{
		Form: url.Values{
			"foo": []string{"bar", "baz"},
		},
	}, tx.Context.Request.Body)
}

func TestHandlerCaptureBodyError(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	tracer.SetCaptureBody(elasticapm.CaptureBodyAll)
	h := apmhttp.Wrap(
		http.HandlerFunc(panicHandler),
		apmhttp.WithTracer(tracer),
	)
	e := testPostError(h, tracer, transport, strings.NewReader("foo"))
	assert.Equal(t, &model.RequestBody{Raw: "foo"}, e.Context.Request.Body)
}

func TestHandlerCaptureBodyErrorIgnored(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	tracer.SetCaptureBody(elasticapm.CaptureBodyTransactions)
	h := apmhttp.Wrap(
		http.HandlerFunc(panicHandler),
		apmhttp.WithTracer(tracer),
	)
	e := testPostError(h, tracer, transport, strings.NewReader("foo"))
	assert.Nil(t, e.Context.Request.Body) // only capturing for transactions
}

func testPostTransaction(h http.Handler, tracer *elasticapm.Tracer, transport *transporttest.RecorderTransport, body io.Reader) model.Transaction {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "http://server.testing/foo", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h.ServeHTTP(w, req)
	tracer.Flush(nil)
	return transport.Payloads()[0].Transactions()[0]
}

func testPostError(h http.Handler, tracer *elasticapm.Tracer, transport *transporttest.RecorderTransport, body io.Reader) *model.Error {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "http://server.testing/foo", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h.ServeHTTP(w, req)
	tracer.Flush(nil)
	return transport.Payloads()[0].Errors()[0]
}

func TestHandlerRecovery(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	h := apmhttp.Wrap(
		http.HandlerFunc(panicHandler),
		apmhttp.WithTracer(tracer),
	)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "http://server.testing/foo", nil)
	h.ServeHTTP(w, req)
	tracer.Flush(nil)

	payloads := transport.Payloads()
	error0 := payloads[0].Errors()[0]
	transaction := payloads[1].Transactions()[0]

	assert.Equal(t, "panicHandler", error0.Culprit)
	assert.Equal(t, "foo", error0.Exception.Message)

	true_ := true
	assert.Equal(t, &model.Response{
		Finished:   &true_,
		StatusCode: 418,
	}, transaction.Context.Response)
}

func TestHandlerRequestIgnorer(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	h := apmhttp.Wrap(
		http.NotFoundHandler(),
		apmhttp.WithTracer(tracer),
		apmhttp.WithServerRequestIgnorer(func(*http.Request) bool {
			return true
		}),
	)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "http://server.testing/foo", nil)
	h.ServeHTTP(w, req)
	tracer.Flush(nil)
	assert.Empty(t, transport.Payloads())
}

func TestHandlerTraceparentHeader(t *testing.T) {
	os.Setenv("ELASTIC_APM_DISTRIBUTED_TRACING", "true")
	defer os.Unsetenv("ELASTIC_APM_DISTRIBUTED_TRACING")

	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	mux := http.NewServeMux()
	mux.Handle("/foo", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		w.Write([]byte("bar"))
	}))

	h := apmhttp.Wrap(mux, apmhttp.WithTracer(tracer))
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "http://server.testing/foo", nil)
	req.Header.Set("Elastic-Apm-Traceparent", "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01")
	req.Header.Set("User-Agent", "apmhttp_test")
	req.RemoteAddr = "client.testing:1234"
	h.ServeHTTP(w, req)
	tracer.Flush(nil)

	payloads := transport.Payloads()
	transactions := payloads[0].Transactions()
	transaction := transactions[0]
	assert.Equal(t, "0af7651916cd43dd8448eb211c80319c", elasticapm.TraceID(transaction.TraceID).String())
	assert.Equal(t, "b7ad6b7169203331", elasticapm.SpanID(transaction.ParentID).String())
	assert.NotZero(t, transaction.ID.SpanID)
	assert.Zero(t, transaction.ID.UUID)
	assert.Equal(t, "GET /foo", transaction.Name)
	assert.Equal(t, "request", transaction.Type)
	assert.Equal(t, "HTTP 4xx", transaction.Result)
}

func panicHandler(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusTeapot)
	panic("foo")
}
