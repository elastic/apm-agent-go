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

//go:build go1.9
// +build go1.9

package apmgrpc_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"

	"go.elastic.co/apm/module/apmgrpc"
)

func TestDefaultServerRequestIgnorer(t *testing.T) {
	s1 := &grpc.UnaryServerInfo{FullMethod: "/helloworld.Greeter/SayHello"}
	s2 := &grpc.UnaryServerInfo{FullMethod: "/bar.Foo/World"}
	s3 := &grpc.UnaryServerInfo{FullMethod: "/foo?bar=baz"}

	testDefaultServerRequestIgnorer(t, "", s1, true) // equivalent to *
	testDefaultServerRequestIgnorer(t, "", s2, true)
	testDefaultServerRequestIgnorer(t, "", s3, true)

	testDefaultServerRequestIgnorer(t, s1.FullMethod, s1, true)
	testDefaultServerRequestIgnorer(t, `^*/helloworld*`, s1, true)
	testDefaultServerRequestIgnorer(t, `^*/bar.*`, s2, true)
	testDefaultServerRequestIgnorer(t, `(?i)^*/Bar.*`, s2, true) // case insensitive
	testDefaultServerRequestIgnorer(t, `(?i)^*/foo.*`, s3, true)
	testDefaultServerRequestIgnorer(t, `^*/BAR.*`, s2, false)

	testDefaultServerRequestIgnorer(t, `^*/foo\?bar=baz`, s1, false)
	testDefaultServerRequestIgnorer(t, `^*/foo\?bar=baz`, s2, false)
	testDefaultServerRequestIgnorer(t, `^*/foo\?bar=baz`, s3, true)
}

func testDefaultServerRequestIgnorer(t *testing.T, ignoreURLs string, r *grpc.UnaryServerInfo, expect bool) {
	testName := fmt.Sprintf("%s_%s", ignoreURLs, r.FullMethod)
	t.Run(testName, func(t *testing.T) {
		re := regexp.MustCompile(ignoreURLs)
		ignorer := apmgrpc.NewRegexpRequestIgnorer(re)
		assert.Equal(t, expect, ignorer(r))
	})
}
