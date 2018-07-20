package apmgometrics_test

import (
	"strings"
	"testing"

	"github.com/rcrowley/go-metrics"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/apm-agent-go"
	"github.com/elastic/apm-agent-go/model"
	"github.com/elastic/apm-agent-go/module/apmgometrics"
	"github.com/elastic/apm-agent-go/transport/transporttest"
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

	assert.Equal(t, []*model.Metrics{{
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

func gatherMetrics(g elasticapm.MetricsGatherer) []*model.Metrics {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()
	tracer.RegisterMetricsGatherer(g)
	tracer.SendMetrics(nil)
	metrics := transport.Payloads()[0].Metrics()
	for _, m := range metrics {
		m.Timestamp = model.Time{}
	}
	return metrics
}
