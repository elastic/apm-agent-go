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

package apmiris_test

import (
	"fmt"
	"github.com/kataras/iris/v12"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"go.elastic.co/apm"
	"go.elastic.co/apm/module/apmiris"
	"go.elastic.co/apm/transport"
)

var benchmarkPaths = []string{"/hello/world", "/sleep/1ms"}

func BenchmarkWithoutMiddleware(b *testing.B) {
	for _, path := range benchmarkPaths {
		b.Run(path, func(b *testing.B) {
			benchmarkEngine(b, path, nil)
		})
	}
}

func BenchmarkWithMiddleware(b *testing.B) {
	tracer := newTracer()
	defer tracer.Close()
	addMiddleware := func(r *iris.Application) {
		r.Use(apmiris.Middleware(r, apmiris.WithTracer(tracer)))
	}
	for _, path := range benchmarkPaths {
		b.Run(path, func(b *testing.B) {
			benchmarkEngine(b, path, addMiddleware)
		})
	}
}

func benchmarkEngine(b *testing.B, path string, addMiddleware func(*iris.Application)) {
	w := httptest.NewRecorder()
	r := testRouter(addMiddleware)
	req, _ := http.NewRequest("GET", path, nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.ServeHTTP(w, req)
	}
}

func newTracer() *apm.Tracer {
	invalidServerURL, err := url.Parse("http://testing.invalid:8200")
	if err != nil {
		panic(err)
	}
	httpTransport, err := transport.NewHTTPTransport()
	if err != nil {
		panic(err)
	}
	httpTransport.SetServerURL(invalidServerURL)
	tracer, err := apm.NewTracerOptions(apm.TracerOptions{
		ServiceName:    "apmiris_test",
		ServiceVersion: "0.1",
		Transport:      httpTransport,
	})
	if err != nil {
		panic(err)
	}
	return tracer
}

func testRouter(addMiddleware func(*iris.Application)) *iris.Application {
	app := iris.New()
	if addMiddleware != nil {
		addMiddleware(app)
	}

	app.Get("/hello/{name:string}", handleHello)
	app.Get("/sleep/{duration:string}", handleSleep)
	return app
}

func handleHello(ctx iris.Context) {
	ctx.StatusCode(http.StatusOK)
	ctx.Writef(fmt.Sprintf("Hello, %s", ctx.Params().Get("name")))
}

func handleSleep(ctx iris.Context) {
	d, err := time.ParseDuration(ctx.Params().Get("duration"))
	if err != nil {
		ctx.StatusCode(http.StatusBadRequest)
		ctx.Writef(err.Error())
		return
	}

	time.Sleep(d)
}
