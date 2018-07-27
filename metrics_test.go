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

	gcPct := builtinMetrics.Samples["golang.heap.gc.cpu_fraction"]
	if assert.NotNil(t, gcPct.Value) && runtime.GOOS == "linux" {
		// NOTE(axw) on Windows and macOS, sometimes
		// MemStats.GCCPUFraction is outside the expected
		// range [0,1). We should isolate the issue and
		// report it upstream.
		assert.Condition(t, func() bool {
			return gcPct.Value >= 0 && gcPct.Value <= 1
		}, "value: %v", gcPct.Value)
	}

	expected := []string{
		"golang.goroutines",
		"golang.heap.allocations.mallocs",
		"golang.heap.allocations.frees",
		"golang.heap.allocations.objects",
		"golang.heap.allocations.total",
		"golang.heap.allocations.allocated",
		"golang.heap.allocations.idle",
		"golang.heap.allocations.active",
		"golang.heap.system.total",
		"golang.heap.system.obtained",
		"golang.heap.system.stack",
		"golang.heap.system.released",
		"golang.heap.gc.next_gc_limit",
		"golang.heap.gc.total_count",
		"golang.heap.gc.total_pause.ns",
		"golang.heap.gc.cpu_fraction",
		"golang.heap.gc.pause.min.ns",
		"golang.heap.gc.pause.max.ns",
		"golang.heap.gc.pause.percentile.25.ns",
		"golang.heap.gc.pause.percentile.50.ns",
		"golang.heap.gc.pause.percentile.75.ns",

		"agent.transactions.sent",
		"agent.transactions.dropped",
		"agent.transactions.send_errors",
		"agent.errors.sent",
		"agent.errors.dropped",
		"agent.errors.send_errors",
	}
	for _, name := range expected {
		assert.Contains(t, builtinMetrics.Samples, name)
	}
	for name := range builtinMetrics.Samples {
		assert.Contains(t, expected, name)
	}
}

func TestTracerMetricsGatherer(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	tracer.RegisterMetricsGatherer(elasticapm.GatherMetricsFunc(
		func(ctx context.Context, m *elasticapm.Metrics) error {
			m.Add("http.request", []elasticapm.MetricLabel{
				{Name: "code", Value: "400"},
				{Name: "path", Value: "/"},
			}, 3)
			m.Add("http.request", []elasticapm.MetricLabel{
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
	assert.Equal(t, map[string]model.Metric{"http.request": {Value: 4}}, metrics[1].Samples)

	assert.Equal(t, model.StringMap{
		{Key: "code", Value: "400"},
		{Key: "path", Value: "/"},
	}, metrics[2].Labels)
	assert.Equal(t, map[string]model.Metric{"http.request": {Value: 3}}, metrics[2].Samples)
}

func TestTracerMetricsDeregister(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	g := elasticapm.GatherMetricsFunc(
		func(ctx context.Context, m *elasticapm.Metrics) error {
			m.Add("with_labels", []elasticapm.MetricLabel{
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
