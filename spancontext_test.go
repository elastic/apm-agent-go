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
	"context"
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.elastic.co/apm"
	"go.elastic.co/apm/apmtest"
	"go.elastic.co/apm/model"
)

func TestSpanContextSetLabel(t *testing.T) {
	_, spans, _ := apmtest.WithTransaction(func(ctx context.Context) {
		span, _ := apm.StartSpan(ctx, "name", "type")
		span.Context.SetTag("foo", "bar")    // deprecated
		span.Context.SetLabel("foo", "bar!") // Last instance wins
		span.Context.SetLabel("bar", "baz")
		span.Context.SetLabel("baz", 123.456)
		span.Context.SetLabel("qux", true)
		span.End()
	})
	require.Len(t, spans, 1)
	assert.Equal(t, model.IfaceMap{
		{Key: "bar", Value: "baz"},
		{Key: "baz", Value: 123.456},
		{Key: "foo", Value: "bar!"},
		{Key: "qux", Value: true},
	}, spans[0].Context.Tags)
}

func TestSpanContextSetHTTPRequest(t *testing.T) {
	type testcase struct {
		url string

		addr     string
		port     int
		name     string
		resource string
	}

	testcases := []testcase{{
		url:      "http://localhost/foo/bar",
		addr:     "localhost",
		port:     80,
		name:     "http://localhost",
		resource: "localhost:80",
	}, {
		url:      "http://localhost:80/foo/bar",
		addr:     "localhost",
		port:     80,
		name:     "http://localhost",
		resource: "localhost:80",
	}, {
		url:      "https://[::1]/foo/bar",
		addr:     "::1",
		port:     443,
		name:     "https://[::1]",
		resource: "[::1]:443",
	}, {
		url:      "https://[::1]:8443/foo/bar",
		addr:     "::1",
		port:     8443,
		name:     "https://[::1]:8443",
		resource: "[::1]:8443",
	}, {
		url:      "gopher://gopher.invalid:70",
		addr:     "gopher.invalid",
		port:     70,
		name:     "gopher://gopher.invalid:70",
		resource: "gopher.invalid:70",
	}, {
		url:      "gopher://gopher.invalid",
		addr:     "gopher.invalid",
		port:     0,
		name:     "gopher://gopher.invalid",
		resource: "gopher.invalid",
	}}

	for _, tc := range testcases {
		t.Run(tc.url, func(t *testing.T) {
			url, err := url.Parse(tc.url)
			require.NoError(t, err)

			_, spans, _ := apmtest.WithTransaction(func(ctx context.Context) {
				span, _ := apm.StartSpan(ctx, "name", "type")
				span.Context.SetHTTPRequest(&http.Request{URL: url})
				span.End()
			})
			require.Len(t, spans, 1)

			assert.Equal(t, &model.DestinationSpanContext{
				Address: tc.addr,
				Port:    tc.port,
				Service: &model.DestinationServiceSpanContext{
					Type:     spans[0].Type,
					Name:     tc.name,
					Resource: tc.resource,
				},
			}, spans[0].Context.Destination)
		})
	}
}

func TestSetDestinationService(t *testing.T) {
	type testcase struct {
		name        string
		resource    string
		expectEmpty bool
	}

	testcases := []testcase{{
		resource:    "",
		expectEmpty: true,
	}, {
		resource:    "nonempty",
		expectEmpty: false,
	}, {
		resource:    "",
		expectEmpty: true,
	}, {
		resource: "nonempty",
	}}
	for _, tc := range testcases {
		t.Run(fmt.Sprintf("%s_%s", tc.name, tc.resource), func(t *testing.T) {
			_, spans, _ := apmtest.WithTransaction(func(ctx context.Context) {
				span, _ := apm.StartSpan(ctx, "name", "span_type")
				span.Context.SetDestinationAddress("testing.invalid", 123)
				span.Context.SetDestinationService(apm.DestinationServiceSpanContext{
					Resource: tc.resource,
				})
				span.End()
			})
			require.Len(t, spans, 1)
			if tc.expectEmpty {
				assert.Nil(t, spans[0].Context.Destination.Service)
			} else {
				assert.Equal(t, &model.DestinationServiceSpanContext{
					Resource: tc.resource,
					Type:     "span_type",
				}, spans[0].Context.Destination.Service)
			}
		})
	}
}
