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
					Samples: map[string]model.Metric{},
				},
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
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			gatherer, err := NewGatherer()
			assert.NoError(t, err)
			provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(gatherer))
			meter := provider.Meter("apmotel_test")

			tt.recordMetrics(ctx, meter)

			metrics := gatherMetrics(gatherer)

			// Remove all internal metrics
			for k := range metrics[0].Samples {
				if strings.HasPrefix(k, "golang.") {
					delete(metrics[0].Samples, k)
				}
			}

			assert.Equal(t, tt.expectedMetrics, metrics)
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
	return metrics
}
