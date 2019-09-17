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

package apm_test

import (
	"context"
	"fmt"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.elastic.co/apm"
	"go.elastic.co/apm/apmtest"
)

func ExampleTraceFormatter() {
	apmtest.WithTransaction(func(ctx context.Context) {
		span, ctx := apm.StartSpan(ctx, "name", "type")
		defer span.End()

		// The %+v format will add
		//
		//     "trace.id=... transaction.id=... span.id=..."
		//
		// to the log output.
		log.Printf("ERROR [%+v] blah blah", apm.TraceFormatter(ctx))
	})
}

func TestTraceFormatterNothing(t *testing.T) {
	f := apm.TraceFormatter(context.Background())
	for _, format := range []string{
		"%v", "%t", "%x", "%s",
		"%+v", "%+t", "%+x", "%+s",
	} {
		out := fmt.Sprintf(format, f)
		assert.Equal(t, "", out)
	}
}

func TestTraceFormatterTransaction(t *testing.T) {
	var results []string
	tx, _, _ := apmtest.WithTransaction(func(ctx context.Context) {
		f := apm.TraceFormatter(ctx)
		for _, format := range []string{
			"%v", "%t", "%x", "%s",
			"%+v", "%+t", "%+x", "%+s",
		} {
			out := fmt.Sprintf(format, f)
			results = append(results, out)
		}
	})
	assert.Equal(t, []string{
		fmt.Sprintf("%x %x", tx.TraceID, tx.ID),
		fmt.Sprintf("%x", tx.TraceID),
		fmt.Sprintf("%x", tx.ID),
		"",
		fmt.Sprintf("trace.id=%x transaction.id=%x", tx.TraceID, tx.ID),
		fmt.Sprintf("trace.id=%x", tx.TraceID),
		fmt.Sprintf("transaction.id=%x", tx.ID),
		"",
	}, results)
}

func TestTraceFormatterSpan(t *testing.T) {
	var results []string
	tx, spans, _ := apmtest.WithTransaction(func(ctx context.Context) {
		span, ctx := apm.StartSpan(ctx, "name", "type")
		defer span.End()

		f := apm.TraceFormatter(ctx)
		for _, format := range []string{
			"%v", "%t", "%x", "%s",
			"%+v", "%+t", "%+x", "%+s",
		} {
			out := fmt.Sprintf(format, f)
			results = append(results, out)
		}
	})
	span := spans[0]
	assert.Equal(t, []string{
		fmt.Sprintf("%x %x %x", tx.TraceID, tx.ID, span.ID),
		fmt.Sprintf("%x", tx.TraceID),
		fmt.Sprintf("%x", tx.ID),
		fmt.Sprintf("%x", span.ID),
		fmt.Sprintf("trace.id=%x transaction.id=%x span.id=%x", tx.TraceID, tx.ID, span.ID),
		fmt.Sprintf("trace.id=%x", tx.TraceID),
		fmt.Sprintf("transaction.id=%x", tx.ID),
		fmt.Sprintf("span.id=%x", span.ID),
	}, results)
}
