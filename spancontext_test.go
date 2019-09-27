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
