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

package apmotel

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.opentelemetry.io/otel"

	"go.elastic.co/apm/v2/transport/transporttest"
)

func Example() {
	apmtracer, recorder := transporttest.NewRecorderTracer()
	provider, err := NewTracerProvider(WithAPMTracer(apmtracer))
	if err != nil {
		log.Fatal(err)
	}
	otel.SetTracerProvider(provider)
	defer apmtracer.Close()
	tracer := provider.Tracer("example")

	ctx := context.Background()
	ctx, parent := tracer.Start(ctx, "parent")

	for i := 0; i < 5; i++ {
		id := fmt.Sprintf("span_%d", i)
		_, child := tracer.Start(ctx, id)
		time.Sleep(10 * time.Millisecond)
		child.End()
	}
	parent.End()
	apmtracer.Flush(nil)

	payloads := recorder.Payloads()
	if len(payloads.Transactions) != 1 {
		panic(fmt.Errorf("expected 1 transaction, got %d", len(payloads.Transactions)))
	}
	for _, transaction := range payloads.Transactions {
		fmt.Printf("transaction: %s/%s\n", transaction.Name, transaction.Type)
	}
	for _, span := range payloads.Spans {
		fmt.Printf("span: %s/%s\n", span.Name, span.Type)
	}

	// Output:
	// transaction: parent/custom
	// span: span_0/custom
	// span: span_1/custom
	// span: span_2/custom
	// span: span_3/custom
	// span: span_4/custom
}
