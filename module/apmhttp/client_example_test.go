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

package apmhttp_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"time"

	"go.elastic.co/apm/apmtest"
	"go.elastic.co/apm/module/apmhttp"
)

func ExampleWrapClient() {
	mux := http.NewServeMux()
	mux.HandleFunc("/slow", serveSlowly)
	server := httptest.NewServer(mux)
	defer server.Close()

	// Wrap the HTTP client with apmhttp.WrapClient. When using the
	// wrapped client, any request whose context contains a transaction
	// will have a span reported.
	client := apmhttp.WrapClient(http.DefaultClient)
	slowReq, _ := http.NewRequest("GET", server.URL+"/slow", nil)
	errorReq, _ := http.NewRequest("GET", "http://testing.invalid", nil)

	_, spans, _ := apmtest.WithTransaction(func(ctx context.Context) {
		// Propagate context with the outgoing request.
		req := slowReq.WithContext(ctx)
		resp, err := client.Do(req)
		if err != nil {
			log.Fatal(err)
		}

		// In the case where the request succeeds (i.e. no error
		// was returned above; unrelated to the HTTP status code),
		// the span is not ended until the body is consumed.
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("response: %s\n", body)

		// Send a request to a URL with an unresolvable host. This
		// will cause the entire request to fail, immediately
		// ending the span.
		resp, err = client.Do(errorReq.WithContext(ctx))
		if err != nil {
			fmt.Println("error occurred")
		} else {
			resp.Body.Close()
		}
	})

	if len(spans) != 2 {
		fmt.Println(len(spans), "spans")
	} else {
		for i, span := range spans {
			const expectedFloor = 250 * time.Millisecond
			if time.Duration(span.Duration*float64(time.Millisecond)) >= expectedFloor {
				// This is the expected case (see output below). As noted
				// previously, the span is only ended once the response body
				// has been consumed (or closed).
				fmt.Printf("span #%d duration >= %s\n", i+1, expectedFloor)
			} else {
				fmt.Printf("span #%d duration < %s\n", i+1, expectedFloor)
			}
		}
	}

	// Output:
	//
	// response: *yawn*
	// error occurred
	// span #1 duration >= 250ms
	// span #2 duration < 250ms
}

func serveSlowly(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusTeapot)
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
	time.Sleep(250 * time.Millisecond)
	w.Write([]byte("*yawn*"))
}
