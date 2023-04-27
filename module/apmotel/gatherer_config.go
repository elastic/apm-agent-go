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
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/aggregation"
)

var customHistogramBoundaries = []float64{
	0.00390625, 0.00552427, 0.0078125, 0.0110485, 0.015625, 0.0220971, 0.03125,
	0.0441942, 0.0625, 0.0883883, 0.125, 0.176777, 0.25, 0.353553, 0.5, 0.707107,
	1, 1.41421, 2, 2.82843, 4, 5.65685, 8, 11.3137, 16, 22.6274, 32, 45.2548, 64,
	90.5097, 128, 181.019, 256, 362.039, 512, 724.077, 1024, 1448.15, 2048,
	2896.31, 4096, 5792.62, 8192, 11585.2, 16384, 23170.5, 32768, 46341.0, 65536,
	92681.9, 131072,
}

type gathererConfig struct {
	aggregation metric.AggregationSelector
}

type GathererOption func(gathererConfig) gathererConfig

func newGathererConfig(opts ...GathererOption) gathererConfig {
	cfg := gathererConfig{
		aggregation: customAggregationSelector,
	}
	for _, opt := range opts {
		cfg = opt(cfg)
	}

	return cfg
}

func (cfg gathererConfig) manualReaderOptions() []metric.ManualReaderOption {
	opts := []metric.ManualReaderOption{}
	opts = append(opts, metric.WithAggregationSelector(cfg.aggregation))
	return opts
}

// WithAggregationSelector configure the Aggregation Selector the exporter will
// use. If no AggregationSelector is provided the DefaultAggregationSelector is
// used.
func WithAggregationSelector(agg metric.AggregationSelector) GathererOption {
	return func(cfg gathererConfig) gathererConfig {
		cfg.aggregation = agg
		return cfg
	}
}

func customAggregationSelector(ik metric.InstrumentKind) aggregation.Aggregation {
	switch ik {
	case metric.InstrumentKindHistogram:
		return aggregation.ExplicitBucketHistogram{
			Boundaries: customHistogramBoundaries,
			NoMinMax:   false,
		}
	default:
		return metric.DefaultAggregationSelector(ik)
	}
}
