---
mapped_pages:
  - https://www.elastic.co/guide/en/apm/agent/go/current/metrics.html
applies_to:
  stack:
  serverless:
    observability:
  product:
    apm_agent_go: ga
products:
  - id: cloud-serverless
  - id: observability
  - id: apm
---

# Metrics [metrics]

The Go agent periodically gathers and reports metrics. Control how often metrics are reported with the [`ELASTIC_APM_METRICS_INTERVAL`](/reference/configuration.md#config-metrics-interval) configuration, and disable metrics with [`ELASTIC_APM_DISABLE_METRICS`](/reference/configuration.md#config-disable-metrics).


## System metrics [metrics-system]

The Go agent reports basic system-level and process-level CPU and memory metrics. For more system metrics, consider installing [Metricbeat](beats://reference/metricbeat/index.md) on your hosts.

As of Elastic Stack version 6.6, these metrics will be visualized in the APM app.

In some cases data from multiple nodes will be combined. As of Elastic Stack version 7.5, you will be able to set a unique name for each node to avoid this problem. Otherwise, data will be aggregated separately based on container ID or host name.

**`system.cpu.total.norm.pct`**
:   type: scaled_float

format: percent

The percentage of CPU time in states other than Idle and IOWait, normalised by the number of cores.


**`system.process.cpu.total.norm.pct`**
:   type: scaled_float

format: percent

The percentage of CPU time spent by the process since the last event. This value is normalized by the number of CPU cores and it ranges from 0 to 100%.


**`system.memory.total`**
:   type: long

format: bytes

Total memory.


**`system.memory.actual.free`**
:   type: long

format: bytes

The actual memory in bytes. It is calculated based on the OS. On Linux it consists of the free memory plus caches and buffers. On OSX it is a sum of free memory and the inactive memory. On Windows, this value does not include memory consumed by system caches and buffers.


**`system.process.memory.size`**
:   type: long

format: bytes

The total virtual memory the process has.



## Go runtime metrics [metrics-golang]

The Go agent reports various Go runtime metrics.

::::{note}
As of now, there are no built-in visualizations for these metrics, so you will need to create custom Kibana dashboards for them.
::::


**`golang.goroutines`**
:   type: long

The number of goroutines that currently exist.


**`golang.heap.allocations.mallocs`**
:   type: long

The number of mallocs.


**`golang.heap.allocations.frees`**
:   type: long

The number of frees.


**`golang.heap.allocations.objects`**
:   type: long

The total number of allocated objects.


**`golang.heap.allocations.total`**
:   type: long

format: bytes

Bytes allocated (even if freed) throughout the lifetime.


**`golang.heap.allocations.allocated`**
:   type: long

format: bytes

Bytes allocated and not yet freed (same as Alloc from [runtime.MemStats](https://golang.org/pkg/runtime/#MemStats)).


**`golang.heap.allocations.idle`**
:   type: long

format: bytes

Bytes in idle spans.


**`golang.heap.allocations.active`**
:   type: long

format: bytes

Bytes in non-idle spans.


**`golang.heap.system.total`**
:   type: long

format: bytes

Total bytes obtained from system (sum of XxxSys from [runtime.MemStats](https://golang.org/pkg/runtime/#MemStats)).


**`golang.heap.system.obtained`**
:   type: long

format: bytes

Via HeapSys from [runtime.MemStats](https://golang.org/pkg/runtime/#MemStats), bytes obtained from system. heap_sys = heap_idle + heap_inuse.


**`golang.heap.system.stack`**
:   type: long

format: bytes

Bytes of stack memory obtained from the OS.


**`golang.heap.system.released`**
:   type: long

format: bytes

Bytes released to the OS.


**`golang.heap.gc.total_pause.ns`**
:   type: long

The total garbage collection duration in nanoseconds.


**`golang.heap.gc.total_count`**
:   type: long

The total number of garbage collections.


**`golang.heap.gc.next_gc_limit`**
:   type: long

format: bytes

Target heap size of the next garbage collection cycle.


**`golang.heap.gc.cpu_fraction`**
:   type: float

Fraction of CPU time used by garbage collection.



## Application Metrics [metrics-application]

**`transaction.duration`**
:   type: simple timer

This timer tracks the duration of transactions and allows for the creation of graphs displaying a weighted average.

Fields:

* `sum.us`: The sum of all transaction durations in microseconds since the last report (the delta)
* `count`: The count of all transactions since the last report (the delta)

You can filter and group by these dimensions:

* `transaction.name`: The name of the transaction
* `transaction.type`: The type of the transaction, for example `request`


**`transaction.breakdown.count`**
:   type: long

format: count (delta)

The number of transactions for which breakdown metrics (`span.self_time`) have been created. As the Go agent tracks the breakdown for both sampled and non-sampled transactions, this metric is equivalent to `transaction.duration.count`

You can filter and group by these dimensions:

* `transaction.name`: The name of the transaction
* `transaction.type`: The type of the transaction, for example `request`


**`span.self_time`**
:   type: simple timer

This timer tracks the span self-times and is the basis of the transaction breakdown visualization.

Fields:

* `sum.us`: The sum of all span self-times in microseconds since the last report (the delta)
* `count`: The count of all span self-times since the last report (the delta)

You can filter and group by these dimensions:

* `transaction.name`: The name of the transaction
* `transaction.type`: The type of the transaction, for example `request`
* `span.type`: The type of the span, for example `app`, `template` or `db`
* `span.subtype`: The sub-type of the span, for example `mysql` (optional)


