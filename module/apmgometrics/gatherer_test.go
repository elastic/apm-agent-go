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
				// go-metrics's counters are gauges by our
				// definition, as they are not required to
				// be monotonically increasing.
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

func TestHistogram(t *testing.T) {
	r := metrics.NewRegistry()
	sample := metrics.NewUniformSample(1024)
	hist := metrics.GetOrRegisterHistogram("histogram", r, sample)
	hist.Update(50)
	hist.Update(100)
	hist.Update(150)

	g := apmgometrics.Wrap(r)
	metrics := gatherMetrics(g)

	assert.Contains(t, metrics[0].Samples, "histogram")
	assert.Equal(t, model.Metric{
		Type:   "summary",
		Count:  newUint64(3),
		Sum:    newFloat64(300),
		Min:    newFloat64(50),
		Max:    newFloat64(150),
		Stddev: newFloat64(40.824829046386306),
		Quantiles: []model.Quantile{
			{Quantile: 0.5, Value: 100},
			{Quantile: 0.9, Value: 150},
			{Quantile: 0.99, Value: 150},
		},
	}, metrics[0].Samples["histogram"])
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

func newUint64(v uint64) *uint64 {
	return &v
}

func newFloat64(v float64) *float64 {
	return &v
}
