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

package apm_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"
	"text/tabwriter"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.elastic.co/apm"
	"go.elastic.co/apm/model"
	"go.elastic.co/apm/transport/transporttest"
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

	// CPU% should be in the range [0,1], not [0,100].
	cpuTotalNormPct := builtinMetrics.Samples["system.cpu.total.norm.pct"]
	if assert.NotNil(t, gcPct.Value) {
		assert.Condition(t, func() bool {
			return cpuTotalNormPct.Value >= 0 && cpuTotalNormPct.Value <= 1
		}, "value: %v", cpuTotalNormPct.Value)
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

	tracer.RegisterMetricsGatherer(apm.GatherMetricsFunc(
		func(ctx context.Context, m *apm.Metrics) error {
			m.Add("http.request", []apm.MetricLabel{
				{Name: "code", Value: "400"},
				{Name: "path", Value: "/"},
			}, 3)
			m.Add("http.request", []apm.MetricLabel{
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

	g := apm.GatherMetricsFunc(
		func(ctx context.Context, m *apm.Metrics) error {
			m.Add("with_labels", []apm.MetricLabel{
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

func TestTracerMetricsBusyTracer(t *testing.T) {
	os.Setenv("ELASTIC_APM_API_BUFFER_SIZE", "10KB")
	defer os.Unsetenv("ELASTIC_APM_API_BUFFER_SIZE")

	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	firstRequestDone := make(chan struct{})
	tracer.Transport = sendStreamFunc(func(ctx context.Context, r io.Reader) error {
		if firstRequestDone != nil {
			firstRequestDone <- struct{}{}
			firstRequestDone = nil
			return nil
		}
		return transport.SendStream(ctx, r)
	})

	// Force a complete request to be flushed, preventing metrics from
	// being added to the request buffer until we unblock the transport.
	nonblocking := make(chan struct{})
	close(nonblocking)
	tracer.StartTransaction("name", "type").End()
	tracer.Flush(nonblocking)

	const interval = 100 * time.Millisecond
	tracer.SetMetricsInterval(interval)
	for i := 0; i < 5; i++ {
		time.Sleep(interval)
	}
	for i := 0; i < 1000; i++ {
		tx := tracer.StartTransaction(
			strings.Repeat("x", 1024),
			strings.Repeat("y", 1024),
		)
		tx.Context.SetLabel(strings.Repeat("a", 7000), "v")
		tx.End()
	}

	<-firstRequestDone
	tracer.Flush(nil) // wait for possibly-latent flush
	tracer.Flush(nil) // wait for buffered events to be flushed

	assert.NotZero(t, transport.Payloads().Metrics)
}

func TestTracerMetricsBuffered(t *testing.T) {
	var recorder transporttest.RecorderTransport
	unblock := make(chan struct{})
	tracer, _ := apm.NewTracerOptions(apm.TracerOptions{
		Transport: sendStreamFunc(func(ctx context.Context, r io.Reader) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-unblock:
				return recorder.SendStream(ctx, r)
			}
		}),
	})
	defer tracer.Close()

	gathered := make(chan struct{})
	tracer.RegisterMetricsGatherer(apm.GatherMetricsFunc(
		func(ctx context.Context, m *apm.Metrics) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case gathered <- struct{}{}:
			}
			return nil
		},
	))
	tracer.SetMetricsInterval(10 * time.Millisecond)

	// Wait for metrics to be gathered several times, and then unblock
	// the transport and check that the metrics were buffered while
	// the transport was blocked.
	timeout := time.After(5 * time.Second)
	const N = 5
	for i := 0; i < N+1; i++ {
		select {
		case <-timeout:
			t.Fatal("timed out waiting for metrics gatherer to be called")
		case <-gathered:
		}
	}
	unblock <- struct{}{}
	tracer.Flush(nil) // wait for buffered metrics to be flushed

	metrics := recorder.Payloads().Metrics
	if assert.Conditionf(t, func() bool { return len(metrics) >= N }, "len(metrics): %d", len(metrics)) {
		for i, m := range metrics[1:] {
			assert.NotEqual(t, metrics[i].Timestamp, m.Timestamp)
		}
	}
}

func TestTracerMetricsDisable(t *testing.T) {
	os.Setenv("ELASTIC_APM_DISABLE_METRICS", "golang.heap.*, system.memory.*, system.process.*")
	defer os.Unsetenv("ELASTIC_APM_DISABLE_METRICS")

	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	tracer.SendMetrics(nil)

	payloads := transport.Payloads()
	builtinMetrics := payloads.Metrics[0]

	expected := []string{"golang.goroutines", "system.cpu.total.norm.pct"}
	var actual []string
	for name := range builtinMetrics.Samples {
		actual = append(actual, name)
	}
	sort.Strings(actual)
	assert.EqualValues(t, expected, actual)
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

type sendStreamFunc func(context.Context, io.Reader) error

func (f sendStreamFunc) SendStream(ctx context.Context, r io.Reader) error {
	return f(ctx, r)
}
