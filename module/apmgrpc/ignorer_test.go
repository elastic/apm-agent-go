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

package apmgrpc_test

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.elastic.co/apm/module/apmgrpc"
)

func TestDefaultServerRequestIgnorer(t *testing.T) {
	s1 := "/helloworld.Greeter/SayHello"
	s2 := "/bar.Foo/World"
	s3 := "/foo?bar=baz"

	testDefaultServerRequestIgnorer(t, "", s1, false)
	testDefaultServerRequestIgnorer(t, "", s2, false)
	testDefaultServerRequestIgnorer(t, "", s3, false)
	testDefaultServerRequestIgnorer(t, ",", s1, false) // equivalent to empty
	
	testDefaultServerRequestIgnorer(t, s1, s1, true)
	testDefaultServerRequestIgnorer(t, "*/helloworld*", s1, true)
	testDefaultServerRequestIgnorer(t, "*/bar*", s2, true)
	testDefaultServerRequestIgnorer(t, "*/Bar*", s2, true)
	testDefaultServerRequestIgnorer(t, "*/foo*", s3, true)
	testDefaultServerRequestIgnorer(t, "*/BAR*", s2, true) // case insensitive by default

	testDefaultServerRequestIgnorer(t, "*/foo?bar=baz", s1, false)
	testDefaultServerRequestIgnorer(t, "*/foo?bar=baz", s2, false)
	testDefaultServerRequestIgnorer(t, "*/foo?bar=baz", s3, true)
}

func testDefaultServerRequestIgnorer(t *testing.T, ignoreURLs string, s string, expect bool) {
	testName := fmt.Sprintf("%s_%s", ignoreURLs, s)
	t.Run(testName, func(t *testing.T) {
		if os.Getenv("_INSIDE_TEST") != "1" {
			cmd := exec.Command(os.Args[0], "-test.run=^"+regexp.QuoteMeta(t.Name())+"$")
			cmd.Env = append(os.Environ(), "_INSIDE_TEST=1")
			cmd.Env = append(cmd.Env, "ELASTIC_APM_IGNORE_URLS="+ignoreURLs)
			assert.NoError(t, cmd.Run())
			return
		}
		ignorer := apmgrpc.DefaultServerRequestIgnorer()
		assert.Equal(t, expect, ignorer(&s))
	})
}
