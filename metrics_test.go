package elasticapm_test

import (
	"context"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/apm-agent-go"
	"github.com/elastic/apm-agent-go/model"
	"github.com/elastic/apm-agent-go/transport/transporttest"
)

func TestTracerMetricsBuiltin(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	tracer.SendMetrics(nil)

	payloads := transport.Payloads()
	require.Len(t, payloads, 1)

	metrics := payloads[0].Metrics()
	require.Len(t, metrics, 1)
	builtinMetrics := metrics[0]

	assert.Nil(t, builtinMetrics.Labels)
	assert.NotEmpty(t, builtinMetrics.Timestamp)

	gcPct := builtinMetrics.Samples["go.mem.gc.cpu.pct"]
	if assert.NotNil(t, gcPct.Value) && runtime.GOOS == "linux" {
		// NOTE(axw) on Windows and macOS, sometimes
		// MemStats.GCCPUFraction is outside the expected
		// range [0,1). We should isolate the issue and
		// report it upstream.
		assert.Condition(t, func() bool {
			return *gcPct.Value >= 0 && *gcPct.Value <= 100
		}, "value: %v", *gcPct.Value)
	}

	var dummy float64 = 123
	maybeSetFloat64 := func(ptr *float64, to float64) {
		if ptr != nil {
			*ptr = to
		}
	}
	for _, m := range builtinMetrics.Samples {
		maybeSetFloat64(m.Value, dummy)
		if m.Count != nil {
			*m.Count = uint64(dummy)
		}
		maybeSetFloat64(m.Sum, dummy)
		maybeSetFloat64(m.Min, dummy)
		maybeSetFloat64(m.Max, dummy)
		maybeSetFloat64(m.Stddev, dummy)
		for i := range m.Quantiles {
			m.Quantiles[i].Value = dummy
		}
	}

	counterMetric := func(unit string) model.Metric {
		return model.Metric{Type: "counter", Unit: unit, Value: &dummy}
	}
	gaugeMetric := func(unit string) model.Metric {
		return model.Metric{Type: "gauge", Unit: unit, Value: &dummy}
	}

	gcSummaryMetric := model.Metric{
		Type:  "summary",
		Unit:  "sec",
		Count: newUint64(uint64(dummy)),
		Sum:   newFloat64(dummy),
		Quantiles: []model.Quantile{
			{Quantile: 0, Value: dummy},
			{Quantile: 0.25, Value: dummy},
			{Quantile: 0.5, Value: dummy},
			{Quantile: 0.75, Value: dummy},
			{Quantile: 1, Value: dummy},
		},
	}

	assert.Equal(t, map[string]model.Metric{
		"go.goroutines": gaugeMetric(""),

		"go.mem.heap.mallocs":       counterMetric(""),
		"go.mem.heap.frees":         counterMetric(""),
		"go.mem.heap.alloc":         gaugeMetric("byte"),
		"go.mem.heap.alloc_total":   counterMetric("byte"),
		"go.mem.heap.idle":          gaugeMetric("byte"),
		"go.mem.heap.inuse":         gaugeMetric("byte"),
		"go.mem.heap.alloc_objects": gaugeMetric(""),
		"go.mem.sys":                gaugeMetric("byte"),
		"go.mem.gc.cpu.pct":         gaugeMetric(""),
		"go.mem.gc.last":            gaugeMetric("sec"),
		"go.mem.gc.next":            gaugeMetric("byte"),
		"go.mem.gc.pause":           gcSummaryMetric,

		"elasticapm.transactions.sent":        counterMetric(""),
		"elasticapm.transactions.dropped":     counterMetric(""),
		"elasticapm.transactions.send_errors": counterMetric(""),
		"elasticapm.errors.sent":              counterMetric(""),
		"elasticapm.errors.dropped":           counterMetric(""),
		"elasticapm.errors.send_errors":       counterMetric(""),
	}, builtinMetrics.Samples)
}

func TestTracerMetricsGatherer(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	tracer.RegisterMetricsGatherer(elasticapm.GatherMetricsFunc(
		func(ctx context.Context, m *elasticapm.Metrics) error {
			m.AddCounter("http.request", "", []elasticapm.MetricLabel{
				{Name: "code", Value: "400"},
				{Name: "path", Value: "/"},
			}, 3)
			m.AddCounter("http.request", "", []elasticapm.MetricLabel{
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

	assert.Equal(t, model.StringMap{{Key: "code", Value: "200"}}, metrics[1].Labels)
	assert.Equal(t, map[string]model.Metric{
		"http.request": {
			Type:  "counter",
			Value: newFloat64(4),
		},
	}, metrics[1].Samples)

	assert.Equal(t, model.StringMap{
		{Key: "code", Value: "400"},
		{Key: "path", Value: "/"},
	}, metrics[2].Labels)
	assert.Equal(t, map[string]model.Metric{
		"http.request": {
			Type:  "counter",
			Value: newFloat64(3),
		},
	}, metrics[2].Samples)
}

func TestTracerMetricsDeregister(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	g := elasticapm.GatherMetricsFunc(
		func(ctx context.Context, m *elasticapm.Metrics) error {
			m.AddCounter("with_labels", "", []elasticapm.MetricLabel{
				{Name: "code", Value: "200"},
			}, 4)
			return nil
		},
	)
	deregister := tracer.RegisterMetricsGatherer(g)
	deregister()
	deregister() // safe to call multiple times
	tracer.SendMetrics(nil)

	payloads := transport.Payloads()
	require.Len(t, payloads, 1)

	metrics := payloads[0].Metrics()
	require.Len(t, metrics, 1) // just the builtin/unlabeled metrics
}

func newUint64(v uint64) *uint64 {
	return &v
}

func newFloat64(f float64) *float64 {
	return &f
}
