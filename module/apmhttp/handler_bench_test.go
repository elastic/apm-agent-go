package apmhttp_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"testing"
	"time"

	"go.elastic.co/apm"
	"go.elastic.co/apm/module/apmhttp"
	"go.elastic.co/apm/transport"
)

var benchmarkPaths = []string{"/hello/world", "/sleep/1ms"}

func BenchmarkHandlerWithoutMiddleware(b *testing.B) {
	for _, path := range benchmarkPaths {
		b.Run(path, func(b *testing.B) {
			benchmarkHandler(b, path, nil)
		})
	}
}

func BenchmarkHandlerWithMiddleware(b *testing.B) {
	tracer := newTracer()
	defer tracer.Close()
	wrapHandler := func(in http.Handler) http.Handler {
		return apmhttp.Wrap(in, apmhttp.WithTracer(tracer))
	}
	for _, path := range benchmarkPaths {
		b.Run(path, func(b *testing.B) {
			benchmarkHandler(b, path, wrapHandler)
		})
	}
}

func benchmarkHandler(b *testing.B, path string, wrapHandler func(http.Handler) http.Handler) {
	w := httptest.NewRecorder()
	h := testMux()
	if wrapHandler != nil {
		h = wrapHandler(h)
	}
	req, _ := http.NewRequest("GET", path, nil)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		h.ServeHTTP(w, req)
	}
}

func newTracer() *apm.Tracer {
	tracer, err := apm.NewTracer("apmhttp_test", "0.1")
	if err != nil {
		panic(err)
	}

	invalidServerURL, err := url.Parse("http://testing.invalid:8200")
	if err != nil {
		panic(err)
	}

	httpTransport, err := transport.NewHTTPTransport()
	if err != nil {
		panic(err)
	}
	httpTransport.SetServerURL(invalidServerURL)
	tracer.Transport = httpTransport
	return tracer
}

func testMux() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/hello/", http.HandlerFunc(handleHello))
	mux.Handle("/sleep/", http.HandlerFunc(handleSleep))
	return mux
}

func handleHello(w http.ResponseWriter, req *http.Request) {
	w.Write([]byte(fmt.Sprintf("Hello, %s!", path.Base(req.URL.Path))))
}

func handleSleep(w http.ResponseWriter, req *http.Request) {
	d, err := time.ParseDuration(path.Base(req.URL.Path))
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to parse duration: %s", err), http.StatusBadRequest)
		return
	}
	time.Sleep(d)
}
