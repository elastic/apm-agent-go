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

	"github.com/stretchr/testify/assert"

	"go.elastic.co/apm/stacktrace"
)

func TestLibraryPackage(t *testing.T) {
	assert.True(t, stacktrace.IsLibraryPackage("encoding/json"))
	assert.True(t, stacktrace.IsLibraryPackage("encoding/json/zzz"))
	assert.False(t, stacktrace.IsLibraryPackage("encoding/jsonzzz"))

	stacktrace.RegisterLibraryPackage("encoding/jsonzzz")
	assert.True(t, stacktrace.IsLibraryPackage("encoding/jsonzzz"))
	assert.True(t, stacktrace.IsLibraryPackage("encoding/jsonzzz/yyy"))

	stacktrace.RegisterApplicationPackage("encoding/jsonzzz/yyy")
	assert.True(t, stacktrace.IsLibraryPackage("encoding/jsonzzz"))
	assert.False(t, stacktrace.IsLibraryPackage("encoding/jsonzzz/yyy"))
	assert.False(t, stacktrace.IsLibraryPackage("encoding/jsonzzz/yyy/xxx"))

	assert.True(t, stacktrace.IsLibraryPackage("github.com/elastic/apm-server/vendor/go.elastic.co/apm"))
}
