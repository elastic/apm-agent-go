package apmhttp_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/apm-agent-go"
	"github.com/elastic/apm-agent-go/module/apmhttp"
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
	assertParseError("fe-", "traceparent header version 254 is unknown")
	assertParseError("ff-", "traceparent header version 255 is forbidden")

	assertParseError("00-0-0-01", `invalid version 0 traceparent header`)
	assertParseError("00-0af7651916cd43dd8448eb211c80319c~b7ad6b7169203331-01", `invalid version 0 traceparent header`)
	assertParseError("00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331~01", `invalid version 0 traceparent header`)

	out, err := apmhttp.ParseTraceparentHeader("00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01")
	require.NoError(t, err)
	assert.Equal(t, "\x0a\xf7\x65\x19\x16\xcd\x43\xdd\x84\x48\xeb\x21\x1c\x80\x31\x9c", string(out.Trace[:]))
	assert.Equal(t, "\xb7\xad\x6b\x71\x69\x20\x33\x31", string(out.Span[:]))
	assert.Equal(t, elasticapm.TraceOptions(1), out.Options)
	assert.True(t, out.Options.Requested())
	assert.False(t, out.Options.MaybeRecorded())
}
