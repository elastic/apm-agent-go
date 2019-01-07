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
	"testing"

	"github.com/stretchr/testify/assert"

	"go.elastic.co/apm"
)

func TestTraceID(t *testing.T) {
	var id apm.TraceID
	assert.EqualError(t, id.Validate(), "zero trace-id is invalid")

	id[0] = 1
	assert.NoError(t, id.Validate())
}

func TestSpanID(t *testing.T) {
	var id apm.SpanID
	assert.EqualError(t, id.Validate(), "zero span-id is invalid")

	id[0] = 1
	assert.NoError(t, id.Validate())
}

func TestTraceOptions(t *testing.T) {
	opts := apm.TraceOptions(0xFE)
	assert.False(t, opts.Recorded())

	opts = opts.WithRecorded(true)
	assert.True(t, opts.Recorded())
	assert.Equal(t, apm.TraceOptions(0xFF), opts)

	opts = opts.WithRecorded(false)
	assert.False(t, opts.Recorded())
	assert.Equal(t, apm.TraceOptions(0xFE), opts)
}
