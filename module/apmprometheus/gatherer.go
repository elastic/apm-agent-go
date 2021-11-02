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

package apmprometheus // import "go.elastic.co/apm/module/apmprometheus"

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
		case dto.MetricType_HISTOGRAM:
			// For the bucket values, we follow the approach described by Prometheus's
			// histogram_quantile function (https://prometheus.io/docs/prometheus/latest/querying/functions/#histogram_quantile)
			// to achieve consistent percentile aggregation results:
			//
			// "The histogram_quantile() function interpolates quantile values by assuming a linear
			// distribution within a bucket. (...) If a quantile is located in the highest bucket,
			// the upper bound of the second highest bucket is returned. A lower limit of the lowest
			// bucket is assumed to be 0 if the upper bound of that bucket is greater than 0. In that
			// case, the usual linear interpolation is applied within that bucket. Otherwise, the upper
			// bound of the lowest bucket is returned for quantiles located in the lowest bucket."
			for _, m := range mf.GetMetric() {
				h := m.GetHistogram()
				// Total count for all values in this
				// histogram. We want the per value count.
				totalCount := h.GetSampleCount()
				if totalCount == 0 {
					continue
				}
				labels := makeLabels(m.GetLabel())
				values := h.GetBucket()
				// The +Inf bucket isn't encoded into the
				// protobuf representation, but observations
				// that fall within it are reflected in the
				// histogram's SampleCount.
				// We compare the totalCount to the bucketCount
				// (sum of all CumulativeCount()s per bucket)
				// to infer if an additional midpoint + count
				// need to be added to their respective slices.
				var bucketCount uint64
				valuesLen := len(values)
				midpoints := make([]float64, 0, valuesLen)
				counts := make([]uint64, 0, valuesLen)
				for i, b := range values {
					count := b.GetCumulativeCount()
					le := b.GetUpperBound()
					if i == 0 {
						if le > 0 {
							le /= 2
						}
					} else {
						// apm-server expects non-cumulative
						// counts. prometheus counts each
						// bucket cumulatively, ie. bucketN
						// contains all counts for bucketN and
						// all counts in preceding values. To
						// get the current bucket's count we
						// subtract bucketN-1 from bucketN,
						// when N>0.
						count = count - values[i-1].GetCumulativeCount()
						le = values[i-1].GetUpperBound() + (le-values[i-1].GetUpperBound())/2.0
					}
					// we are excluding zero-count
					// prometheus buckets.
					// the cumulative count may have
					// initially been non-zero, but when we
					// subtract the preceding bucket, it
					// may end up having a zero count.
					if count == 0 {
						continue
					}
					bucketCount += count
					counts = append(counts, count)
					midpoints = append(midpoints, le)
				}
				// Check if there were observations that fell
				// outside of the defined histogram buckets, so
				// we need to modify the current final bucket,
				// and add an additional bucket with these
				// observations.
				if infBucketCount := totalCount - bucketCount; infBucketCount > 0 && valuesLen > 0 {
					// Set the midpoint for the +Inf bucket
					// to be the final defined bucket value.
					midpoints = append(midpoints, values[valuesLen-1].GetUpperBound())
					counts = append(counts, infBucketCount)
				}
				out.AddHistogram(name, labels, midpoints, counts)
			}
		default:
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
