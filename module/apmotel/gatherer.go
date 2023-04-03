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

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"go.elastic.co/apm"
)

// NewGatherer creates a new gatherer/exporter to bridge between agent metrics and OpenTelemetry
func NewGatherer(opts ...Option) (Gatherer, error) {
	cfg := newConfig(opts...)
	reader := metric.NewManualReader(cfg.manualReaderOptions()...)

	return Gatherer{reader}, nil
}

// Gatherer is a gatherer/exporter which can act as an OpenTelemetry manual
// reader, to retrieve metrics and expose them to the agent.
type Gatherer struct {
	metric.Reader
}

var _ metric.Reader = Gatherer{}
var _ apm.MetricsGatherer = Gatherer{}

// GatherMetrics gathers metrics into out.
func (e Gatherer) GatherMetrics(ctx context.Context, out *apm.Metrics) error {
	metrics := metricdata.ResourceMetrics{}
	err := e.Reader.Collect(ctx, &metrics)
	if err != nil {
		return err
	}

	for _, scopeMetrics := range metrics.ScopeMetrics {

		for _, sm := range scopeMetrics.Metrics {
			switch m := sm.Data.(type) {
			case metricdata.Histogram[int64]:
				addHistogramMetric(out, sm, m)
			case metricdata.Histogram[float64]:
				addHistogramMetric(out, sm, m)
			case metricdata.Sum[int64]:
				for _, dp := range m.DataPoints {
					out.Add(sm.Name, makeLabels(dp.Attributes), float64(dp.Value))
				}
			case metricdata.Sum[float64]:
				for _, dp := range m.DataPoints {
					out.Add(sm.Name, makeLabels(dp.Attributes), dp.Value)
				}
			case metricdata.Gauge[int64]:
				for _, dp := range m.DataPoints {
					out.Add(sm.Name, makeLabels(dp.Attributes), float64(dp.Value))
				}
			case metricdata.Gauge[float64]:
				for _, dp := range m.DataPoints {
					out.Add(sm.Name, makeLabels(dp.Attributes), dp.Value)
				}
			default:
				return fmt.Errorf("unknown metric type %q", m)
			}
		}
	}

	return nil
}

func addHistogramMetric[N int64 | float64](out *apm.Metrics, sm metricdata.Metrics, m metricdata.Histogram[N]) {
	for _, dp := range m.DataPoints {
		values := make([]float64, 0, len(dp.Bounds))
		counts := make([]uint64, 0, len(dp.BucketCounts))

		for i, v := range dp.Bounds {
			count := dp.BucketCounts[i]

			if count > 0 {
				counts = append(counts, count)
				values = append(values, v)
			}
		}

		out.AddHistogram(sm.Name, makeLabels(dp.Attributes), values, counts)
	}
}

func makeLabels(attrs attribute.Set) []apm.MetricLabel {
	labels := make([]apm.MetricLabel, attrs.Len())
	iter := attrs.Iter()
	for iter.Next() {
		i, kv := iter.IndexedAttribute()
		labels[i] = apm.MetricLabel{Name: string(kv.Key), Value: kv.Value.Emit()}
	}

	return labels
}
