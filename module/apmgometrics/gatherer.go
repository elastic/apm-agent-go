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

package apmgometrics

import (
	"context"

	metrics "github.com/rcrowley/go-metrics"

	"go.elastic.co/apm"
)

// Wrap wraps r, a go-metrics Registry, so that it can be used
// as an apm.MetricsGatherer.
func Wrap(r metrics.Registry) apm.MetricsGatherer {
	return gatherer{r}
}

type gatherer struct {
	r metrics.Registry
}

// GatherMEtrics gathers metrics into m.
func (g gatherer) GatherMetrics(ctx context.Context, m *apm.Metrics) error {
	g.r.Each(func(name string, v interface{}) {
		switch v := v.(type) {
		case metrics.Counter:
			m.Add(name, nil, float64(v.Count()))
		case metrics.Gauge:
			m.Add(name, nil, float64(v.Value()))
		case metrics.GaugeFloat64:
			m.Add(name, nil, v.Value())
		case metrics.Histogram:
			m.Add(name+".count", nil, float64(v.Count()))
			m.Add(name+".total", nil, float64(v.Sum()))
			m.Add(name+".min", nil, float64(v.Min()))
			m.Add(name+".max", nil, float64(v.Max()))
			m.Add(name+".stddev", nil, v.StdDev())
			m.Add(name+".percentile.50", nil, v.Percentile(0.5))
			m.Add(name+".percentile.95", nil, v.Percentile(0.95))
			m.Add(name+".percentile.99", nil, v.Percentile(0.99))
		default:
			// TODO(axw) Meter, Timer, EWMA
		}
	})
	return nil
}
