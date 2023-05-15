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
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/aggregation"

	"go.elastic.co/apm/v2"
	"go.elastic.co/apm/v2/apmtest"
	"go.elastic.co/apm/v2/model"
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
							Values: []float64{4.828425, 9.65685},
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
							Values: []float64{3.414215},
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
							Values: []float64{4.828425},
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

func TestGathererWithCustomView(t *testing.T) {
	ctx := context.Background()

	gatherer, err := NewGatherer()
	assert.NoError(t, err)
	provider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(gatherer),
		sdkmetric.WithView(sdkmetric.NewView(
			sdkmetric.Instrument{Name: "*"},
			sdkmetric.Stream{Aggregation: aggregation.ExplicitBucketHistogram{
				Boundaries: []float64{0, 5, 10, 25, 50, 75, 100, 250, 500, 1000},
			}},
		)),
	)
	meter := provider.Meter("apmotel_test")

	counter, err := meter.Float64Histogram("histogram_foo")
	assert.NoError(t, err)
	counter.Record(ctx, 3.4,
		attribute.Key("code").String("200"),
		attribute.Key("method").String("GET"),
	)

	metrics := gatherMetrics(gatherer)

	assert.ElementsMatch(t, []model.Metrics{
		{
			Samples: map[string]model.Metric{
				"histogram_foo": {
					Type:   "histogram",
					Values: []float64{2.5},
					Counts: []uint64{1},
				},
			},
			Labels: model.StringMap{
				{Key: "code", Value: "200"},
				{Key: "method", Value: "GET"},
			},
		},
	}, metrics)
}

func TestComputeCountAndBounds(t *testing.T) {
	for _, tt := range []struct {
		name string

		index  int
		bounds []float64
		counts []uint64

		expectedBound float64
		expectedCount uint64
	}{
		{
			name:   "with a zero count",
			index:  0,
			bounds: []float64{5},
			counts: []uint64{0, 0},

			expectedBound: 0,
			expectedCount: 0,
		},
		{
			name:   "with the -infinity bucket (zero index)",
			index:  0,
			bounds: []float64{6},
			counts: []uint64{1, 0},

			expectedBound: 3,
			expectedCount: 1,
		},
		{
			name:   "with the +infinity bucket (last index)",
			index:  2,
			bounds: []float64{6, 8},
			counts: []uint64{1, 2, 1},

			expectedBound: 8,
			expectedCount: 1,
		},
		{
			name:   "with midpoint boundaries",
			index:  1,
			bounds: []float64{6, 8},
			counts: []uint64{1, 2, 1},

			expectedBound: 7,
			expectedCount: 2,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			bound, count := computeCountAndBounds(tt.index, tt.bounds, tt.counts)
			assert.Equal(t, tt.expectedBound, bound)
			assert.Equal(t, tt.expectedCount, count)
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

	removeInternalMetrics(&metrics)
	return metrics
}

func TestDeltaTemporalityFilterOutZero(t *testing.T) {
	gatherer, err := NewGatherer()
	assert.NoError(t, err)
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(gatherer))
	meter := provider.Meter("apmotel_test")

	wg := sync.WaitGroup{}
	wg.Add(2)

	i64Histogram, err := meter.Int64Histogram("i64Histogram")
	require.NoError(t, err)
	i64Histogram.Record(context.Background(), 1)
	i64Counter, err := meter.Int64Counter("i64Counter")
	require.NoError(t, err)
	i64Counter.Add(context.Background(), 1)
	i64UDCounter, err := meter.Int64UpDownCounter("i64UDCounter")
	require.NoError(t, err)
	i64UDCounter.Add(context.Background(), 1)
	i64ObservableCounter, err := meter.Int64ObservableCounter("i64ObservableCounter")
	require.NoError(t, err)
	i64ObservableUDCounter, err := meter.Int64ObservableUpDownCounter("i64ObservableUDCounter")
	require.NoError(t, err)
	i64ObservableGauge, err := meter.Int64ObservableGauge("i64ObservableGauge")
	require.NoError(t, err)
	f64Histogram, err := meter.Float64Histogram("f64Histogram")
	require.NoError(t, err)
	f64Histogram.Record(context.Background(), 1)
	f64Counter, err := meter.Float64Counter("f64Counter")
	require.NoError(t, err)
	f64Counter.Add(context.Background(), 1)
	f64UDCounter, err := meter.Float64UpDownCounter("f64UDCounter")
	require.NoError(t, err)
	f64UDCounter.Add(context.Background(), 1)
	f64ObservableCounter, err := meter.Float64ObservableCounter("f64ObservableCounter")
	require.NoError(t, err)
	f64ObservableUDCounter, err := meter.Float64ObservableUpDownCounter("f64ObservableUDCounter")
	require.NoError(t, err)
	f64ObservableGauge, err := meter.Float64ObservableGauge("f64ObservableGauge")
	require.NoError(t, err)
	registration, err := meter.RegisterCallback(
		func(_ context.Context, obs metric.Observer) error {
			wg.Done()
			obs.ObserveInt64(i64ObservableCounter, 1)
			obs.ObserveInt64(i64ObservableUDCounter, 1)
			obs.ObserveInt64(i64ObservableGauge, 1)
			obs.ObserveFloat64(f64ObservableCounter, 1)
			obs.ObserveFloat64(f64ObservableUDCounter, 1)
			obs.ObserveFloat64(f64ObservableGauge, 1)
			return nil
		},
		i64ObservableCounter,
		i64ObservableUDCounter,
		i64ObservableGauge,
		f64ObservableCounter,
		f64ObservableUDCounter,
		f64ObservableGauge,
	)
	require.NoError(t, err)
	defer registration.Unregister()

	t.Setenv("ELASTIC_APM_METRICS_INTERVAL", "1s")
	tracer := apmtest.NewRecordingTracer()
	defer tracer.Close()
	tracer.RegisterMetricsGatherer(gatherer)
	tracer.SendMetrics(nil)
	metrics := tracer.Payloads().Metrics
	for i := range metrics {
		metrics[i].Timestamp = model.Time{}
	}

	removeInternalMetrics(&metrics)
	var names []string
	for _, m := range metrics {
		for k := range m.Samples {
			names = append(names, k)
		}
	}
	assert.ElementsMatch(t, []string{"i64Histogram",
		"i64Counter",
		"i64UDCounter",
		"i64ObservableCounter",
		"i64ObservableUDCounter",
		"i64ObservableGauge",
		"f64Histogram",
		"f64Counter",
		"f64UDCounter",
		"f64ObservableCounter",
		"f64ObservableUDCounter",
		"f64ObservableGauge",
	}, names)

	tracer.ResetPayloads()
	wg.Wait()
	tracer.SendMetrics(nil)
	metrics = tracer.Payloads().Metrics
	for i := range metrics {
		metrics[i].Timestamp = model.Time{}
	}

	removeInternalMetrics(&metrics)
	names = names[:0]
	for _, m := range metrics {
		for k := range m.Samples {
			names = append(names, k)
		}
	}
	assert.ElementsMatch(t, []string{
		"i64UDCounter",
		"f64UDCounter",
	}, names)
}

func removeInternalMetrics(metrics *[]model.Metrics) {
	for i, m := range *metrics {
		for k := range m.Samples {
			if strings.HasPrefix(k, "golang.") || strings.HasPrefix(k, "system.") {
				delete(m.Samples, k)
			}
		}

		if len(m.Samples) == 0 {
			(*metrics)[i] = (*metrics)[len(*metrics)-1]
			*metrics = (*metrics)[:len(*metrics)-1]
		}
	}
}
