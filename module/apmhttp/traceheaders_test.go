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

package apmhttp_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.elastic.co/apm"
	"go.elastic.co/apm/module/apmhttp"
)

func TestParseTraceparentHeader(t *testing.T) {
	assertParseError := func(h, expect string) {
		_, err := apmhttp.ParseTraceparentHeader(h)
		if assert.Error(t, err) {
			assert.Regexp(t, expect, err.Error())
		}
	}
	assertParseError("", `invalid traceparent header ""`)
	assertParseError("00~", `invalid traceparent header "00~"`)
	assertParseError("zz-", `error decoding traceparent header version: encoding/hex: invalid byte:.*`)
	assertParseError("ff-", "traceparent header version 255 is forbidden")

	assertParseError("00-0-0-01", `invalid version 0 traceparent header`)
	assertParseError("00-0af7651916cd43dd8448eb211c80319c~b7ad6b7169203331-01", `invalid version 0 traceparent header`)
	assertParseError("00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331~01", `invalid version 0 traceparent header`)
	assertParseError("00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01-", `invalid version 0 traceparent header`)
	assertParseError("00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-zz", `error decoding trace-options for version 0`)
	assertParseError("00-0af7651916cd43dd8448eb211c80319c-zzzzzzzzzzzzzzzz-zz", `error decoding span-id for version 0`)
	assertParseError("00-zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz-zzzzzzzzzzzzzzzz-zz", `error decoding trace-id for version 0: .*`)

	assertParse := func(h string) (apm.TraceContext, bool) {
		out, err := apmhttp.ParseTraceparentHeader(h)
		return out, assert.NoError(t, err)
	}

	// "If higher version is detected - implementation SHOULD try to parse it."
	//        -- https://w3c.github.io/trace-context/#versioning-of-traceparent
	for _, versionPrefix := range []string{"00", "fe"} {
		if out, ok := assertParse(versionPrefix + "-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01"); ok {
			assert.Equal(t, "\x0a\xf7\x65\x19\x16\xcd\x43\xdd\x84\x48\xeb\x21\x1c\x80\x31\x9c", string(out.Trace[:]))
			assert.Equal(t, "\xb7\xad\x6b\x71\x69\x20\x33\x31", string(out.Span[:]))
			assert.Equal(t, apm.TraceOptions(1), out.Options)
			assert.True(t, out.Options.Recorded())
		}
	}

	// For an unknown version, there may be a trailing string trailing string, but it must start with "-".
	assertParse("fe-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01-foo")
	assertParseError("fe-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01.foo", `invalid version 254 traceparent header`)
}

func TestParseTracestateHeader(t *testing.T) {
	assertParseError := func(h, expect string) {
		_, err := apmhttp.ParseTracestateHeader(h)
		if assert.Error(t, err) {
			assert.Regexp(t, expect, err.Error())
		}
	}

	assertParseError("a", `missing '=' in tracestate entry`)
	assertParseError("a=b, c ", `missing '=' in tracestate entry`)

	assertParse := func(h ...string) (apm.TraceState, bool) {
		out, err := apmhttp.ParseTracestateHeader(h...)
		return out, assert.NoError(t, err)
	}

	tracestate, _ := assertParse("vendorname1=opaqueValue1,vendorname2=opaqueValue2")
	assert.Equal(t, "vendorname1=opaqueValue1,vendorname2=opaqueValue2", tracestate.String())

	tracestate, _ = assertParse("vendorname1=opaqueValue1", "vendorname2=opaqueValue2")
	assert.Equal(t, "vendorname1=opaqueValue1,vendorname2=opaqueValue2", tracestate.String())
}
