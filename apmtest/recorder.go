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

package apmtest

import (
	"go.elastic.co/apm"
	"go.elastic.co/apm/transport/transporttest"
)

// NewRecordingTracer returns a new RecordingTracer, containing a new
// Tracer using the RecorderTransport stored inside.
func NewRecordingTracer() *RecordingTracer {
	var result RecordingTracer
	tracer, err := apm.NewTracerOptions(apm.TracerOptions{
		Transport: &result.RecorderTransport,
	})
	if err != nil {
		panic(err)
	}
	result.Tracer = tracer
	return &result
}

// RecordingTracer holds an apm.Tracer and transporttest.RecorderTransport.
type RecordingTracer struct {
	*apm.Tracer
	transporttest.RecorderTransport
}
