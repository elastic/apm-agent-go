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
	"time"

	"github.com/opentracing/opentracing-go/log"

	"go.elastic.co/apm"
)

func logKV(tracer *apm.Tracer, tx *apm.Transaction, span *apm.Span, time time.Time, keyValues []interface{}) {
	var ctx logContext
	for i := 0; i*2 < len(keyValues); i++ {
		key, ok := keyValues[2*i].(string)
		if !ok {
			continue
		}
		value := keyValues[2*i+1]
		if !ctx.field(key, value) {
			return
		}
	}
	ctx.emit(tracer, tx, span, time)
}

func logFields(tracer *apm.Tracer, tx *apm.Transaction, span *apm.Span, time time.Time, fields []log.Field) {
	var ctx logContext
	for _, field := range fields {
		if !ctx.field(field.Key(), field.Value()) {
			return
		}
	}
	ctx.emit(tracer, tx, span, time)
}

type logContext struct {
	errorEvent bool
	message    string
	err        error
}

// field processes a log field, returning false if the log event should be ignored.
func (c *logContext) field(key string, value interface{}) bool {
	switch key {
	case "event":
		c.errorEvent = value == "error"
		if !c.errorEvent {
			return false
		}
	case "message":
		if v, ok := value.(string); ok {
			c.message = v
		}
	case "error.object", "error":
		// https://opentracing.io/specification/conventions/
		// says that error values should be recorded with the
		// key "error.object", but opentracing-go's log.Error
		// function records them with the key "error". We will
		// handle either, so long as the value is an error.
		if v, ok := value.(error); ok {
			c.err = v
		}
	}
	return true
}

// emit emits an error log record if the context is complete.
func (c *logContext) emit(tracer *apm.Tracer, tx *apm.Transaction, span *apm.Span, time time.Time) {
	if !c.errorEvent {
		return
	}
	if c.message == "" && c.err != nil {
		c.message = c.err.Error()
	}
	e := tracer.NewErrorLog(apm.ErrorLogRecord{Message: c.message, Error: c.err})
	if !time.IsZero() {
		e.Timestamp = time
	}
	if span != nil {
		e.SetSpan(span)
	} else {
		e.SetTransaction(tx)
	}
	e.Send()
}
