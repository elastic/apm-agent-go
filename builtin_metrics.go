package elasticapm

import (
	"context"
	"runtime"
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
	m.AddGauge("go.goroutines", "", nil, float64(runtime.NumGoroutine()))
	g.gatherMemStatsMetrics(m)
	g.gatherTracerStatsMetrics(m)
	return nil
}

func (*builtinMetricsGatherer) gatherMemStatsMetrics(m *Metrics) {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	addCounterUint64 := func(name, unit string, v uint64) {
		m.AddCounter(name, unit, nil, float64(v))
	}
	addGauge := func(name, unit string, v float64) {
		m.AddGauge(name, unit, nil, v)
	}
	addGaugeUint64 := func(name, unit string, v uint64) {
		addGauge(name, unit, float64(v))
	}

	const (
		unitObject = "object"
		unitByte   = "byte"
		unitSecond = "sec"
	)

	addCounterUint64("go.mem.heap.mallocs", "", mem.Mallocs)
	addCounterUint64("go.mem.heap.frees", "", mem.Frees)
	addCounterUint64("go.mem.heap.alloc_total", unitByte, mem.TotalAlloc)
	addGaugeUint64("go.mem.heap.alloc", unitByte, mem.HeapAlloc)
	addGaugeUint64("go.mem.heap.alloc_objects", "", mem.HeapObjects)
	addGaugeUint64("go.mem.heap.inuse", unitByte, mem.HeapInuse)
	addGaugeUint64("go.mem.heap.idle", unitByte, mem.HeapIdle)
	addGaugeUint64("go.mem.sys", unitByte, mem.Sys)
	addGaugeUint64("go.mem.gc.next", unitByte, mem.NextGC)
	addGauge("go.mem.gc.last", unitSecond, time.Duration(mem.LastGC).Seconds())
	addGauge("go.mem.gc.cpu.pct", "", mem.GCCPUFraction*100)
}

func (g *builtinMetricsGatherer) gatherTracerStatsMetrics(m *Metrics) {
	g.tracer.statsMu.Lock()
	stats := g.tracer.stats
	g.tracer.statsMu.Unlock()

	const p = "elasticapm"
	m.AddCounter(p+".transactions.sent", "", nil, float64(stats.TransactionsSent))
	m.AddCounter(p+".transactions.dropped", "", nil, float64(stats.TransactionsDropped))
	m.AddCounter(p+".transactions.send_errors", "", nil, float64(stats.Errors.SendTransactions))
	m.AddCounter(p+".errors.sent", "", nil, float64(stats.ErrorsSent))
	m.AddCounter(p+".errors.dropped", "", nil, float64(stats.ErrorsDropped))
	m.AddCounter(p+".errors.send_errors", "", nil, float64(stats.Errors.SendErrors))
}
