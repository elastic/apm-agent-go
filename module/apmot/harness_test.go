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

package apmot

import (
	"encoding/json"
	"io"
	"testing"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/harness"
	"github.com/stretchr/testify/suite"

	"go.elastic.co/apm"
	"go.elastic.co/apm/apmtest"
	"go.elastic.co/apm/module/apmhttp"
)

func TestHarness(t *testing.T) {
	// NOTE(axw) we do not support binary propagation, but we patch in
	// basic support *for the tests only* so we can check compatibility
	// with the HTTP and text formats.
	binaryInject = func(w io.Writer, traceContext apm.TraceContext) error {
		return json.NewEncoder(w).Encode(apmhttp.FormatTraceparentHeader(traceContext))
	}
	binaryExtract = func(r io.Reader) (apm.TraceContext, error) {
		var headerValue string
		if err := json.NewDecoder(r).Decode(&headerValue); err != nil {
			return apm.TraceContext{}, err
		}
		return apmhttp.ParseTraceparentHeader(headerValue)
	}
	defer func() {
		binaryInject = binaryInjectUnsupported
		binaryExtract = binaryExtractUnsupported
	}()

	newTracer := func() (opentracing.Tracer, func()) {
		tracer := New(WithTracer(apmtest.DiscardTracer))
		return tracer, func() {}
	}

	var done bool
	defer func() {
		if done {
			recover()
		}
	}()
	harness.RunAPIChecks(t, newTracer,
		harness.CheckExtract(true),
		harness.CheckInject(true),
		harness.UseProbe(harnessAPIProbe{}),
		func(s *harness.APICheckSuite) {
			suite.Run(t, harnessSuiteWrapper{s})
			done = true
			panic("done") // prevent suite.Run(t, s)
		},
	)
}

type harnessSuiteWrapper struct {
	*harness.APICheckSuite
}

func (w harnessSuiteWrapper) TestStartSpanWithParent() {
	// APICheckSuite.TestStartSpanWithParent tests both child-of and
	// follows-from. We don't support follows-from, but we don't want
	// that to prevent us from testing the child-of case.
	isValidSpanRef = func(ref opentracing.SpanReference) bool {
		switch ref.Type {
		case opentracing.ChildOfRef, opentracing.FollowsFromRef:
			return true
		}
		return false
	}
	defer func() { isValidSpanRef = isChildOfSpanRef }()
	w.APICheckSuite.TestStartSpanWithParent()
}

type harnessAPIProbe struct{}

func (harnessAPIProbe) SameTrace(first, second opentracing.Span) bool {
	ctx1, ok := first.Context().(*spanContext)
	if !ok {
		return false
	}
	ctx2, ok := second.Context().(*spanContext)
	if !ok {
		return false
	}
	return ctx1.traceContext.Trace == ctx2.traceContext.Trace
}

func (harnessAPIProbe) SameSpanContext(span opentracing.Span, sc opentracing.SpanContext) bool {
	ctx1, ok := span.Context().(*spanContext)
	if !ok {
		return false
	}
	ctx2, ok := sc.(*spanContext)
	if !ok {
		return false
	}
	return ctx1.traceContext == ctx2.traceContext
}
