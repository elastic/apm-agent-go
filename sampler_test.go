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
	"encoding/binary"
	"math/rand"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.elastic.co/apm"
)

func TestRatioSampler(t *testing.T) {
	ratio := 0.75
	s := apm.NewRatioSampler(ratio)

	const (
		numGoroutines = 100
		numIterations = 1000
	)

	sampled := make([]int, numGoroutines)
	var wg sync.WaitGroup
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			// fixed seed to avoid intermittent failures
			rng := rand.New(rand.NewSource(int64(i)))
			for j := 0; j < numIterations; j++ {
				var traceContext apm.TraceContext
				binary.LittleEndian.PutUint64(traceContext.Span[:], rng.Uint64())
				if s.Sample(traceContext) {
					sampled[i]++
				}
			}

		}(i)
	}
	wg.Wait()

	var total int
	for i := 0; i < numGoroutines; i++ {
		total += sampled[i]
	}
	assert.InDelta(t, ratio, float64(total)/(numGoroutines*numIterations), 0.1)
}

func TestRatioSamplerAlways(t *testing.T) {
	s := apm.NewRatioSampler(1.0)
	assert.False(t, s.Sample(apm.TraceContext{})) // invalid span ID
	assert.True(t, s.Sample(apm.TraceContext{
		Span: apm.SpanID{0, 0, 0, 0, 0, 0, 0, 1},
	}))
	assert.True(t, s.Sample(apm.TraceContext{
		Span: apm.SpanID{255, 255, 255, 255, 255, 255, 255, 255},
	}))
}

func TestRatioSamplerNever(t *testing.T) {
	s := apm.NewRatioSampler(0)
	assert.False(t, s.Sample(apm.TraceContext{})) // invalid span ID
	assert.False(t, s.Sample(apm.TraceContext{
		Span: apm.SpanID{0, 0, 0, 0, 0, 0, 0, 1},
	}))
	assert.False(t, s.Sample(apm.TraceContext{
		Span: apm.SpanID{255, 255, 255, 255, 255, 255, 255, 255},
	}))
}
