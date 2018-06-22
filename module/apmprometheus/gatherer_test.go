package apmprometheus_test

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/apm-agent-go"
	"github.com/elastic/apm-agent-go/model"
	"github.com/elastic/apm-agent-go/module/apmprometheus"
	"github.com/elastic/apm-agent-go/transport/transporttest"
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
		Objectives: prometheus.DefObjectives,
	})
	r.MustRegister(s)

	s.Observe(50)
	s.Observe(100)
	s.Observe(150)

	g := apmprometheus.Wrap(r)
	metrics := gatherMetrics(g)
	assert.Contains(t, metrics[0].Samples, "summary")
	assert.Equal(t, model.Metric{
		Type:  "summary",
		Count: newUint64(3),
		Sum:   newFloat64(300),
		Quantiles: []model.Quantile{
			{Quantile: 0.5, Value: 100},
			{Quantile: 0.9, Value: 150},
			{Quantile: 0.99, Value: 150},
		},
	}, metrics[0].Samples["summary"])
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
	assert.Contains(t, metrics[0].Samples, "go.mem.heap.mallocs")
	metrics = metrics[1:]

	assert.Equal(t, []*model.Metrics{{
		Labels: model.StringMap{
			{Key: "code", Value: "200"},
			{Key: "method", Value: "GET"},
		},
		Samples: map[string]model.Metric{
			"http_requests_total": {
				Type:  "counter",
				Value: newFloat64(123),
			},
			"http_requests_inflight": {
				Type:  "gauge",
				Value: newFloat64(10),
			},
		},
	}, {
		Labels: model.StringMap{
			{Key: "code", Value: "200"},
			{Key: "method", Value: "PUT"},
		},
		Samples: map[string]model.Metric{
			"http_requests_total": {
				Type:  "counter",
				Value: newFloat64(1),
			},
		},
	}, {
		Labels: model.StringMap{
			{Key: "code", Value: "404"},
			{Key: "method", Value: "GET"},
		},
		Samples: map[string]model.Metric{
			"http_requests_total": {
				Type:  "counter",
				Value: newFloat64(1),
			},
		},
	}}, metrics)
}

func gatherMetrics(g elasticapm.MetricsGatherer) []*model.Metrics {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()
	tracer.RegisterMetricsGatherer(g)
	tracer.SendMetrics(nil)
	metrics := transport.Payloads()[0].Metrics()
	for _, s := range metrics {
		s.Timestamp = model.Time{}
	}
	return metrics
}

func newUint64(v uint64) *uint64 {
	return &v
}

func newFloat64(v float64) *float64 {
	return &v
}
