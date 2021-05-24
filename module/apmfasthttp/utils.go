package apmfasthttp

import (
	"net/url"

	"github.com/valyala/bytebufferpool"
	"github.com/valyala/fasthttp"
	"go.elastic.co/apm"
	"go.elastic.co/apm/module/apmhttp"
)

func getRequestTraceparent(ctx *fasthttp.RequestCtx, header string) (apm.TraceContext, bool) {
	if value := ctx.Request.Header.Peek(header); len(value) > 0 {
		if c, err := apmhttp.ParseTraceparentHeader(string(value)); err == nil {
			return c, true
		}
	}

	return apm.TraceContext{}, false
}

// NewDynamicServerRequestIgnorer returns the RequestIgnorer to use in
// handler. The list of wildcard patterns comes from central config
func NewDynamicServerRequestIgnorer(t *apm.Tracer) RequestIgnorerFunc {
	return func(ctx *fasthttp.RequestCtx) bool {
		uri := string(ctx.Request.URI().RequestURI())
		u, _ := url.ParseRequestURI(uri)

		return t.IgnoredTransactionURL(u)
	}
}

// ServerRequestName returns the transaction name for the server request context, ctx.
func ServerRequestName(ctx *fasthttp.RequestCtx) string {
	b := bytebufferpool.Get()
	defer bytebufferpool.Put(b)

	b.Write(ctx.Request.Header.Method())
	b.WriteByte(' ')
	b.Write(ctx.Request.URI().Path())

	return b.String()
}
