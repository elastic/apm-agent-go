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
	"time"

	"go.elastic.co/apm/model"
)

const (
	_ int = iota
	compressedStrategyExactMatch
	compressedStrategySameKind
)

type compositeSpan struct {
	// this internal representation should be set in Nanoseconds, although
	// the model unit is set in Milliseconds.
	sum                 int64
	count               int
	compressionStrategy int
	lastSiblingEndTime  time.Time
}

func (cs compositeSpan) build() *model.CompositeSpan {
	var out model.CompositeSpan
	switch cs.compressionStrategy {
	case compressedStrategyExactMatch:
		out.CompressionStrategy = "exact_match"
	case compressedStrategySameKind:
		out.CompressionStrategy = "same_kind"
	}
	out.Count = cs.count
	out.Sum = float64(cs.sum) / float64(time.Millisecond)
	return &out
}

func (cs compositeSpan) empty() bool {
	return cs.count < 1
}

//
// Span //
//

// attemptCompress tries to compress a span into a "composite span" when:
// * Compression is enabled on agent.
// * The buffered span and the incoming span:
//   * Share the same parent (are siblings).
//   * Are consecutive spans.
//   * Are both exit spans, outcome == success and are short enough (See
//     `ELASTIC_APM_SPAN_COMPRESSION_EXACT_MATCH_MAX_DURATION` and
//     `ELASTIC_APM_SPAN_COMPRESSION_SAME_KIND_MAX_DURATION` for more info).
//   * Represent the same exact operation or the same kind of operation:
//     * Are an exact match (same name, kind and destination service).
//       OR
//     * Are the same kind match (same kind and destination service).
//     When a span has already been compressed using a particular strategy, it
//     CANNOT continue to compress spans using a different strategy.
// The compression algorithm is fairly simple and only compresses spans into a
// composite span when the conditions listed above are met for all consecutive
// spans, when a span comes in that doesn't meet any of the conditions, the
// buffer will be flushed (buffered span will be enqueued) and:
// * When the incoming span is compressible, it will replace the buffered span.
// * When the incoming span is not compressible, it will be enqueued as well.
//
// Returns `true` when the span has been buffered, thus the caller should not
// reportSelfTime and enqueue the span. When `false` is returned, the buffer is
// flushed and the called should reportSelfTime and enqueue.
func (s *Span) attemptCompress(compressionEnabled bool) bool {
	if !compressionEnabled {
		return false
	}

	// There are two distinct places where the span can be buffered; the parent
	// span and the transaction (when a transaction is the span's parent).
	if s.parent != nil {
		// Flush the buffer, have the caller report the span.
		if !s.isCompressionEligible() {
			if !s.buffer.empty() {
				s.buffer.flush(nil)
			}
			return false
		}

		// An incoming span (s) is compressable, check if the buffer is empty,
		// if so, store the the event and report the span as compressed.
		if s.parent.buffer.empty() {
			s.parent.buffer.store(s)
			return true
		}

		// When the span is compressable, try to compress it, report back true:
		// On success: store it.
		// On failure: flush and swap the cache with the current span.
		if s.compress(s.parent.buffer.span) {
			s.parent.buffer.store(s)
		} else {
			s.parent.buffer.flush(s)
		}
		return true
	}
	// If the span has not parent and no transaction, it should be reported by
	// the caller.
	if s.tx == nil {
		return false
	}

	if !s.isCompressionEligible() {
		// At this point, the span isn't compressable which is likely to be a
		// parent or non-compressable sibling, either way, the transaction or
		// the span's buffer needs to be flushed and `false` is returned.
		if !s.tx.buffer.empty() {
			s.tx.buffer.flush(nil)
		}
		if !s.buffer.empty() {
			s.buffer.flush(nil)
		}
		return false
	}

	// The span is compressable and we need to store in the parent's cache.
	if s.tx.buffer.empty() {
		s.tx.buffer.store(s)
		return true
	}

	// When the span is compressable, try to compress it, report back true:
	// On success: store it.
	// On failure: flush and swap the cache with the current span.
	if s.tx.buffer.span.compress(s) {
		s.tx.buffer.span.buffer.store(s)
	} else {
		s.tx.buffer.flush(s)
	}
	return true
}

func (s *Span) isCompressionEligible() bool {
	if s == nil {
		return false
	}
	return s.exit && !s.ctxPropagated &&
		(s.Outcome == "" || s.Outcome == "success")
}

