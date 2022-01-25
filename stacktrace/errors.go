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

package stacktrace // import "go.elastic.co/apm/v2/stacktrace"

import (
	"reflect"
	"runtime"
	"unsafe"

	"github.com/pkg/errors"
)

var (
	uintptrType             = reflect.TypeOf(uintptr(0))
	runtimeFrameType        = reflect.TypeOf(runtime.Frame{})
	errorsStackTraceUintptr = uintptrType.ConvertibleTo(reflect.TypeOf(*new(errors.Frame)))
	errorsStackTraceFrame   = reflect.TypeOf(*new(errors.Frame)).ConvertibleTo(runtimeFrameType)
)

// AppendErrorStacktrace appends at most n entries extracted from err
// to frames, and returns the extended slice. If n is negative, then
// all stack frames will be appended.
func AppendErrorStacktrace(frames []Frame, err error, n int) []Frame {
	type internalStackTracer interface {
		StackTrace() []Frame
	}
	type errorsStackTracer interface {
		StackTrace() errors.StackTrace
	}
	type runtimeStackTracer interface {
		StackTrace() *runtime.Frames
	}
	switch stackTracer := err.(type) {
	case internalStackTracer:
		stackTrace := stackTracer.StackTrace()
		if n >= 0 && len(stackTrace) > n {
			stackTrace = stackTrace[:n]
		}
		frames = append(frames, stackTrace...)
	case errorsStackTracer:
		stackTrace := stackTracer.StackTrace()
		frames = appendPkgerrorsStacktrace(frames, stackTrace, n)
	case runtimeStackTracer:
		runtimeFrames := stackTracer.StackTrace()
		count := 0
		for {
			if n >= 0 && count == n {
				break
			}
			frame, more := runtimeFrames.Next()
			frames = append(frames, RuntimeFrame(frame))
			if !more {
				break
			}
			count++
		}
	}
	return frames
}

func appendPkgerrorsStacktrace(frames []Frame, stackTrace errors.StackTrace, n int) []Frame {
	// github.com/pkg/errors 0.8.x and earlier represent
	// stack frames as uintptr; 0.9.0 and later represent
	// them as runtime.Frames.
	//
	// TODO(axw) drop support for older github.com/pkg/errors
	// versions when we release go.elastic.co/apm/v2 v2.0.0.
	if errorsStackTraceUintptr {
		pc := make([]uintptr, len(stackTrace))
		for i, frame := range stackTrace {
			pc[i] = *(*uintptr)(unsafe.Pointer(&frame))
		}
		frames = AppendCallerFrames(frames, pc, n)
	} else if errorsStackTraceFrame {
		if n >= 0 && len(stackTrace) > n {
			stackTrace = stackTrace[:n]
		}
		for _, frame := range stackTrace {
			rf := (*runtime.Frame)(unsafe.Pointer(&frame))
			frames = append(frames, RuntimeFrame(*rf))
		}
	}
	return frames
}
