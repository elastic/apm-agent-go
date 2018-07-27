package elasticapm

import (
	"context"
	"runtime"
	"runtime/debug"
	"time"
)

// builtinMetricsGatherer is an MetricsGatherer which gathers builtin metrics:
//   - memstats (allocations, usage, GC, etc.)
//   - goroutines
//   - tracer stats (number of transactions/errors sent, dropped, etc.)
type builtinMetricsGatherer struct {
	tracer *Tracer
}

// GatherMetrics gathers mem metrics into m.
func (g *builtinMetricsGatherer) GatherMetrics(ctx context.Context, m *Metrics) error {
	m.Add("golang.goroutines", nil, float64(runtime.NumGoroutine()))
	g.gatherMemStatsMetrics(m)
	g.gatherTracerStatsMetrics(m)
	return nil
}

func (g *builtinMetricsGatherer) gatherMemStatsMetrics(m *Metrics) {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	addUint64 := func(name string, v uint64) {
		m.Add(name, nil, float64(v))
	}
	add := func(name string, v float64) {
		m.Add(name, nil, v)
	}

	addUint64("golang.heap.allocations.mallocs", mem.Mallocs)
	addUint64("golang.heap.allocations.frees", mem.Frees)
	addUint64("golang.heap.allocations.objects", mem.HeapObjects)
	addUint64("golang.heap.allocations.total", mem.TotalAlloc)
	addUint64("golang.heap.allocations.allocated", mem.HeapAlloc)
	addUint64("golang.heap.allocations.idle", mem.HeapIdle)
	addUint64("golang.heap.allocations.active", mem.HeapInuse)
	addUint64("golang.heap.system.total", mem.Sys)
	addUint64("golang.heap.system.obtained", mem.HeapSys)
	addUint64("golang.heap.system.stack", mem.StackSys)
	addUint64("golang.heap.system.released", mem.HeapReleased)
	addUint64("golang.heap.gc.next_gc_limit", mem.NextGC)
	addUint64("golang.heap.gc.total_count", uint64(mem.NumGC))
	addUint64("golang.heap.gc.total_pause.ns", mem.PauseTotalNs)
	add("golang.heap.gc.cpu_fraction", mem.GCCPUFraction)

	gcStats := debug.GCStats{
		PauseQuantiles: make([]time.Duration, 5),
	}
	debug.ReadGCStats(&gcStats)
	addUint64("golang.heap.gc.pause.min.ns", uint64(gcStats.PauseQuantiles[0].Nanoseconds()))
	addUint64("golang.heap.gc.pause.max.ns", uint64(gcStats.PauseQuantiles[4].Nanoseconds()))
	addUint64("golang.heap.gc.pause.percentile.25.ns", uint64(gcStats.PauseQuantiles[1].Nanoseconds()))
	addUint64("golang.heap.gc.pause.percentile.50.ns", uint64(gcStats.PauseQuantiles[2].Nanoseconds()))
	addUint64("golang.heap.gc.pause.percentile.75.ns", uint64(gcStats.PauseQuantiles[3].Nanoseconds()))
}

func (g *builtinMetricsGatherer) gatherTracerStatsMetrics(m *Metrics) {
	g.tracer.statsMu.Lock()
	stats := g.tracer.stats
	g.tracer.statsMu.Unlock()

	const p = "agent"
	m.Add(p+".transactions.sent", nil, float64(stats.TransactionsSent))
	m.Add(p+".transactions.dropped", nil, float64(stats.TransactionsDropped))
	m.Add(p+".transactions.send_errors", nil, float64(stats.Errors.SendTransactions))
	m.Add(p+".errors.sent", nil, float64(stats.ErrorsSent))
	m.Add(p+".errors.dropped", nil, float64(stats.ErrorsDropped))
	m.Add(p+".errors.send_errors", nil, float64(stats.Errors.SendErrors))
}
