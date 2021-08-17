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

//go:build go1.14
// +build go1.14

package apm

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBreakdownMetricsAlignment(t *testing.T) {
	// This test is to ensure the alignment properties
	// of the breakdownMetricsMapEntry are maintained
	// for both 32-bit and 64-bit systems, since we use
	// sync/atomic operations on them.
	if runtime.GOOS != "darwin" {
		// Go 1.15 dropped support for darwin/386
		t.Run("32-bit", func(t *testing.T) { testBreakdownMetricsAlignment(t, "386") })
	}
	t.Run("64-bit", func(t *testing.T) { testBreakdownMetricsAlignment(t, "amd64") })
}

func testBreakdownMetricsAlignment(t *testing.T, arch string) {
	cfg := types.Config{
		IgnoreFuncBodies: true,
		Importer:         importer.For("source", nil),
		Sizes:            types.SizesFor("gc", arch),
	}

	cmd := exec.Command("go", "list", "-f", "{{.GoFiles}}")
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "GOARCH="+arch)
	output, err := cmd.Output()
	require.NoError(t, err, string(output))
	filenames := strings.Fields(string(output[1 : len(output)-2])) // strip "[" and "]"

	fset := token.NewFileSet()
	files := make([]*ast.File, len(filenames))
	for i, filename := range filenames {
		f, err := parser.ParseFile(fset, filename, nil, 0)
		require.NoError(t, err)
		files[i] = f
	}

	pkg, err := cfg.Check("go.elastic.co/apm", fset, files, nil)
	require.NoError(t, err)

	// breakdownMetricsMapEntry's size must be multiple of 8,
	// as it is used in a slice. This ensures that the embedded
	// fields are always aligned.
	breakdownMetricsMapEntryObj := pkg.Scope().Lookup("breakdownMetricsMapEntry")
	require.NotNil(t, breakdownMetricsMapEntryObj)
	assert.Equal(t, int64(0), cfg.Sizes.Sizeof(breakdownMetricsMapEntryObj.Type())%8)

	// breakdownMetricsMapEntry.breakdownTiming must be the first field,
	// to ensure it remains 64-bit aligned.
	breakdownTimingObj, breakdownTimingFieldIndex, _ := types.LookupFieldOrMethod(
		breakdownMetricsMapEntryObj.Type(), false, pkg, "breakdownTiming",
	)
	require.NotNil(t, breakdownTimingObj)
	assert.Equal(t, []int{0}, breakdownTimingFieldIndex)

	// breakdownTiming.transaction.duration and breakdownTiming.span.duration
	// should be 64-bit aligned. We know that the breakdownTiming type is
	// 64-bit aligned, so check that its transaction/span fields are also,
	// and that spanTiming's duration field is its first field.

	spanTimingObj := pkg.Scope().Lookup("spanTiming")
	require.NotNil(t, spanTimingObj)
	_, durationFieldIndex, _ := types.LookupFieldOrMethod(spanTimingObj.Type(), false, pkg, "duration")
	assert.Equal(t, []int{0}, durationFieldIndex)

	breakdownTimingStruct := breakdownTimingObj.Type().Underlying().(*types.Struct)
	var spanTimingFieldIndices []int
	fields := make([]*types.Var, breakdownTimingStruct.NumFields())
	for i := range fields {
		field := breakdownTimingStruct.Field(i)
		fields[i] = field
		if field.Type() == spanTimingObj.Type() {
			spanTimingFieldIndices = append(spanTimingFieldIndices, i)
		}
	}
	require.NotEmpty(t, spanTimingFieldIndices)
	offsets := cfg.Sizes.Offsetsof(fields)
	for _, fieldIndex := range spanTimingFieldIndices {
		assert.Equal(t, int64(0), offsets[fieldIndex]%8)
	}
}
