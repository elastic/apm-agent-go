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

	"github.com/elastic/apm-agent-go"
	"github.com/elastic/apm-agent-go/contrib/apmhttp"
	"github.com/elastic/apm-agent-go/transport/transporttest"
)

func TestHandler(t *testing.T) {
	tracer, transport := newRecordingTracer()
	defer tracer.Close()

	mux := http.NewServeMux()
	mux.Handle("/foo", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		w.Write([]byte("bar"))
	}))

	h := &apmhttp.Handler{
		Handler: mux,
		Tracer:  tracer,
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "http://server.testing/foo", nil)
	req.Header.Set("User-Agent", "apmhttp_test")
	req.RemoteAddr = "client.testing:1234"
	h.ServeHTTP(w, req)
	tracer.Flush(nil)

	payloads := transport.Payloads()
	assert.Len(t, payloads, 1)
	assert.Contains(t, payloads[0], "transactions")

	transactions := payloads[0]["transactions"].([]interface{})
	assert.Len(t, transactions, 1)
	transaction := transactions[0].(map[string]interface{})
	assert.Equal(t, "GET /foo", transaction["name"])
	assert.Equal(t, "request", transaction["type"])
	assert.Equal(t, "418", transaction["result"])

	context := transaction["context"].(map[string]interface{})
	assert.Equal(t, map[string]interface{}{
		"request": map[string]interface{}{
			"socket": map[string]interface{}{
				"remote_address": "client.testing",
			},
			"url": map[string]interface{}{
				"full":     "http://server.testing/foo",
				"protocol": "http",
				"hostname": "server.testing",
				"pathname": "/foo",
			},
			"method": "GET",
			"headers": map[string]interface{}{
				"user-agent": "apmhttp_test",
			},
			"http_version": "1.1",
		},
		"response": map[string]interface{}{
			"status_code":  float64(418),
			"headers_sent": true,
			"finished":     true,
		},
	}, context)
}

func TestHandlerHTTP2(t *testing.T) {
	tracer, transport := newRecordingTracer()
	defer tracer.Close()

	mux := http.NewServeMux()
	mux.Handle("/foo", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		w.Write([]byte("bar"))
	}))
	srv := httptest.NewUnstartedServer(&apmhttp.Handler{
		Handler: mux,
		Tracer:  tracer,
	})
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
	assert.Len(t, payloads, 1)
	assert.Contains(t, payloads[0], "transactions")
	transactions := payloads[0]["transactions"].([]interface{})
	assert.Len(t, transactions, 1)
	transaction := transactions[0].(map[string]interface{})
	context := transaction["context"].(map[string]interface{})
	assert.Equal(t, map[string]interface{}{
		"request": map[string]interface{}{
			"socket": map[string]interface{}{
				"encrypted":      true,
				"remote_address": "client.testing",
			},
			"url": map[string]interface{}{
				"full":     srv.URL + "/foo",
				"protocol": "https",
				"hostname": srvAddr.IP.String(),
				"port":     strconv.Itoa(srvAddr.Port),
				"pathname": "/foo",
			},
			"method": "GET",
			"headers": map[string]interface{}{
				"user-agent": "Go-http-client/2.0",
			},
			"http_version": "2.0",
		},
		"response": map[string]interface{}{
			"status_code":  float64(418),
			"headers_sent": true,
			"finished":     true,
		},
	}, context)
}

func TestHandlerRecovery(t *testing.T) {
	tracer, transport := newRecordingTracer()
	defer tracer.Close()

	h := &apmhttp.Handler{
		Handler:  http.HandlerFunc(panicHandler),
		Recovery: apmhttp.NewTraceRecovery(tracer),
		Tracer:   tracer,
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "http://server.testing/foo", nil)
	h.ServeHTTP(w, req)
	tracer.Flush(nil)

	payloads := transport.Payloads()
	assert.Len(t, payloads, 2)
	assert.Contains(t, payloads[0], "errors")
	assert.Contains(t, payloads[1], "transactions")

	errors := payloads[0]["errors"].([]interface{})
	error0 := errors[0].(map[string]interface{})
	assert.Equal(t, "panicHandler", error0["culprit"])
	exception := error0["exception"].(map[string]interface{})
	assert.Equal(t, "foo", exception["message"])

	transactions := payloads[1]["transactions"].([]interface{})
	transaction := transactions[0].(map[string]interface{})
	context := transaction["context"].(map[string]interface{})
	assert.Equal(t, map[string]interface{}{
		"headers_sent": true,
		"finished":     false,
		"status_code":  float64(418),
	}, context["response"])
}

func panicHandler(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusTeapot)
	panic("foo")
}

func newRecordingTracer() (*elasticapm.Tracer, *transporttest.RecorderTransport) {
	var transport transporttest.RecorderTransport
	tracer, err := elasticapm.NewTracer("apmhttp_test", "0.1")
	if err != nil {
		panic(err)
	}
	tracer.Transport = &transport
	return tracer, &transport
}
