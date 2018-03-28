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
				// go-metrics's counters are
				// gauges by our definition.
				Type:  "gauge",
				Value: newFloat64(123),
			},
			"http.requests_inflight": {
				Type:  "gauge",
				Value: newFloat64(10),
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
	for _, m := range metrics {
		m.Timestamp = model.Time{}
	}
	return metrics
}

func newFloat64(v float64) *float64 {
	return &v
}
