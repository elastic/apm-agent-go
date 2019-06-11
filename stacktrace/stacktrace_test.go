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

package stacktrace_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"

	"go.elastic.co/apm/stacktrace"
)

func TestStacktrace(t *testing.T) {
	expect := []string{
		"go.elastic.co/apm/stacktrace_test.callPanickerDefer",
		"runtime.gopanic",
		"go.elastic.co/apm/stacktrace_test.(*panicker).panic",
		"go.elastic.co/apm/stacktrace_test.callPanicker",
	}

	ch := make(chan []string)
	go callPanicker(ch)
	functions := <-ch
	require.NotNil(t, functions)
	if diff := cmp.Diff(expect, functions); diff != "" {
		t.Fatalf("%s", diff)
	}
}

func callPanicker(ch chan<- []string) {
	defer callPanickerDefer(ch)
	(&panicker{}).panic()
}

func callPanickerDefer(ch chan<- []string) {
	if recover() == nil {
		ch <- nil
		return
	}
	allFrames := stacktrace.AppendStacktrace(nil, 1, 5)
	functions := make([]string, 0, len(allFrames))
	for _, frame := range allFrames {
		switch frame.Function {
		case "runtime.call32", "runtime.goexit":
			// Depending on the Go toolchain version, these may or may not be present.
		default:
			functions = append(functions, frame.Function)
		}
	}
	ch <- functions
}

type panicker struct{}

func (*panicker) panic() {
	panic("oh noes")
}

func TestSplitFunctionName(t *testing.T) {
	testSplitFunctionName(t, "main", "main")
	testSplitFunctionName(t, "main", "Foo.Bar")
	testSplitFunctionName(t, "main", "(*Foo).Bar")
	testSplitFunctionName(t, "go.elastic.co/apm/foo", "bar")
	testSplitFunctionName(t,
		"go.elastic.co/apm/module/apmgin",
		"(*middleware).(go.elastic.co/apm/module/apmgin.handle)-fm",
	)
}

func testSplitFunctionName(t *testing.T, module, function string) {
	outModule, outFunction := stacktrace.SplitFunctionName(module + "." + function)
	assertModule(t, outModule, module)
	assertFunction(t, outFunction, function)
}

func TestSplitFunctionNameUnescape(t *testing.T) {
	module, function := stacktrace.SplitFunctionName("github.com/elastic/apm-agent%2ego.funcName")
	assertModule(t, module, "github.com/elastic/apm-agent.go")
	assertFunction(t, function, "funcName")

	// malformed escape sequences are left alone
	module, function = stacktrace.SplitFunctionName("github.com/elastic/apm-agent%.funcName")
	assertModule(t, module, "github.com/elastic/apm-agent%")
	assertFunction(t, function, "funcName")
}

func assertModule(t *testing.T, got, expect string) {
	if got != expect {
		t.Errorf("got module %q, expected %q", got, expect)
	}
}

func assertFunction(t *testing.T, got, expect string) {
	if got != expect {
		t.Errorf("got function %q, expected %q", got, expect)
	}
}

func BenchmarkAppendStacktraceUnlimited(b *testing.B) {
	var frames []stacktrace.Frame
	for i := 0; i < b.N; i++ {
		frames = stacktrace.AppendStacktrace(frames[:0], 0, -1)
	}
}

func BenchmarkAppendStacktrace10(b *testing.B) {
	var frames []stacktrace.Frame
	for i := 0; i < b.N; i++ {
		frames = stacktrace.AppendStacktrace(frames[:0], 0, 10)
	}
}

func BenchmarkAppendStacktrace50(b *testing.B) {
	var frames []stacktrace.Frame
	for i := 0; i < b.N; i++ {
		frames = stacktrace.AppendStacktrace(frames[:0], 0, 50)
	}
}
