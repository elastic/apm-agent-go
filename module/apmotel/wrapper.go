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
	"context"

	"go.elastic.co/apm/v2"
	"go.opentelemetry.io/otel/trace"
)

func init() {
	// We override the apm context functions so that spans started with the
	// native API are wrapped and made available as OTel spans.
	apm.OverrideContextWithSpan = contextWithSpan
	apm.OverrideSpanFromContext = spanFromContext
}

func contextWithSpan(ctx context.Context, s *apm.Span) context.Context {
	// Should we make a note that they need to update the default tracer to
	// use this? Or is there some better way to do this.
	otelSpan := &span{inner: s, tracer: apm.DefaultTracer()}
	return trace.ContextWithSpan(ctx, otelSpan)
}

func spanFromContext(ctx context.Context) *apm.Span {
	otelSpan := trace.SpanFromContext(ctx).(interface {
		Span() *apm.Span
	})
	if otelSpan == nil {
		return nil
	}
	return otelSpan.Span()
}
