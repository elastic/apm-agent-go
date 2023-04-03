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

package apmotel

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"

	"go.elastic.co/apm"
	"go.elastic.co/apm/apmtest"
	"go.elastic.co/apm/model"
)

func TestGatherer(t *testing.T) {
	for _, tt := range []struct {
		name string

		recordMetrics   func(ctx context.Context, meter metric.Meter)
		expectedMetrics []model.Metrics
	}{
		{
			name: "with a float64 counter",
			recordMetrics: func(ctx context.Context, meter metric.Meter) {
				counter, err := meter.Float64Counter("foo")
				assert.NoError(t, err)
				counter.Add(ctx, 5, attribute.Key("A").String("B"))
			},
			expectedMetrics: []model.Metrics{
				{
					Samples: map[string]model.Metric{
						"foo": {
							Value: 5,
						},
					},
					Labels: model.StringMap{
						model.StringMapItem{Key: "A", Value: "B"},
					},
				},
			},
		},
		{
			name: "with an int64 counter",
			recordMetrics: func(ctx context.Context, meter metric.Meter) {
				counter, err := meter.Int64Counter("foo")
				assert.NoError(t, err)
				counter.Add(ctx, 5, attribute.Key("A").String("B"))
			},
			expectedMetrics: []model.Metrics{
				{
					Samples: map[string]model.Metric{
						"foo": {
							Value: 5,
						},
					},
					Labels: model.StringMap{
						model.StringMapItem{Key: "A", Value: "B"},
					},
				},
			},
		},
		{
			name: "with a float64 gauge",
			recordMetrics: func(ctx context.Context, meter metric.Meter) {
				counter, err := meter.Float64UpDownCounter("foo")
				assert.NoError(t, err)
				counter.Add(ctx, 5, attribute.Key("A").String("B"))
			},
			expectedMetrics: []model.Metrics{
				{
					Samples: map[string]model.Metric{
						"foo": {
							Value: 5,
						},
					},
					Labels: model.StringMap{
						model.StringMapItem{Key: "A", Value: "B"},
					},
				},
			},
		},
		{
			name: "with an int64 gauge",
			recordMetrics: func(ctx context.Context, meter metric.Meter) {
				counter, err := meter.Float64UpDownCounter("foo")
				assert.NoError(t, err)
				counter.Add(ctx, 5, attribute.Key("A").String("B"))
			},
			expectedMetrics: []model.Metrics{
				{
					Samples: map[string]model.Metric{
						"foo": {
							Value: 5,
						},
					},
					Labels: model.StringMap{
						model.StringMapItem{Key: "A", Value: "B"},
					},
				},
			},
		},
		{
			name: "with a float64 histogram",
			recordMetrics: func(ctx context.Context, meter metric.Meter) {
				counter, err := meter.Float64Histogram("histogram_foo")
				assert.NoError(t, err)
				counter.Record(ctx, 3.4,
					attribute.Key("code").String("200"),
					attribute.Key("method").String("GET"),
				)
				counter.Record(ctx, 3.4,
					attribute.Key("code").String("200"),
					attribute.Key("method").String("GET"),
				)
				counter.Record(ctx, 3.4,
					attribute.Key("code").String("200"),
					attribute.Key("method").String("GET"),
				)

				counter.Record(ctx, 5.5,
					attribute.Key("code").String("302"),
					attribute.Key("method").String("GET"),
				)
				counter.Record(ctx, 5.5,
					attribute.Key("code").String("302"),
					attribute.Key("method").String("GET"),
				)
				counter.Record(ctx, 5.5,
					attribute.Key("code").String("302"),
					attribute.Key("method").String("GET"),
				)

				counter.Record(ctx, 11.2,
					attribute.Key("code").String("302"),
					attribute.Key("method").String("GET"),
				)
				counter.Record(ctx, 11.2,
					attribute.Key("code").String("302"),
					attribute.Key("method").String("GET"),
				)
				counter.Record(ctx, 11.2,
					attribute.Key("code").String("302"),
					attribute.Key("method").String("GET"),
				)
			},
			expectedMetrics: []model.Metrics{
				{
					Samples: map[string]model.Metric{
						"histogram_foo": {
							Type:   "histogram",
							Values: []float64{10, 25},
							Counts: []uint64{3, 3},
						},
					},
					Labels: model.StringMap{
						{Key: "code", Value: "302"},
						{Key: "method", Value: "GET"},
					},
				},
				{
					Samples: map[string]model.Metric{
						"histogram_foo": {
							Type:   "histogram",
							Values: []float64{5},
							Counts: []uint64{3},
						},
					},
					Labels: model.StringMap{
						{Key: "code", Value: "200"},
						{Key: "method", Value: "GET"},
					},
				},
			},
		},
		{
			name: "with an int64 histogram",
			recordMetrics: func(ctx context.Context, meter metric.Meter) {
				counter, err := meter.Int64Histogram("foo")
				assert.NoError(t, err)
				counter.Record(ctx, 5, attribute.Key("A").String("B"))
			},
			expectedMetrics: []model.Metrics{
				{
					Samples: map[string]model.Metric{
						"foo": {
							Type:   "histogram",
							Values: []float64{5},
							Counts: []uint64{1},
						},
					},
					Labels: model.StringMap{
						model.StringMapItem{Key: "A", Value: "B"},
					},
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			gatherer, err := NewGatherer()
			assert.NoError(t, err)
			provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(gatherer))
			meter := provider.Meter("apmotel_test")

			tt.recordMetrics(ctx, meter)

			metrics := gatherMetrics(gatherer)

			assert.ElementsMatch(t, tt.expectedMetrics, metrics)
		})
	}
}

func gatherMetrics(g apm.MetricsGatherer) []model.Metrics {
	tracer := apmtest.NewRecordingTracer()
	defer tracer.Close()
	tracer.RegisterMetricsGatherer(g)
	tracer.SendMetrics(nil)
	metrics := tracer.Payloads().Metrics
	for i := range metrics {
		metrics[i].Timestamp = model.Time{}
	}

	// Remove internal metrics
	for i, m := range metrics {
		for k := range m.Samples {
			if strings.HasPrefix(k, "golang.") {
				delete(m.Samples, k)
			}
		}

		if len(m.Samples) == 0 {
			metrics[i] = metrics[len(metrics)-1]
			metrics = metrics[:len(metrics)-1]
		}
	}
	return metrics
}
