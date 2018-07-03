package elasticapm

import (
	"context"
	"sort"
	"strings"
	"sync"

	"github.com/elastic/apm-agent-go/model"
)

// Metrics holds a set of metrics.
type Metrics struct {
	mu      sync.Mutex
	metrics []*model.Metrics
}

func (m *Metrics) reset() {
	m.metrics = m.metrics[:0]
}

// MetricLabel is a name/value pair for labeling metrics.
type MetricLabel struct {
	// Name is the label name.
	Name string

	// Value is the label value.
	Value string
}

// SummaryMetric holds summary statistics for a set of metric values.
type SummaryMetric struct {
	// Count is the count of values.
	Count uint64

	// Sum is the sum of values.
	Sum float64

	// Min holds the minimum value. This is optional, and only
	// reported if non-nil.
	Min *float64

	// Max holds the maximum value. This is optional, and only
	// reported if non-nil.
	Max *float64

	// Stddev holds the standard deviation. This is optional,
	// and only reported if non-nil.
	Stddev *float64

	// Quantile holds the φ-quantiles. This is optional, and
	// only reported if non-empty.
	Quantiles map[float64]float64
}

// MetricsGatherer provides an interface for gathering metrics.
type MetricsGatherer interface {
	// GatherMetrics gathers metrics and adds them to m.
	//
	// If ctx.Done() is signaled, gathering should be aborted and
	// ctx.Err() returned. If GatherMetrics returns an error, it
	// will be logged, but otherwise there is no effect; the
	// implementation must take care not to leave m in an invalid
	// state due to errors.
	GatherMetrics(ctx context.Context, m *Metrics) error
}

// GatherMetricsFunc is a function type implementing MetricsGatherer.
type GatherMetricsFunc func(context.Context, *Metrics) error

// GatherMetrics calls f(ctx, m).
func (f GatherMetricsFunc) GatherMetrics(ctx context.Context, m *Metrics) error {
	return f(ctx, m)
}

// AddCounter adds a counter metric with the given name, optional unit and labels,
// and value. The labels are expected to be sorted lexicographically.
func (m *Metrics) AddCounter(name, unit string, labels []MetricLabel, count float64) {
	m.addMetric(name, labels, model.Metric{
		Type:  "counter",
		Unit:  unit,
		Value: &count,
	})
}

// AddGauge adds a gauge metric with the given name, optional unit and labels,
// and value. The labels are expected to be sorted lexicographically.
func (m *Metrics) AddGauge(name, unit string, labels []MetricLabel, value float64) {
	m.addMetric(name, labels, model.Metric{
		Type:  "gauge",
		Unit:  unit,
		Value: &value,
	})
}

// AddSummary adds a summary metric with the given name, optional unit and labels,
// and values. The labels are expected to be sorted lexicographically.
func (m *Metrics) AddSummary(name, unit string, labels []MetricLabel, summary SummaryMetric) {
	var quantiles []model.Quantile
	if len(summary.Quantiles) > 0 {
		quantiles = make([]model.Quantile, 0, len(summary.Quantiles))
		for q, v := range summary.Quantiles {
			quantiles = append(quantiles, model.Quantile{
				Quantile: q,
				Value:    v,
			})
		}
		sort.Slice(quantiles, func(i, j int) bool {
			return quantiles[i].Quantile < quantiles[j].Quantile
		})
	}
	m.addMetric(name, labels, model.Metric{
		Type:      "summary",
		Unit:      unit,
		Count:     &summary.Count,
		Sum:       &summary.Sum,
		Min:       summary.Min,
		Max:       summary.Max,
		Stddev:    summary.Stddev,
		Quantiles: quantiles,
	})
}

func (m *Metrics) addMetric(name string, labels []MetricLabel, metric model.Metric) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var metrics *model.Metrics
	results := make([]int, len(m.metrics))
	i := sort.Search(len(m.metrics), func(j int) bool {
		results[j] = compareLabels(m.metrics[j].Labels, labels)
		return results[j] >= 0
	})
	if i < len(results) && results[i] == 0 {
		// labels are equal
		metrics = m.metrics[i]
	} else {
		var modelLabels model.StringMap
		if len(labels) > 0 {
			modelLabels = make(model.StringMap, len(labels))
			for i, l := range labels {
				modelLabels[i] = model.StringMapItem{
					Key: l.Name, Value: l.Value,
				}
			}
		}
		metrics = &model.Metrics{
			Labels:  modelLabels,
			Samples: make(map[string]model.Metric),
		}
		if i == len(results) {
			m.metrics = append(m.metrics, metrics)
		} else {
			m.metrics = append(m.metrics, nil)
			copy(m.metrics[i+1:], m.metrics[i:])
			m.metrics[i] = metrics
		}
	}
	metrics.Samples[name] = metric
}

func compareLabels(a model.StringMap, b []MetricLabel) int {
	na, nb := len(a), len(b)
	n := na
	if na > nb {
		n = nb
	}
	for i := 0; i < n; i++ {
		la, lb := a[i], b[i]
		d := strings.Compare(la.Key, lb.Name)
		if d == 0 {
			d = strings.Compare(la.Value, lb.Value)
		}
		if d != 0 {
			return d
		}
	}
	switch {
	case na < nb:
		return -1
	case na > nb:
		return 1
	}
	return 0
}

func gatherMetrics(ctx context.Context, g MetricsGatherer, m *Metrics, logger Logger) {
	defer func() {
		if r := recover(); r != nil {
			if logger != nil {
				logger.Debugf("%T.GatherMetrics panicked: %s", g, r)
			}
		}
	}()
	if err := g.GatherMetrics(ctx, m); err != nil {
		if logger != nil && err != context.Canceled {
			logger.Debugf("%T.GatherMetrics failed: %s", g, err)
		}
	}
}
