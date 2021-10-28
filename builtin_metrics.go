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

package apm // import "go.elastic.co/apm"

import (
	"context"
	"runtime"

	sysinfo "github.com/elastic/go-sysinfo"
	"github.com/elastic/go-sysinfo/types"
)

// builtinMetricsGatherer is an MetricsGatherer which gathers builtin metrics:
//   - goroutines
//   - memstats (allocations, usage, GC, etc.)
//   - system and process CPU and memory usage
type builtinMetricsGatherer struct {
	tracer         *Tracer
	lastSysMetrics sysMetrics
}

func newBuiltinMetricsGatherer(t *Tracer) *builtinMetricsGatherer {
	g := &builtinMetricsGatherer{tracer: t}
	if metrics, err := gatherSysMetrics(); err == nil {
		g.lastSysMetrics = metrics
	}
	return g
}

// GatherMetrics gathers mem metrics into m.
func (g *builtinMetricsGatherer) GatherMetrics(ctx context.Context, m *Metrics) error {
	m.AddGauge("golang.goroutines", nil, float64(runtime.NumGoroutine()))
	g.gatherSystemMetrics(m)
	g.gatherMemStatsMetrics(m)
	g.tracer.breakdownMetrics.gather(m)
	return nil
}

func (g *builtinMetricsGatherer) gatherSystemMetrics(m *Metrics) {
	metrics, err := gatherSysMetrics()
	if err != nil {
		return
	}
	systemCPU, processCPU := calculateCPUUsage(metrics.cpu, g.lastSysMetrics.cpu)
	m.AddGauge("system.cpu.total.norm.pct", nil, systemCPU)
	m.AddGauge("system.process.cpu.total.norm.pct", nil, processCPU)
	m.AddGauge("system.memory.total", nil, float64(metrics.mem.system.Total))
	m.AddGauge("system.memory.actual.free", nil, float64(metrics.mem.system.Available))
	m.AddGauge("system.process.memory.size", nil, float64(metrics.mem.process.Virtual))
	m.AddGauge("system.process.memory.rss.bytes", nil, float64(metrics.mem.process.Resident))
	g.lastSysMetrics = metrics
}

func (g *builtinMetricsGatherer) gatherMemStatsMetrics(m *Metrics) {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	addUint64Counter := func(name string, v uint64) {
		m.AddCounter(name, nil, float64(v))
	}
	addUint64Gauge := func(name string, v uint64) {
		m.AddGauge(name, nil, float64(v))
	}

	addUint64Counter("golang.heap.allocations.mallocs", mem.Mallocs)
	addUint64Counter("golang.heap.allocations.frees", mem.Frees)
	addUint64Counter("golang.heap.allocations.objects", mem.HeapObjects)
	addUint64Gauge("golang.heap.allocations.total", mem.TotalAlloc)
	addUint64Gauge("golang.heap.allocations.allocated", mem.HeapAlloc)
	addUint64Gauge("golang.heap.allocations.idle", mem.HeapIdle)
	addUint64Gauge("golang.heap.allocations.active", mem.HeapInuse)
	addUint64Gauge("golang.heap.system.total", mem.Sys)
	addUint64Gauge("golang.heap.system.obtained", mem.HeapSys)
	addUint64Gauge("golang.heap.system.stack", mem.StackSys)
	addUint64Counter("golang.heap.system.released", mem.HeapReleased)
	addUint64Gauge("golang.heap.gc.next_gc_limit", mem.NextGC)
	addUint64Counter("golang.heap.gc.total_count", uint64(mem.NumGC))
	addUint64Counter("golang.heap.gc.total_pause.ns", mem.PauseTotalNs)
	m.AddGauge("golang.heap.gc.cpu_fraction", nil, mem.GCCPUFraction)
}

func calculateCPUUsage(current, last cpuMetrics) (systemUsage, processUsage float64) {
	idleDelta := current.system.Idle + current.system.IOWait - last.system.Idle - last.system.IOWait
	systemTotalDelta := current.system.Total() - last.system.Total()
	if systemTotalDelta <= 0 {
		return 0, 0
	}

	idlePercent := float64(idleDelta) / float64(systemTotalDelta)
	systemUsage = 1 - idlePercent

	processTotalDelta := current.process.Total() - last.process.Total()
	processUsage = float64(processTotalDelta) / float64(systemTotalDelta)

	return systemUsage, processUsage
}

type sysMetrics struct {
	cpu cpuMetrics
	mem memoryMetrics
}

type cpuMetrics struct {
	process types.CPUTimes
	system  types.CPUTimes
}

type memoryMetrics struct {
	process types.MemoryInfo
	system  *types.HostMemoryInfo
}

func gatherSysMetrics() (sysMetrics, error) {
	proc, err := sysinfo.Self()
	if err != nil {
		return sysMetrics{}, err
	}
	host, err := sysinfo.Host()
	if err != nil {
		return sysMetrics{}, err
	}
	hostTimes, err := host.CPUTime()
	if err != nil {
		return sysMetrics{}, err
	}
	hostMemory, err := host.Memory()
	if err != nil {
		return sysMetrics{}, err
	}
	procTimes, err := proc.CPUTime()
	if err != nil {
		return sysMetrics{}, err
	}
	procMemory, err := proc.Memory()
	if err != nil {
		return sysMetrics{}, err
	}

	return sysMetrics{
		cpu: cpuMetrics{
			system:  hostTimes,
			process: procTimes,
		},
		mem: memoryMetrics{
			system:  hostMemory,
			process: procMemory,
		},
	}, nil
}
