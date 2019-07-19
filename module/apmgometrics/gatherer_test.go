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

package apmgometrics_test

import (
	"strings"
	"testing"

	metrics "github.com/rcrowley/go-metrics"
	"github.com/stretchr/testify/assert"

	"go.elastic.co/apm"
	"go.elastic.co/apm/apmtest"
	"go.elastic.co/apm/model"
	"go.elastic.co/apm/module/apmgometrics"
)

func TestGatherer(t *testing.T) {
	r := metrics.NewRegistry()
	httpReqsTotal := metrics.GetOrRegisterCounter("http.requests_total", r)
	httpReqsInflight := metrics.GetOrRegisterGauge("http.requests_inflight", r)
	httpReqsTotal.Inc(123)
	httpReqsInflight.Update(10)

	g := apmgometrics.Wrap(r)
	metrics := gatherMetrics(g)

	assert.Len(t, metrics, 1)
	for k := range metrics[0].Samples {
		if !strings.HasPrefix(k, "http.") {
			delete(metrics[0].Samples, k)
		}
	}

	assert.Equal(t, []model.Metrics{{
		Samples: map[string]model.Metric{
			"http.requests_total": {
				Value: 123,
			},
			"http.requests_inflight": {
				Value: 10,
			},
		},
	}}, metrics)
}

func TestHistogram(t *testing.T) {
	r := metrics.NewRegistry()
	sample := metrics.NewUniformSample(1024)
	hist := metrics.GetOrRegisterHistogram("histogram", r, sample)
	hist.Update(50)
	hist.Update(100)
	hist.Update(150)

	g := apmgometrics.Wrap(r)
	metrics := gatherMetrics(g)
	for name := range metrics[0].Samples {
		if !strings.HasPrefix(name, "histogram.") {
			delete(metrics[0].Samples, name)
		}
	}

	assert.Equal(t, map[string]model.Metric{
		"histogram.count":         {Value: 3},
		"histogram.total":         {Value: 300},
		"histogram.min":           {Value: 50},
		"histogram.max":           {Value: 150},
		"histogram.stddev":        {Value: 40.824829046386306},
		"histogram.percentile.50": {Value: 100},
		"histogram.percentile.95": {Value: 150},
		"histogram.percentile.99": {Value: 150},
	}, metrics[0].Samples)
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
