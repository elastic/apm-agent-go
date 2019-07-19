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

package apmprometheus_test

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.elastic.co/apm"
	"go.elastic.co/apm/apmtest"
	"go.elastic.co/apm/model"
	"go.elastic.co/apm/module/apmprometheus"
)

func TestGoCollector(t *testing.T) {
	g := apmprometheus.Wrap(prometheus.DefaultGatherer)
	metrics := gatherMetrics(g)
	require.Len(t, metrics, 1)
	assert.Nil(t, metrics[0].Labels)

	assert.Contains(t, metrics[0].Samples, "go_memstats_alloc_bytes")
	assert.Contains(t, metrics[0].Samples, "go_memstats_alloc_bytes_total")
	assert.NotNil(t, metrics[0].Samples["go_memstats_alloc_bytes"].Value)
	assert.NotNil(t, metrics[0].Samples["go_memstats_alloc_bytes_total"].Value)
}

func TestSummary(t *testing.T) {
	r := prometheus.NewRegistry()
	s := prometheus.NewSummary(prometheus.SummaryOpts{
		Name:       "summary",
		Help:       "halp",
		Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
	})
	r.MustRegister(s)

	s.Observe(50)
	s.Observe(100)
	s.Observe(150)

	g := apmprometheus.Wrap(r)
	metrics := gatherMetrics(g)
	for name := range metrics[0].Samples {
		if !strings.HasPrefix(name, "summary.") {
			delete(metrics[0].Samples, name)
		}
	}
	assert.Equal(t, map[string]model.Metric{
		"summary.count":         {Value: 3},
		"summary.total":         {Value: 300},
		"summary.percentile.50": {Value: 100},
		"summary.percentile.90": {Value: 150},
		"summary.percentile.99": {Value: 150},
	}, metrics[0].Samples)
}

func TestLabels(t *testing.T) {
	r := prometheus.NewRegistry()
	httpReqsTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "http_requests_total", Help: "."},
		[]string{"code", "method"},
	)
	httpReqsInflight := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{Name: "http_requests_inflight", Help: "."},
		[]string{"code", "method"},
	)
	r.MustRegister(httpReqsTotal)
	r.MustRegister(httpReqsInflight)

	httpReqsTotal.WithLabelValues("404", "GET").Inc()
	httpReqsTotal.WithLabelValues("200", "PUT").Inc()
	httpReqsTotal.WithLabelValues("200", "GET").Add(123)
	httpReqsInflight.WithLabelValues("200", "GET").Set(10)

	g := apmprometheus.Wrap(r)
	metrics := gatherMetrics(g)

	assert.NotEmpty(t, metrics)
	assert.Empty(t, metrics[0].Labels)
	assert.Contains(t, metrics[0].Samples, "golang.heap.allocations.mallocs")
	metrics = metrics[1:]

	assert.Equal(t, []model.Metrics{{
		Labels: model.StringMap{
			{Key: "code", Value: "200"},
			{Key: "method", Value: "GET"},
		},
		Samples: map[string]model.Metric{
			"http_requests_total": {
				Value: 123,
			},
			"http_requests_inflight": {
				Value: 10,
			},
		},
	}, {
		Labels: model.StringMap{
			{Key: "code", Value: "200"},
			{Key: "method", Value: "PUT"},
		},
		Samples: map[string]model.Metric{
			"http_requests_total": {
				Value: 1,
			},
		},
	}, {
		Labels: model.StringMap{
			{Key: "code", Value: "404"},
			{Key: "method", Value: "GET"},
		},
		Samples: map[string]model.Metric{
			"http_requests_total": {
				Value: 1,
			},
		},
	}}, metrics)
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
