package elasticapm_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/apm-agent-go"
	"github.com/elastic/apm-agent-go/model"
	"github.com/elastic/apm-agent-go/transport/transporttest"
)

func TestTracerMetricsGatherer(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	tracer.AddMetricsGatherer(elasticapm.GatherMetricsFunc(
		func(ctx context.Context, m *elasticapm.Metrics) error {
			m.AddCounter("http.request", []elasticapm.MetricLabel{
				{Name: "code", Value: "400"},
				{Name: "path", Value: "/"},
			}, 3)
			m.AddCounter("http.request", []elasticapm.MetricLabel{
				{Name: "code", Value: "200"},
			}, 4)
			return nil
		},
	))
	tracer.SendMetrics(nil)

	payloads := transport.Payloads()
	require.Len(t, payloads, 1)

	metrics := payloads[0].Metrics()
	require.Len(t, metrics, 3)

	assert.NotEmpty(t, metrics[0].Timestamp)
	for i := 1; i < len(metrics); i++ {
		assert.Equal(t, metrics[0].Timestamp, metrics[i].Timestamp)
	}
	for i := 0; i < len(metrics); i++ {
		metrics[i].Timestamp = model.Time{}
	}

	builtinMetrics := metrics[0]
	assert.Nil(t, builtinMetrics.Labels)
	assert.Contains(t, builtinMetrics.Samples, "go.goroutines")
	assert.Contains(t, builtinMetrics.Samples, "go.heap.mallocs")
	assert.NotNil(t, builtinMetrics.Samples["go.goroutines"].Value)
	assert.NotNil(t, builtinMetrics.Samples["go.heap.mallocs"].Count)

	assert.Equal(t, &model.Metrics{
		Labels: model.StringMap{{Key: "code", Value: "200"}},
		Samples: map[string]model.Metric{
			"http.request": {Count: newFloat64(4)},
		},
	}, metrics[1])

	assert.Equal(t, &model.Metrics{
		Labels: model.StringMap{
			{Key: "code", Value: "400"},
			{Key: "path", Value: "/"},
		},
		Samples: map[string]model.Metric{
			"http.request": {Count: newFloat64(3)},
		},
	}, metrics[2])
}

func newFloat64(f float64) *float64 {
	return &f
}
