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

package apmzerolog

import (
	"strconv"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/pkgerrors"

	"go.elastic.co/apm/internal/pkgerrorsutil"
	"go.elastic.co/apm/stacktrace"
)

// MarshalErrorStack marshals the stack trace in err, if err
// was produced (or wrapped) by github.com/pkg/errors.
//
// This is similar to github.com/rs/zerolog/pkgerrors.MarshalStack,
// with the following differences:
//  - the "source" field value may be an absolute path
//  - the "func" field value will be fully qualified
func MarshalErrorStack(err error) interface{} {
	stackTracer, ok := err.(interface {
		StackTrace() errors.StackTrace
	})
	if !ok {
		return nil
	}

	var frames []stacktrace.Frame
	pkgerrorsutil.AppendStacktrace(stackTracer.StackTrace(), &frames, -1)
	if len(frames) == 0 {
		return nil
	}

	out := make([]map[string]interface{}, len(frames))
	for i, frame := range frames {
		out[i] = map[string]interface{}{
			pkgerrors.StackSourceFileName:     frame.File,
			pkgerrors.StackSourceLineName:     strconv.Itoa(frame.Line),
			pkgerrors.StackSourceFunctionName: frame.Function,
		}
	}
	return out
}
