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

package apmot

import (
	"testing"

	"go.elastic.co/apm"
)

func BenchmarkSetSpanContext(b *testing.B) {
	otSpan := &otSpan{
		span: &apm.Span{
			SpanData: &apm.SpanData{},
		},
		tags: map[string]interface{}{
			"component":    "myComponent",
			"db.instance":  "myDbInstance",
			"db.statement": "myStatement",
			"db.type":      "myDbType",
			"db.user":      "myUser",
			"http.url":     "myHttpUrl",
			"http.method":  "myHttpMethod",
			"type":         "myType",
			"custom1":      "myCustom1",
			"custom2":      "myCustom2",
		},
	}
	for n := 0; n < b.N; n++ {
		otSpan.setSpanContext()
	}
}

func BenchmarkSetTransactionContext(b *testing.B) {
	otSpan := &otSpan{
		ctx: spanContext{
			tx: &apm.Transaction{
				TransactionData: &apm.TransactionData{
					Context: apm.Context{},
				},
			},
		},
		tags: map[string]interface{}{
			"component":        "myComponent",
			"http.method":      "myHttpMethod",
			"http.status_code": 200,
			"http.url":         "myHttpUrl",
			"error":            false,
			"type":             "myType",
			"result":           "myResult",
			"user.id":          "myUserId",
			"user.email":       "myUserEmail",
			"user.username":    "myUserUserName",
			"custom1":          "myCustom1",
			"custom2":          "myCustom2",
		},
	}
	for n := 0; n < b.N; n++ {
		otSpan.setTransactionContext()
	}
}
