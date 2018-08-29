package elasticapm_test

import (
	"bytes"
	"context"
	"fmt"
	"runtime"
	"sort"
	"strconv"
	"testing"
	"text/tabwriter"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/apm-agent-go"
	"github.com/elastic/apm-agent-go/model"
	"github.com/elastic/apm-agent-go/transport/transporttest"
)

func TestTracerMetricsBuiltin(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	busyWork(10 * time.Millisecond)
	tracer.SendMetrics(nil)

	payloads := transport.Payloads()
	builtinMetrics := payloads.Metrics[0]

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

		"system.cpu.total.norm.pct",
		"system.memory.total",
		"system.memory.actual.free",
		"system.process.cpu.total.norm.pct",
		"system.process.memory.size",
		"system.process.memory.rss.bytes",

		"agent.send_errors",
		"agent.transactions.sent",
		"agent.transactions.dropped",
		"agent.errors.sent",
		"agent.errors.dropped",
	}
	sort.Strings(expected)
	for name := range builtinMetrics.Samples {
		assert.Contains(t, expected, name)
	}

	var buf bytes.Buffer
	tw := tabwriter.NewWriter(&buf, 10, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "NAME\tVALUE")
	for _, name := range expected {
		assert.Contains(t, builtinMetrics.Samples, name)
		metric := builtinMetrics.Samples[name]
		fmt.Fprintf(tw, "%s\t%s\n", name, strconv.FormatFloat(metric.Value, 'f', -1, 64))
	}
	tw.Flush()
	t.Logf("\n\n%s\n", buf.String())
}

func TestTracerMetricsInterval(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	interval := 1 * time.Second
	tracer.SetMetricsInterval(interval)
	before := time.Now()
	deadline := before.Add(5 * time.Second)
	for len(transport.Payloads().Metrics) == 0 {
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for metrics")
		}
		time.Sleep(time.Millisecond)
	}
	after := time.Now()
	assert.WithinDuration(t, before.Add(interval), after, 200*time.Millisecond)
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
	metrics1 := payloads.Metrics[1]
	metrics2 := payloads.Metrics[2]

	assert.Equal(t, model.StringMap{{Key: "code", Value: "200"}}, metrics1.Labels)
	assert.Equal(t, map[string]model.Metric{"http.request": {Value: 4}}, metrics1.Samples)

	assert.Equal(t, model.StringMap{
		{Key: "code", Value: "400"},
		{Key: "path", Value: "/"},
	}, metrics2.Labels)
	assert.Equal(t, map[string]model.Metric{"http.request": {Value: 3}}, metrics2.Samples)
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
	metrics := payloads.Metrics
	require.Len(t, metrics, 1) // just the builtin/unlabeled metrics
}

// busyWork does meaningless work for the specified duration,
// so we can observe CPU usage.
func busyWork(d time.Duration) int {
	var n int
	afterCh := time.After(d)
	for {
		select {
		case <-afterCh:
			return n
		default:
			n++
		}
	}
}
