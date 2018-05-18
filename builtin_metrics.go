package elasticapm

import (
	"context"
	"runtime"
	"time"
)

// builtinMetricsGatherer is an MetricsGatherer which gathers builtin metrics:
//   - memstats (allocations, usage, GC, etc.)
//   - goroutines
type builtinMetricsGatherer struct{}

// GatherMetrics gathers mem metrics into m.
func (g builtinMetricsGatherer) GatherMetrics(ctx context.Context, m *Metrics) error {
	m.AddGauge("go.goroutines", nil, float64(runtime.NumGoroutine()))
	g.gatherMemStatsMetrics(ctx, m)
	return nil
}

func (builtinMetricsGatherer) gatherMemStatsMetrics(ctx context.Context, m *Metrics) {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	// TODO(axw) units?

	addCounterUint64 := func(name string, v uint64) {
		m.AddCounter(name, nil, float64(v))
	}
	addGauge := func(name string, v float64) {
		m.AddGauge(name, nil, v)
	}
	addGaugeUint64 := func(name string, v uint64) {
		addGauge(name, float64(v))
	}

	addCounterUint64("go.heap.mallocs", mem.Mallocs)
	addCounterUint64("go.heap.frees", mem.Frees)
	addCounterUint64("go.heap.alloc_bytes_total", mem.TotalAlloc)
	addGaugeUint64("go.heap.alloc_bytes", mem.HeapAlloc)
	addGaugeUint64("go.heap.alloc_objects", mem.HeapObjects)
	addGaugeUint64("go.heap.inuse_bytes", mem.HeapInuse)
	addGaugeUint64("go.heap.idle_bytes", mem.HeapIdle)

	addGaugeUint64("go.sysmem.bytes", mem.Sys) // XXX name

	addGaugeUint64("go.gc.next_bytes", mem.NextGC)
	addGauge("go.gc.last_sec", time.Duration(mem.LastGC).Seconds())
	addGauge("go.gc.cpu_fraction", mem.GCCPUFraction)
}
