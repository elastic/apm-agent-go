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

//go:build go1.18
// +build go1.18

package apmotel // import "go.elastic.co/apm/module/apmotel/v2"

import (
	"go.opentelemetry.io/otel/sdk/resource"

	"go.elastic.co/apm/v2"
)

type tracerProviderConfig struct {
	apmTracer *apm.Tracer

	// resource contains attributes representing an entity that produces telemetry.
	resource *resource.Resource
}

type TracerProviderOption func(tracerProviderConfig) tracerProviderConfig

func newTracerProviderConfig(opts ...TracerProviderOption) tracerProviderConfig {
	cfg := tracerProviderConfig{}
	for _, opt := range opts {
		cfg = opt(cfg)
	}

	if cfg.apmTracer == nil {
		cfg.apmTracer = apm.DefaultTracer()
	}

	if cfg.resource == nil {
		cfg.resource = resource.Default()
	}

	return cfg
}

// WithAPMTracer configures a custom apm.Tracer which will be used as the tracing bridge.
func WithAPMTracer(t *apm.Tracer) TracerProviderOption {
	return func(cfg tracerProviderConfig) tracerProviderConfig {
		cfg.apmTracer = t
		return cfg
	}
}

// WithResource configures the provided resource, which will be referenced by
// all tracers this provider creates.
//
// If this option is not used, the TracerProvider will use the
// resource.Default() Resource by default.
func WithResource(r *resource.Resource) TracerProviderOption {
	return func(cfg tracerProviderConfig) tracerProviderConfig {
		cfg.resource, _ = resource.Merge(resource.Environment(), r)
		return cfg
	}
}
