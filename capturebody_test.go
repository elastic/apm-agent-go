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
	"net/http"
	"strings"
	"testing"

	"go.elastic.co/apm"
	"go.elastic.co/apm/apmtest"
)

func BenchmarkBodyCapturer(b *testing.B) {
	tracer := apmtest.NewDiscardTracer()
	defer tracer.Close()
	tracer.SetCaptureBody(apm.CaptureBodyAll)

	req, _ := http.NewRequest("GET", "http://testing.invalid", strings.NewReader(strings.Repeat("*", 1024*1024)))
	tx := tracer.StartTransaction("name", "type")

	for i := 0; i < b.N; i++ {
		bodyCapturer := tracer.CaptureHTTPRequestBody(req)
		tx.Context.SetHTTPRequestBody(bodyCapturer)
		bodyCapturer.Discard()
	}
}
