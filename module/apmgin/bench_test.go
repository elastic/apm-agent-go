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

package apmgin_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"go.elastic.co/apm"
	"go.elastic.co/apm/module/apmgin"
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
	addMiddleware := func(r *gin.Engine) {
		r.Use(apmgin.Middleware(r, apmgin.WithTracer(tracer)))
	}
	for _, path := range benchmarkPaths {
		b.Run(path, func(b *testing.B) {
			benchmarkEngine(b, path, addMiddleware)
		})
	}
}

func benchmarkEngine(b *testing.B, path string, addMiddleware func(*gin.Engine)) {
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
		ServiceName:    "apmgin_test",
		ServiceVersion: "0.1",
		Transport:      httpTransport,
	})
	if err != nil {
		panic(err)
	}
	return tracer
}

func testRouter(addMiddleware func(*gin.Engine)) *gin.Engine {
	r := gin.New()
	if addMiddleware != nil {
		addMiddleware(r)
	}
	r.GET("/hello/:name", handleHello)
	r.GET("/sleep/:duration", handleSleep)
	return r
}

func handleHello(c *gin.Context) {
	c.String(http.StatusOK, "Hello, %s!", c.Param("name"))
}

func handleSleep(c *gin.Context) {
	d, err := time.ParseDuration(c.Param("duration"))
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}
	time.Sleep(d)
}
