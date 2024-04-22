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

package apmslog_test

import (
	"context"
	"errors"
	"log/slog"

	"go.elastic.co/apm/module/apmslog/v2"
	"go.elastic.co/apm/v2"
)

func ExampleHandler() {
	// Report slog "ERROR" level messages to Elastic APM using
	// apm.DefaultTracer() while utilizing slog.Default().Handler()
	// to format logging messages
	apmHandler := apmslog.NewApmHandler()
	logger := slog.New(apmHandler)

	// while using slog context aware methods, any existing trace,
	// transaction, or span ID are added from the given context
	tx := apm.DefaultTracer().StartTransaction("name", "type")
	defer tx.End()

	ctx := apm.ContextWithTransaction(context.Background(), tx)
	span, ctx := apm.StartSpan(ctx, "name", "type")
	defer span.End()

	// log msg will have a trace, transaction, and a span attached
	logger.InfoContext(ctx, "I should have a trace, transaction, and span id attached!")

	// the log msg will be reported to apm
	logger.ErrorContext(ctx, "I want this to be reported, but have no error to attach")

	// the log msg with its error will be reported to apm
	logger.ErrorContext(ctx, "I will report this error to apm", "error", errors.New("new error"))

	// BOTH errors with the log msg will be reported to apm. [ error, err ] slog attribute keys are by default reported
	logger.ErrorContext(ctx, "I will report this error to apm", "error", errors.New("new error"), "err", errors.New("new err"))
}
