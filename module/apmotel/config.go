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

package apmotel // import "go.elastic.co/apm/module/apmotel"

import "go.opentelemetry.io/otel/sdk/metric"

type config struct {
	aggregation metric.AggregationSelector
}

type Option func(config) config

func newConfig(opts ...Option) config {
	cfg := config{}
	for _, opt := range opts {
		opt(cfg)
	}

	return cfg
}

func (cfg config) manualReaderOptions() []metric.ManualReaderOption {
	opts := []metric.ManualReaderOption{}
	if cfg.aggregation != nil {
		opts = append(opts, metric.WithAggregationSelector(cfg.aggregation))
	}
	return opts
}

// WithAggregationSelector configure the Aggregation Selector the exporter will
// use. If no AggregationSelector is provided the DefaultAggregationSelector is
// used.
func WithAggregationSelector(agg metric.AggregationSelector) Option {
	return func(cfg config) config {
		cfg.aggregation = agg
		return cfg
	}
}
