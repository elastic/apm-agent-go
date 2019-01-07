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
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"go.elastic.co/apm/model"
	"go.elastic.co/apm/stacktrace"
)

func TestFilesystemContextSetter(t *testing.T) {
	setter := stacktrace.FileSystemContextSetter(http.Dir("./testdata"))
	frame := model.StacktraceFrame{
		AbsolutePath: "/foo.go",
		Line:         5,
	}

	data, err := ioutil.ReadFile("./testdata/foo.go")
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")

	testSetContext(t, setter, frame, 2, 1,
		lines[4],
		lines[2:4],
		lines[5:],
	)
	testSetContext(t, setter, frame, 0, 0, lines[4], []string{}, []string{})
	testSetContext(t, setter, frame, 500, 0, lines[4], lines[:4], []string{})
	testSetContext(t, setter, frame, 0, 500, lines[4], []string{}, lines[5:])
}

func TestFilesystemContextSetterFileNotFound(t *testing.T) {
	setter := stacktrace.FileSystemContextSetter(http.Dir("./testdata"))
	frame := model.StacktraceFrame{
		AbsolutePath: "/foo.go",
		Line:         5,
	}

	data, err := ioutil.ReadFile("./testdata/foo.go")
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")

	testSetContext(t, setter, frame, 2, 1,
		lines[4],
		lines[2:4],
		lines[5:],
	)
	testSetContext(t, setter, frame, 0, 0, lines[4], []string{}, []string{})
	testSetContext(t, setter, frame, 500, 0, lines[4], lines[:4], []string{})
	testSetContext(t, setter, frame, 0, 500, lines[4], []string{}, lines[5:])
}

func testSetContext(
	t *testing.T,
	setter stacktrace.ContextSetter,
	frame model.StacktraceFrame,
	nPre, nPost int,
	expectedContext string, expectedPre, expectedPost []string,
) {
	frames := []model.StacktraceFrame{frame}
	err := stacktrace.SetContext(setter, frames, nPre, nPost)
	if err != nil {
		t.Fatalf("SetContext failed: %s", err)
	}
	if diff := cmp.Diff(frames[0].ContextLine, expectedContext); diff != "" {
		t.Fatalf("ContextLine differs: %s", diff)
	}
	if diff := cmp.Diff(frames[0].PreContext, expectedPre); diff != "" {
		t.Fatalf("PreContext differs: %s", diff)
	}
	if diff := cmp.Diff(frames[0].PostContext, expectedPost); diff != "" {
		t.Fatalf("PostContext differs: %s", diff)
	}
}
