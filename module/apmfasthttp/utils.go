package apmfasthttp

import (
	"net/http"
	"net/url"

	"github.com/valyala/bytebufferpool"
	"github.com/valyala/fasthttp"
	"go.elastic.co/apm"
	"go.elastic.co/apm/module/apmhttp"
)

func resetHTTPRequest(req *http.Request) {
	req.Method = ""
	req.URL = nil
	req.Proto = ""
	req.ProtoMajor = 0
	req.ProtoMinor = 0
	resetHTTPMap(req.Header)
	req.Body = nil
	req.GetBody = nil
	req.ContentLength = 0
	req.TransferEncoding = req.TransferEncoding[:0]
	req.Close = false
	req.Host = ""
	resetHTTPMap(req.Form)
	resetHTTPMap(req.PostForm)
	req.MultipartForm = nil
	resetHTTPMap(req.Trailer)
	req.RemoteAddr = ""
	req.RequestURI = ""
	req.TLS = nil
	req.Response = nil
}

func resetHTTPMap(m map[string][]string) {
	for k := range m {
		delete(m, k)
	}
}

func requestCtxToRequest(ctx *fasthttp.RequestCtx, req *http.Request, httpBody *netHTTPBody) error {
	body := ctx.Request.Body()

	req.Method = string(ctx.Method())
	req.Proto = "HTTP/1.1"
	req.ProtoMajor = 1
	req.ProtoMinor = 1
	req.RequestURI = string(ctx.RequestURI())
	req.ContentLength = int64(len(body))
	req.Host = string(ctx.Host())
	req.RemoteAddr = ctx.RemoteAddr().String()
	req.TLS = ctx.TLSConnectionState()

	httpBody.b = append(httpBody.b[:0], body...)
	req.Body = httpBody

	req.Header = make(http.Header)
	ctx.Request.Header.VisitAll(func(k, v []byte) {
		sk := string(k)
		sv := string(v)

		switch sk {
		case "Transfer-Encoding":
			req.TransferEncoding = append(req.TransferEncoding, sv)
		default:
			req.Header.Set(sk, sv)
		}
	})

	rURL, err := url.ParseRequestURI(req.RequestURI)
	if err != nil {
		return err
	}

	req.URL = rURL

	return nil
}

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
	b.Write(ctx.URI().Path())

	return b.String()
}
