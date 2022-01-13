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

package apmtest // import "go.elastic.co/apm/apmtest"

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"sort"
	"text/tabwriter"
	"time"
	"unicode/utf8"

	"go.elastic.co/apm/model"
)

// WriteTraceTable displays the trace as a table which can be used on tests to aid
// debugging.
func WriteTraceTable(writer io.Writer, tx model.Transaction, spans []model.Span) {
	w := tabwriter.NewWriter(writer, 2, 4, 2, ' ', tabwriter.TabIndent)
	fmt.Fprintln(w, "#\tNAME\tTYPE\tCOMP\tN\tDURATION(ms)\tOFFSET\tSPAN ID\tPARENT ID\tTRACE ID")

	fmt.Fprintf(w, "TX\t%s\t%s\t-\t-\t%f\t%d\t%x\t%x\t%x\n", tx.Name,
		tx.Type, tx.Duration,
		0,
		tx.ID, tx.ParentID, tx.TraceID,
	)

	sort.SliceStable(spans, func(i, j int) bool {
		return time.Time(spans[i].Timestamp).Before(time.Time(spans[j].Timestamp))
	})
	for i, span := range spans {
		count := 1
		if span.Composite != nil {
			count = span.Composite.Count
		}

		fmt.Fprintf(w, "%d\t%s\t%s\t%v\t%d\t%f\t+%d\t%x\t%x\t%x\n", i, span.Name,
			span.Type, span.Composite != nil, count, span.Duration,
			time.Time(span.Timestamp).Sub(time.Time(tx.Timestamp))/1e3,
			span.ID, span.ParentID, span.TraceID,
		)
	}
	w.Flush()
}

// WriteTraceWaterfall the trace waterfall "console output" to the specified
// writer sorted by timestamp.
func WriteTraceWaterfall(w io.Writer, tx model.Transaction, spans []model.Span) {
	maxDuration := time.Duration(tx.Duration * float64(time.Millisecond))
	if maxDuration == 0 {
		for _, span := range spans {
			maxDuration += time.Duration(span.Duration * float64(time.Millisecond))
		}
	}

	maxWidth := int64(72)
	buf := new(bytes.Buffer)
	if tx.Duration > 0.0 {
		writeSpan(buf, int(maxWidth), 0, fmt.Sprintf("transaction (%x) - %s", tx.ID, maxDuration.String()))
	}

	sort.SliceStable(spans, func(i, j int) bool {
		return time.Time(spans[i].Timestamp).Before(time.Time(spans[j].Timestamp))
	})

	for _, span := range spans {
		pos := int(math.Round(
			float64(time.Time(span.Timestamp).Sub(time.Time(tx.Timestamp))) /
				float64(maxDuration) * float64(maxWidth),
		))
		tDur := time.Duration(span.Duration * float64(time.Millisecond))
		dur := float64(tDur) / float64(maxDuration)
		width := int(math.Round(dur * float64(maxWidth)))
		if width == int(maxWidth) {
			width = int(maxWidth) - 1
		}

		spancontent := fmt.Sprintf("%s %s - %s",
			span.Type, span.Name,
			time.Duration(span.Duration*float64(time.Millisecond)).String(),
		)
		if span.Composite != nil {
			spancontent = fmt.Sprintf("%d %s - %s",
				span.Composite.Count, span.Name,
				time.Duration(span.Duration*float64(time.Millisecond)).String(),
			)
		}
		writeSpan(buf, width, pos, spancontent)
	}

	io.Copy(w, buf)
}

func writeSpan(buf *bytes.Buffer, width, pos int, content string) {
	spaceRune := ' '
	fillRune := '_'
	startRune := '|'
	endRune := '|'

	// Prevent the spans from going out of bounds.
	if pos == width {
		pos = pos - 2
	} else if pos >= width {
		pos = pos - 1
	}

	for i := 0; i < int(pos); i++ {
		buf.WriteRune(spaceRune)
	}

	if width <= 1 {
		width = 1
		// Write the first letter of the span type when the width is too small.
		startRune, _ = utf8.DecodeRuneInString(content)
	}

	var written int
	written, _ = buf.WriteRune(startRune)
	if len(content) >= int(width)-1 {
		content = content[:int(width)-1]
	}

	spacing := (width - len(content) - 2) / 2
	for i := 0; i < spacing; i++ {
		n, _ := buf.WriteRune(fillRune)
		written += n
	}

	n, _ := buf.WriteString(content)
	written += n
	for i := 0; i < spacing; i++ {
		n, _ := buf.WriteRune(fillRune)
		written += n
	}

	if written < width {
		buf.WriteRune(fillRune)
	}
	if width > 1 {
		buf.WriteRune(endRune)
	}

	buf.WriteString("\n")
}
