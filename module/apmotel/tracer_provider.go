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

package apmotel // import "go.elastic.co/apm/module/apmotel/v2"

import (
	"sync"

	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/embedded"

	"go.elastic.co/apm/v2"
)

type tracerProvider struct {
	mux sync.Mutex

	apmTracer *apm.Tracer

	tracers  map[string]*tracer
	resource *resource.Resource

	embedded.TracerProvider
}

// NewTracerProvider creates a new tracer provider which acts as a bridge with the Elastic Agent tracer.
func NewTracerProvider(opts ...TracerProviderOption) (trace.TracerProvider, error) {
	cfg := newTracerProviderConfig(opts...)
	tp := &tracerProvider{
		apmTracer: cfg.apmTracer,
		tracers:   map[string]*tracer{},
		resource:  cfg.resource,
	}

	return tp, nil
}

func (tp *tracerProvider) Tracer(name string, options ...trace.TracerOption) trace.Tracer {
	tp.mux.Lock()
	defer tp.mux.Unlock()

	if t, ok := tp.tracers[name]; ok {
		return t
	}

	t := newTracer(tp)
	tp.tracers[name] = t
	return t
}
