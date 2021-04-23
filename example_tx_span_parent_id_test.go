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
	"log"

	"go.elastic.co/apm"
)

func ExampleTransaction_ParentID() {
	var ctx context.Context // request context
	tx := apm.TransactionFromContext(ctx)
	defer tx.End()

	// Use the ParentID in your logs
	log.Println(tx.ParentID())
}

func ExampleSpan_ParentID() {
	var ctx context.Context // request context
	tx := apm.TransactionFromContext(ctx)
	defer tx.End()

	span := apm.SpanFromContext(ctx)
	defer span.End()

	// Use the ParentID in your logs
	log.Println(span.ParentID())
}