// A span is eligible for compression if all the following conditions are met
// 1. It's an exit span
// 2. The trace context has not been propagated to a downstream service
// 3. If the span has outcome (i.e., outcome is present and it's not null) then
//    it should be success. It means spans with outcome indicating an issue of
//    potential interest should not be compressed.
// The second condition is important so that we don't remove (compress) a span
// that may be the parent of a downstream service. This would orphan the sub-
// graph started by the downstream service and cause it to not appear in the
// waterfall view.
func (s *Span) compress(sibling *Span) bool {
	strategy := s.canCompressComposite(sibling)
	if strategy == 0 {
		strategy = s.canCompressStandard(sibling)
	}

	// If the span cannot be compressed using any strategy.
	if strategy == 0 {
		return false
	}

	if s.composite.empty() {
		s.composite = compositeSpan{
			count:               1,
			sum:                 int64(s.Duration),
			compressionStrategy: strategy,
		}
	}

	s.composite.count++
	s.composite.sum += int64(sibling.Duration)
	siblingTimestamp := sibling.timestamp.Add(sibling.Duration)
	if siblingTimestamp.After(s.composite.lastSiblingEndTime) {
		s.composite.lastSiblingEndTime = siblingTimestamp
	}
	return true
}

func (s *Span) canCompressStandard(sibling *Span) int {
	if !s.isSameKind(sibling) {
		return 0
	}

	if s.isExactMatch(sibling) {
		if s.durationLowerOrEq(sibling,
			s.tx.compressedSpans.exactMatchMaxDuration,
		) {
			return compressedStrategyExactMatch
		}
		return 0
	}
	if s.isSameKind(sibling) {
		if s.durationLowerOrEq(sibling,
			s.tx.compressedSpans.sameKindMaxDuration,
		) {
			s.Name = "Calls to " + s.Context.destinationService.Resource
			return compressedStrategyExactMatch
		}
	}
	return 0
}

func (s *Span) canCompressComposite(sibling *Span) int {
	if s.composite.empty() {
		return 0
	}
	switch s.composite.compressionStrategy {
	case compressedStrategyExactMatch:
		if s.isExactMatch(sibling) && s.durationLowerOrEq(sibling,
			s.tx.compressedSpans.exactMatchMaxDuration,
		) {
			return compressedStrategyExactMatch
		}
	case compressedStrategySameKind:
		if s.isSameKind(sibling) && s.durationLowerOrEq(sibling,
			s.tx.compressedSpans.sameKindMaxDuration,
		) {
			return compressedStrategySameKind
		}
	}
	return 0
}

func (s *Span) durationLowerOrEq(sibling *Span, max time.Duration) bool {
	return s.Duration <= max && sibling.Duration <= max
}

// spanBuffer acts as an intermediary buffer that stores compressable spans on
// their parents to compress with other compression eligible siblings.
//
// Not concurrently safe.
type spanBuffer struct {
	span *Span
}

func (b *spanBuffer) empty() bool { return b.span == nil }

func (b *spanBuffer) store(s *Span) { preserveSpanData(s, b) }

// flush enqueues a span to the tracer event queue, but first, it reports the
// parent and tx timers when the breakdown metrics are enabled and when it's
// a compressed span, the duration is adjusted to end when the last sibling
// ended - first span event timestamp.
func (b *spanBuffer) flush(s *Span) {
	// When the span is a composite span, we need to adjust the duration
	// just before it is reported and no more spans will be compressed into
	// the composite. If we did this any time before, the duration of the span
	// would potentially grow over the compressable threshold and result in
	// compressable span not being compressed and reported separately.
	if !b.span.composite.empty() {
		b.span.Duration = b.span.composite.lastSiblingEndTime.Sub(b.span.timestamp)
	}
	if !b.span.tx.ended() && b.span.tx.breakdownMetricsEnabled {
		b.span.reportSelfTimeLockless(b.span.timestamp.Add(b.span.Duration))
	}

	b.span.enqueue()
	b.span = s
	if s != nil {
		preserveSpanData(s, b)
	}
}

func preserveSpanData(from *Span, to *spanBuffer) {
	te := struct {
		*Span
		*SpanData
	}{
		Span:     from,
		SpanData: from.SpanData,
	}
	to.span = te.Span
	to.span.SpanData = te.SpanData
}

//
// SpanData //
//

// isExactMatch is used for compression purposes, two spans are considered an
// exact match if the have the same name and are of the same kind (see
// isSameKind for more details).
func (s *SpanData) isExactMatch(span *Span) bool {
	return s.Name == span.Name && s.isSameKind(span)
}

// isSameKind is used for compression purposes, two spans are considered to be
// of the same kind if they have the same values for type, subtype, and
// `destination.service.resource`.
func (s *SpanData) isSameKind(span *Span) bool {
	sameType := s.Type == span.Type
	sameSubType := s.Subtype == span.Subtype
	dstSrv := s.Context.destination.Service
	otherDstSrv := span.Context.destination.Service
	sameDstSrvRs := dstSrv != nil && otherDstSrv != nil &&
		dstSrv.Resource == otherDstSrv.Resource

	return sameType && sameSubType && sameDstSrvRs
}
