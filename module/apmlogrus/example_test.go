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

package apmlogrus_test

import (
	"context"

	"github.com/sirupsen/logrus"

	"go.elastic.co/apm"
	"go.elastic.co/apm/module/apmlogrus"
)

func ExampleHook() {
	logger := logrus.New()

	// Report "error", "panic", and "fatal" log messages
	// to Elastic APM using apm.DefaultTracer.
	logger.AddHook(&apmlogrus.Hook{})

	// Report "error", "panic", and "fatal" log messages
	// to Elastic APM using a specific tracer.
	var tracer *apm.Tracer
	logger.AddHook(&apmlogrus.Hook{
		Tracer: tracer,
	})

	// Report only "panic" log messages to Elastic APM
	// using apm.DefaultTracer.
	logger.AddHook(&apmlogrus.Hook{
		LogLevels: []logrus.Level{logrus.PanicLevel},
	})
}

func ExampleTraceContext() {
	logger := logrus.New()

	tx := apm.DefaultTracer.StartTransaction("name", "type")
	defer tx.End()

	ctx := apm.ContextWithTransaction(context.Background(), tx)
	span, ctx := apm.StartSpan(ctx, "name", "type")
	defer span.End()

	// apmlogrus.TraceContext returns fields including the trace ID,
	// transaction ID, and span ID, for the transaction and span in
	// the given context.
	logger.WithFields(apmlogrus.TraceContext(ctx)).Fatal("ohhh, what a world")
}
