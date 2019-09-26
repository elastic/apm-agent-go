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
	"compress/zlib"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"time"

	"go.elastic.co/apm"
	"go.elastic.co/apm/transport"
)

// ExampleTracer shows how to use the Tracer API
func ExampleTracer() {
	var r recorder
	server := httptest.NewServer(&r)
	defer server.Close()

	// ELASTIC_APM_SERVER_URL should typically set in the environment
	// when the process is started. The InitDefault call below is only
	// required in this case because the environment variable is set
	// after the program has been initialized.
	os.Setenv("ELASTIC_APM_SERVER_URL", server.URL)
	defer os.Unsetenv("ELASTIC_APM_SERVER_URL")
	transport.InitDefault()

	const serviceName = "service-name"
	const serviceVersion = "1.0.0"
	tracer, err := apm.NewTracer(serviceName, serviceVersion)
	if err != nil {
		log.Fatal(err)
	}
	defer tracer.Close()

	// api is a very basic API handler, to demonstrate the usage
	// of the tracer. api.handlerOrder creates a transaction for
	// every call; api.handleOrder calls through to storeOrder,
	// which adds a span to the transaction.
	api := &api{tracer: tracer}
	api.handleOrder(context.Background(), "fish fingers")
	api.handleOrder(context.Background(), "detergent")

	// The tracer will stream events to the APM server, and will
	// close the request when it reaches a given size in bytes
	// (ELASTIC_APM_API_REQUEST_SIZE) or a given duration has
	// elapsed (ELASTIC_APM_API_REQUEST_TIME). Even so, we flush
	// here to ensure the data reaches the server.
	tracer.Flush(nil)

	fmt.Println("number of payloads:", len(r.payloads))
	metadata := r.payloads[0]["metadata"].(map[string]interface{})
	service := metadata["service"].(map[string]interface{})
	agent := service["agent"].(map[string]interface{})
	language := service["language"].(map[string]interface{})
	runtime := service["runtime"].(map[string]interface{})
	fmt.Println("  service name:", service["name"])
	fmt.Println("  service version:", service["version"])
	fmt.Println("  agent name:", agent["name"])
	fmt.Println("  language name:", language["name"])
	fmt.Println("  runtime name:", runtime["name"])

	var transactions []map[string]interface{}
	var spans []map[string]interface{}
	for _, p := range r.payloads[1:] {
		t, ok := p["transaction"].(map[string]interface{})
		if ok {
			transactions = append(transactions, t)
			continue
		}
		s, ok := p["span"].(map[string]interface{})
		if ok {
			spans = append(spans, s)
			continue
		}
	}
	if len(transactions) != len(spans) {
		fmt.Printf("%d transaction(s), %d span(s)\n", len(transactions), len(spans))
		return
	}
	for i, t := range transactions {
		s := spans[i]
		fmt.Printf("  transaction %d:\n", i)
		fmt.Println("    name:", t["name"])
		fmt.Println("    type:", t["type"])
		fmt.Println("    context:", t["context"])
		fmt.Printf("    span %d:\n", i)
		fmt.Println("      name:", s["name"])
		fmt.Println("      type:", s["type"])
	}

	// Output:
	// number of payloads: 5
	//   service name: service-name
	//   service version: 1.0.0
	//   agent name: go
	//   language name: go
	//   runtime name: gc
	//   transaction 0:
	//     name: order
	//     type: request
	//     context: map[tags:map[product:fish fingers]]
	//     span 0:
	//       name: store_order
	//       type: rpc
	//   transaction 1:
	//     name: order
	//     type: request
	//     context: map[tags:map[product:detergent]]
	//     span 1:
	//       name: store_order
	//       type: rpc
}

type api struct {
	tracer *apm.Tracer
}

func (api *api) handleOrder(ctx context.Context, product string) {
	tx := api.tracer.StartTransaction("order", "request")
	defer tx.End()
	ctx = apm.ContextWithTransaction(ctx, tx)

	tx.Context.SetLabel("product", product)

	time.Sleep(10 * time.Millisecond)
	storeOrder(ctx, product)
	time.Sleep(20 * time.Millisecond)
}

func storeOrder(ctx context.Context, product string) {
	span, _ := apm.StartSpan(ctx, "store_order", "rpc")
	defer span.End()

	time.Sleep(50 * time.Millisecond)
}

type recorder struct {
	mu       sync.Mutex
	payloads []map[string]interface{}
}

func (r *recorder) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.payloads)
}

func (r *recorder) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.URL.Path != "/intake/v2/events" {
		// Ignore config requests.
		return
	}
	body, err := zlib.NewReader(req.Body)
	if err != nil {
		panic(err)
	}
	decoder := json.NewDecoder(body)
	var payloads []map[string]interface{}
	for {
		var m map[string]interface{}
		if err := decoder.Decode(&m); err != nil {
			if err == io.EOF {
				break
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		payloads = append(payloads, m)
	}
	r.mu.Lock()
	r.payloads = append(r.payloads, payloads...)
	r.mu.Unlock()
}
