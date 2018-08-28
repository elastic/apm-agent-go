package elasticapm_test

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

	"github.com/elastic/apm-agent-go"
	"github.com/elastic/apm-agent-go/transport"
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
	tracer, err := elasticapm.NewTracer(serviceName, serviceVersion)
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
	for i, p := range r.payloads[1:] {
		t := p["transaction"].(map[string]interface{})
		fmt.Printf("  transaction %d:\n", i)
		fmt.Println("    name:", t["name"])
		fmt.Println("    type:", t["type"])
		fmt.Println("    context:", t["context"])
		spans := t["spans"].([]interface{})
		for i := range spans {
			s := spans[i].(map[string]interface{})
			fmt.Printf("    span %d:\n", i)
			fmt.Println("      name:", s["name"])
			fmt.Println("      type:", s["type"])
		}
	}

	// Output:
	// number of payloads: 3
	//   service name: service-name
	//   service version: 1.0.0
	//   agent name: go
	//   language name: go
	//   runtime name: gc
	//   transaction 0:
	//     name: order
	//     type: request
	//     context: map[custom:map[product:fish fingers]]
	//     span 0:
	//       name: store_order
	//       type: rpc
	//   transaction 1:
	//     name: order
	//     type: request
	//     context: map[custom:map[product:detergent]]
	//     span 0:
	//       name: store_order
	//       type: rpc
}

type api struct {
	tracer *elasticapm.Tracer
}

func (api *api) handleOrder(ctx context.Context, product string) {
	tx := api.tracer.StartTransaction("order", "request")
	defer tx.End()
	ctx = elasticapm.ContextWithTransaction(ctx, tx)

	tx.Context.SetCustom("product", product)

	time.Sleep(10 * time.Millisecond)
	storeOrder(ctx, product)
	time.Sleep(20 * time.Millisecond)
}

func storeOrder(ctx context.Context, product string) {
	span, _ := elasticapm.StartSpan(ctx, "store_order", "rpc")
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
