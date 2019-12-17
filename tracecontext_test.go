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
	"fmt"
	"strings"
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

func TestTraceStateInvalidLength(t *testing.T) {
	const maxEntries = 32

	entries := make([]apm.TraceStateEntry, 0, maxEntries)
	for i := 0; i < cap(entries); i++ {
		entries = append(entries, apm.TraceStateEntry{Key: fmt.Sprintf("k%d", i), Value: "value"})
		ts := apm.NewTraceState(entries...)
		assert.NoError(t, ts.Validate())
	}

	entries = append(entries, apm.TraceStateEntry{Key: "straw", Value: "camel's back"})
	ts := apm.NewTraceState(entries...)
	assert.EqualError(t, ts.Validate(), "tracestate contains more than the maximum allowed number of entries, 32")
}

func TestTraceStateDuplicateKey(t *testing.T) {
	ts := apm.NewTraceState(
		apm.TraceStateEntry{Key: "x", Value: "b"},
		apm.TraceStateEntry{Key: "a", Value: "b"},
		apm.TraceStateEntry{Key: "y", Value: "b"},
		apm.TraceStateEntry{Key: "a", Value: "b"},
	)
	assert.EqualError(t, ts.Validate(), `duplicate tracestate key "a" at positions 1 and 3`)
}

func TestTraceStateInvalidKey(t *testing.T) {
	ts := apm.NewTraceState(apm.TraceStateEntry{Key: "~"})
	assert.EqualError(t, ts.Validate(), `invalid tracestate entry at position 0: invalid key "~"`)
}

func TestTraceStateInvalidValueLength(t *testing.T) {
	ts := apm.NewTraceState(apm.TraceStateEntry{Key: "oy"})
	assert.EqualError(t, ts.Validate(), `invalid tracestate entry at position 0: invalid value for key "oy": value is empty`)

	ts = apm.NewTraceState(apm.TraceStateEntry{Key: "oy", Value: strings.Repeat("*", 257)})
	assert.EqualError(t, ts.Validate(),
		`invalid tracestate entry at position 0: invalid value for key "oy": value contains 257 characters, maximum allowed is 256`)
}

func TestTraceStateInvalidValueCharacter(t *testing.T) {
	for _, value := range []string{
		string(0),
		"header" + string(0) + "trailer",
	} {
		ts := apm.NewTraceState(apm.TraceStateEntry{Key: "oy", Value: value})
		assert.EqualError(t, ts.Validate(),
			`invalid tracestate entry at position 0: invalid value for key "oy": value contains invalid character '\x00'`)
	}
}
