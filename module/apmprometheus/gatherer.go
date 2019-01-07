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

package apmprometheus

import (
	"context"
	"strconv"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"

	"go.elastic.co/apm"
)

// Wrap returns an apm.MetricsGatherer wrapping g.
func Wrap(g prometheus.Gatherer) apm.MetricsGatherer {
	return gatherer{g}
}

type gatherer struct {
	p prometheus.Gatherer
}

// GatherMetrics gathers metrics from the prometheus.Gatherer p.g,
// and adds them to out.
func (g gatherer) GatherMetrics(ctx context.Context, out *apm.Metrics) error {
	metricFamilies, err := g.p.Gather()
	if err != nil {
		return errors.WithStack(err)
	}
	for _, mf := range metricFamilies {
		name := mf.GetName()
		switch mf.GetType() {
		case dto.MetricType_COUNTER:
			for _, m := range mf.GetMetric() {
				v := m.GetCounter().GetValue()
				out.Add(name, makeLabels(m.GetLabel()), v)
			}
		case dto.MetricType_GAUGE:
			metrics := mf.GetMetric()
			if name == "go_info" && len(metrics) == 1 && metrics[0].GetGauge().GetValue() == 1 {
				// Ignore the "go_info" metric from the
				// built-in GoCollector, as we provide
				// the same information in the payload.
				continue
			}
			for _, m := range metrics {
				v := m.GetGauge().GetValue()
				out.Add(name, makeLabels(m.GetLabel()), v)
			}
		case dto.MetricType_UNTYPED:
			for _, m := range mf.GetMetric() {
				v := m.GetUntyped().GetValue()
				out.Add(name, makeLabels(m.GetLabel()), v)
			}
		case dto.MetricType_SUMMARY:
			for _, m := range mf.GetMetric() {
				s := m.GetSummary()
				labels := makeLabels(m.GetLabel())
				out.Add(name+".count", labels, float64(s.GetSampleCount()))
				out.Add(name+".total", labels, float64(s.GetSampleSum()))
				for _, q := range s.GetQuantile() {
					p := int(q.GetQuantile() * 100)
					out.Add(name+".percentile."+strconv.Itoa(p), labels, q.GetValue())
				}
			}
		default:
			// TODO(axw) MetricType_HISTOGRAM
		}
	}
	return nil
}

func makeLabels(lps []*dto.LabelPair) []apm.MetricLabel {
	labels := make([]apm.MetricLabel, len(lps))
	for i, lp := range lps {
		labels[i] = apm.MetricLabel{Name: lp.GetName(), Value: lp.GetValue()}
	}
	return labels
}
