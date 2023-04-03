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

package apmotel // import "go.elastic.co/apm/module/apmotel"

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/aggregation"
)

func TestNewConfig(t *testing.T) {
	aggregationSelector := func(metric.InstrumentKind) aggregation.Aggregation { return nil }

	testCases := []struct {
		name       string
		options    []Option
		wantConfig config
	}{
		{
			name:       "Default",
			options:    nil,
			wantConfig: config{},
		},
		{
			name: "WithAggregationSelector",
			options: []Option{
				WithAggregationSelector(aggregationSelector),
			},
			wantConfig: config{},
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			cfg := newConfig(tt.options...)
			// tested by TestConfigManualReaderOptions
			cfg.aggregation = nil

			assert.Equal(t, tt.wantConfig, cfg)
		})
	}
}

func TestConfigManualReaderOptions(t *testing.T) {
	aggregationSelector := func(metric.InstrumentKind) aggregation.Aggregation { return nil }

	testCases := []struct {
		name            string
		config          config
		wantOptionCount int
	}{
		{
			name:            "Default",
			config:          config{},
			wantOptionCount: 0,
		},

		{
			name:            "WithAggregationSelector",
			config:          config{aggregation: aggregationSelector},
			wantOptionCount: 1,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			opts := tt.config.manualReaderOptions()
			assert.Len(t, opts, tt.wantOptionCount)
		})
	}
}
